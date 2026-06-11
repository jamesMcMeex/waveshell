package scanner

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jamesMcMeex/waveshell/internal/db"
	"github.com/jamesMcMeex/waveshell/internal/messages"
	"github.com/jamesMcMeex/waveshell/internal/model"
)

type scanState struct {
	paths     []string
	db        *sql.DB
	queue     []scanWorkItem
	walkDone  bool
	processed int
	skipped   int
	total     int
	seenPaths map[string]bool
}

type scanWorkItem struct {
	path    string
	modTime int64
	size    int64
}

func StartScanCmd(paths []string, database *sql.DB) tea.Cmd {
	slog.Info("starting scan", "paths", paths)
	state := &scanState{
		paths:     paths,
		db:        database,
		seenPaths: make(map[string]bool),
		total:     -1,
	}
	return continueScanCmd(state)
}

func continueScanCmd(state *scanState) tea.Cmd {
	return func() tea.Msg {
		state.drainWalk()

		if len(state.queue) == 0 {
			return finishScan(state)
		}

		item := state.queue[0]
		state.queue = state.queue[1:]

		tags, err := ScanFile(item.path)
		if err != nil {
			slog.Warn("continueScan: scan file error", "path", item.path, "error", err)
			state.skipped++
			return messages.ScanFileErrorMsg{
				Path:    item.path,
				Err:     err,
				NextCmd: continueScanCmd(state),
			}
		}

		err = persistTrack(state.db, tags, item)
		if err != nil {
			slog.Error("continueScan: persist error", "path", item.path, "error", err)
			state.skipped++
			return messages.ScanFileErrorMsg{
				Path:    item.path,
				Err:     err,
				NextCmd: continueScanCmd(state),
			}
		}

		state.seenPaths[item.path] = true
		state.processed++
		slog.Debug("continueScan: processed", "path", item.path, "processed", state.processed)
		return messages.ScanProgressMsg{
			Processed:   state.processed,
			Total:       state.total,
			CurrentPath: item.path,
			NextCmd:     continueScanCmd(state),
		}
	}
}

func (s *scanState) drainWalk() {
	if s.walkDone {
		return
	}

	for _, root := range s.paths {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				slog.Warn("walk error", "path", path, "error", err)
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !model.IsSupportedExt(path) {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				s.skipped++
				return nil
			}

			needed, err := needsScan(s.db, path, info.ModTime().Unix())
			if err != nil {
				slog.Warn("needsScan failed", "path", path, "error", err)
				s.skipped++
				return nil
			}
			if !needed {
				s.processed++
				s.seenPaths[path] = true
				return nil
			}

			s.queue = append(s.queue, scanWorkItem{
				path:    path,
				modTime: info.ModTime().Unix(),
				size:    info.Size(),
			})
			return nil
		})
	}

	s.walkDone = true
	s.total = len(s.queue) + s.processed
	slog.Debug("walk complete", "total", s.total, "to_scan", len(s.queue))
}

func finishScan(state *scanState) tea.Msg {
	pruneMissing(state.db, state.seenPaths)
	return messages.ScanCompleteMsg{
		Processed: state.processed,
		Skipped:   state.skipped,
	}
}

func needsScan(database *sql.DB, path string, modTime int64) (bool, error) {
	t, err := db.TrackByPath(database, path)
	if err != nil {
		return false, err
	}
	if t == nil {
		return true, nil
	}
	return t.LastModified != modTime, nil
}

func persistTrack(database *sql.DB, tags *model.RawTrackTags, item scanWorkItem) error {
	dir := filepath.Dir(item.path)
	dirParts := strings.Split(dir, string(os.PathSeparator))
	albumDir := dirParts[len(dirParts)-1]

	artistName := tags.Artist
	if tags.AlbumArtist != "" {
		artistName = tags.AlbumArtist
	}
	if artistName == "" {
		artistName = "Unknown Artist"
	}

	nameSort := sortName(artistName)
	artistID, err := db.InsertArtist(database, artistName, nameSort)
	if err != nil {
		return fmt.Errorf("artist: %w", err)
	}

	albumTitle := tags.Album
	if albumTitle == "" {
		albumTitle = albumDir
	}

	albumRecord := model.Album{
		Title:       albumTitle,
		ArtistID:    &artistID,
		AlbumArtist: tags.AlbumArtist,
		Year:        tags.Year,
		Genre:       tags.Genre,
		Label:       tags.Label,
		Grouping:    tags.Grouping,
		TrackCount:  0,
		DiscCount:   1,
	}
	albumID, err := db.InsertAlbum(database, albumRecord)
	if err != nil {
		return fmt.Errorf("album: %w", err)
	}

	t := TrackFromTags(tags, item.path, item.size, item.modTime)
	t.AlbumID = albumID
	t.ArtistID = artistID

	_, err = db.UpsertTrack(database, t, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("track: %w", err)
	}

	return nil
}

func pruneMissing(database *sql.DB, seenPaths map[string]bool) {
	knownPaths, err := db.AllTrackPaths(database)
	if err != nil {
		slog.Warn("prune: cannot list tracks", "error", err)
		return
	}

	for _, p := range knownPaths {
		if !seenPaths[p] {
			slog.Info("pruning missing track", "path", p)
			if err := db.DeleteTrack(database, p); err != nil {
				slog.Warn("prune: delete failed", "path", p, "error", err)
			}
		}
	}

	if _, err := database.Exec(`
		DELETE FROM albums WHERE id NOT IN (SELECT DISTINCT album_id FROM tracks WHERE album_id IS NOT NULL)
	`); err != nil {
		slog.Warn("prune: cleanup albums failed", "error", err)
	}

	if _, err := database.Exec(`
		DELETE FROM artists WHERE id NOT IN (SELECT DISTINCT artist_id FROM tracks WHERE artist_id IS NOT NULL)
	`); err != nil {
		slog.Warn("prune: cleanup artists failed", "error", err)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	return strings.ToUpper(string(r)) + s[size:]
}

func sortName(name string) string {
	lower := strings.ToLower(name)
	for _, prefix := range []string{"the ", "a ", "an "} {
		if strings.HasPrefix(lower, prefix) {
			suffix := name[len(prefix):]
			return suffix + ", " + capitalize(strings.TrimSpace(prefix))
		}
	}
	return name
}
