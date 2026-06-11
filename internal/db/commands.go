package db

import (
	"database/sql"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jamesMcMeex/waveshell/internal/messages"
	"github.com/jamesMcMeex/waveshell/internal/model"
)

func QueryArtistsCmd(d *sql.DB) tea.Cmd {
	return func() tea.Msg {
		rows, err := d.Query(`SELECT id, name, name_sort FROM artists ORDER BY name_sort`)
		if err != nil {
			return messages.DBErrorMsg{Op: "query artists", Err: err}
		}
		defer func() { _ = rows.Close() }()

		var artists []model.Artist
		for rows.Next() {
			var a model.Artist
			if err := rows.Scan(&a.ID, &a.Name, &a.NameSort); err != nil {
				return messages.DBErrorMsg{Op: "scan artist row", Err: err}
			}
			artists = append(artists, a)
		}
		if err := rows.Err(); err != nil {
			return messages.DBErrorMsg{Op: "iterate artist rows", Err: err}
		}
		return messages.ArtistListResultMsg{Artists: artists}
	}
}

func QueryTagSliceCmd(d *sql.DB, mode model.BrowseMode) tea.Cmd {
	return func() tea.Msg {
		col, err := tagSliceColumn(mode)
		if err != nil {
			return messages.DBErrorMsg{Op: fmt.Sprintf("query %s slice", mode), Err: err}
		}
		q := fmt.Sprintf(`SELECT DISTINCT %s FROM tracks WHERE %s IS NOT NULL ORDER BY %s`, col, col, col)
		rows, err := d.Query(q)
		if err != nil {
			return messages.DBErrorMsg{Op: fmt.Sprintf("query %s slice", mode), Err: err}
		}
		defer func() { _ = rows.Close() }()

		var values []string
		for rows.Next() {
			var v string
			if err := rows.Scan(&v); err != nil {
				return messages.DBErrorMsg{Op: "scan tag slice row", Err: err}
			}
			values = append(values, v)
		}
		if err := rows.Err(); err != nil {
			return messages.DBErrorMsg{Op: "iterate tag slice rows", Err: err}
		}
		return messages.TagSliceResultMsg{Mode: mode, Values: values}
	}
}

func tagSliceColumn(mode model.BrowseMode) (string, error) {
	switch mode {
	case model.BrowseModeLabel:
		return "label", nil
	case model.BrowseModeGenre:
		return "genre", nil
	case model.BrowseModeYear:
		return "year", nil
	default:
		return "", fmt.Errorf("unsupported browse mode for tag slice: %s", mode)
	}
}

func QueryAlbumsForArtistCmd(d *sql.DB, artistID int64) tea.Cmd {
	return func() tea.Msg {
		rows, err := d.Query(`
			SELECT id, title, COALESCE(album_artist,''), year, track_count
			FROM albums WHERE artist_id = ? ORDER BY year, title`, artistID)
		if err != nil {
			return messages.DBErrorMsg{Op: "query albums for artist", Err: err}
		}
		defer func() { _ = rows.Close() }()

		albums := scanAlbums(rows)
		return messages.AlbumListResultMsg{Mode: model.BrowseModeArtist, Key: fmt.Sprintf("%d", artistID), Albums: albums}
	}
}

func QueryAlbumsForTagCmd(d *sql.DB, mode model.BrowseMode, key string) tea.Cmd {
	return func() tea.Msg {
		col, err := tagSliceColumn(mode)
		if err != nil {
			return messages.DBErrorMsg{Op: "query albums for tag", Err: err}
		}
		q := fmt.Sprintf(`
			SELECT DISTINCT a.id, a.title, COALESCE(a.album_artist,''), a.year, a.track_count
			FROM albums a
			JOIN tracks t ON t.album_id = a.id
			WHERE t.%s = ?
			ORDER BY a.year, a.title`, col)
		rows, err := d.Query(q, key)
		if err != nil {
			return messages.DBErrorMsg{Op: fmt.Sprintf("query albums for %s=%s", col, key), Err: err}
		}
		defer func() { _ = rows.Close() }()

		albums := scanAlbums(rows)
		return messages.AlbumListResultMsg{Mode: mode, Key: key, Albums: albums}
	}
}

func scanAlbums(rows *sql.Rows) []model.Album {
	var albums []model.Album
	for rows.Next() {
		var a model.Album
		var year sql.NullInt64
		if err := rows.Scan(&a.ID, &a.Title, &a.AlbumArtist, &year, &a.TrackCount); err != nil {
			continue
		}
		if year.Valid {
			a.Year = int(year.Int64)
		}
		albums = append(albums, a)
	}
	return albums
}

func QueryTracksCmd(d *sql.DB, albumID int64) tea.Cmd {
	return func() tea.Msg {
		rows, err := d.Query(`
			SELECT id, title, track_number, duration_ms, format, sample_rate, bit_depth, bitrate
			FROM tracks WHERE album_id = ? ORDER BY disc_number, track_number`, albumID)
		if err != nil {
			return messages.DBErrorMsg{Op: "query tracks", Err: err}
		}
		defer func() { _ = rows.Close() }()

		var tracks []model.Track
		for rows.Next() {
			var t model.Track
			var tn, bd sql.NullInt64
			if err := rows.Scan(&t.ID, &t.Title, &tn, &t.DurationMs, &t.Format, &t.SampleRate, &bd, &t.Bitrate); err != nil {
				continue
			}
			if tn.Valid {
				t.TrackNumber = int(tn.Int64)
			}
			if bd.Valid {
				v := int(bd.Int64)
				t.BitDepth = &v
			}
			tracks = append(tracks, t)
		}
		return messages.TrackListResultMsg{AlbumID: albumID, Tracks: tracks}
	}
}
