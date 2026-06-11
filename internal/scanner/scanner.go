package scanner

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dhowden/tag"

	"github.com/jamesMcMeex/waveshell/internal/model"
)

type ScanResult struct {
	Path     string
	FileSize int64
	ModTime  int64
	Tags     *model.RawTrackTags
	Err      error
}

type mtimeChecker func(path string, modTime int64) (bool, error)

func ScanFile(path string) (*model.RawTrackTags, error) {
	ext := filepath.Ext(path)
	if !model.IsSupportedExt(path) {
		return nil, fmt.Errorf("unsupported format: %s", ext)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}

	tags := &model.RawTrackTags{
		Format: formatFromExt(ext),
		Codec:  codecFromExt(ext),
	}

	streamInfo, err := inspectStream(f, ext)
	if err == nil {
		tags.SampleRate = streamInfo.SampleRate
		tags.BitDepth = streamInfo.BitDepth
		tags.Bitrate = streamInfo.Bitrate
		tags.Channels = streamInfo.Channels
		tags.DurationMs = streamInfo.DurationMs
	} else {
		avgBitrate := defaultBitrate(ext)
		if avgBitrate > 0 {
			tags.DurationMs = int((info.Size() * 8) / int64(avgBitrate) / 1000)
			tags.Bitrate = avgBitrate
		}
		tags.SampleRate = defaultSampleRate(ext)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek: %w", err)
	}

	m, err := tag.ReadFrom(f)
	if err != nil {
		if tags.Title == "" {
			tags.Title = filepath.Base(path)
		}
		return tags, nil
	}

	tags.Title = m.Title()
	tags.Artist = m.Artist()
	tags.AlbumArtist = m.AlbumArtist()
	tags.Album = m.Album()
	tags.Genre = m.Genre()
	tags.Composer = m.Composer()
	tags.Container = containerFromFileType(m.FileType())

	if y := m.Year(); y != 0 {
		tags.Year = y
	}

	if tn, _ := m.Track(); tn != 0 {
		tags.TrackNumber = tn
	}
	if dn, _ := m.Disc(); dn != 0 {
		tags.DiscNumber = dn
	}

	raw := m.Raw()
	tags.Grouping = extractTag(raw, "grouping", "TIT1", "©grp", "GROUPING")
	tags.Label = extractTag(raw, "label", "TPUB", "publisher", "LABEL", "----:com.apple.iTunes:LABEL")

	if tags.Year == 0 {
		if y, ok := extractIntFromRaw(raw, "year", "TYER", "©day"); ok {
			tags.Year = y
		}
	}

	if tags.TrackNumber == 0 {
		parseTrackDisc(raw, tags)
	}

	pic := m.Picture()
	if pic != nil {
		tags.HasArtwork = true
		tags.ArtworkFormat = pic.Ext
		tags.ArtworkData = pic.Data
	}

	parseReplayGain(raw, tags)

	return tags, nil
}

func WalkAndScan(paths []string, checkMtime mtimeChecker, onResult func(ScanResult)) (int, int) {
	var processed, skipped int

	for _, root := range paths {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
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
				skipped++
				return nil
			}
			modTime := info.ModTime().Unix()

			if checkMtime != nil {
				needed, err := checkMtime(path, modTime)
				if err != nil {
					skipped++
					return nil
				}
				if !needed {
					processed++
					return nil
				}
			}

			tags, err := ScanFile(path)
			if err != nil {
				skipped++
				onResult(ScanResult{Path: path, Err: err})
				return nil
			}

			processed++
			onResult(ScanResult{
				Path:     path,
				FileSize: info.Size(),
				ModTime:  modTime,
				Tags:     tags,
			})
			return nil
		})
		if err != nil {
			slog.Error("walk failed", "path", root, "error", err)
		}
	}

	return processed, skipped
}

func TrackFromTags(tags *model.RawTrackTags, path string, fileSize int64, modTime int64) model.Track {
	t := model.Track{
		FilePath:         path,
		FileSizeBytes:    fileSize,
		LastModified:     modTime,
		Title:            tags.Title,
		Artist:           tags.Artist,
		AlbumArtist:      tags.AlbumArtist,
		Album:            tags.Album,
		TrackNumber:      tags.TrackNumber,
		DiscNumber:       tags.DiscNumber,
		Year:             tags.Year,
		Genre:            tags.Genre,
		Grouping:         tags.Grouping,
		Label:            tags.Label,
		DurationMs:       tags.DurationMs,
		Format:           string(tags.Format),
		Codec:            string(tags.Codec),
		Container:        tags.Container,
		SampleRate:       tags.SampleRate,
		Bitrate:          tags.Bitrate,
		HasArtwork:       tags.HasArtwork,
		ArtworkWidth:     tags.ArtworkWidth,
		ArtworkHeight:    tags.ArtworkHeight,
		ArtworkFormat:    tags.ArtworkFormat,
		ArtworkSizeBytes: int64(len(tags.ArtworkData)),
	}
	if tags.BitDepth != 0 {
		bd := tags.BitDepth
		t.BitDepth = &bd
	}
	return t
}

func formatFromExt(ext string) model.AudioFormat {
	switch ext {
	case ".flac":
		return model.AudioFormatFLAC
	case ".m4a", ".alac":
		return model.AudioFormatALAC
	case ".mp3":
		return model.AudioFormatMP3
	case ".aiff", ".aif":
		return model.AudioFormatAIFF
	case ".wav":
		return model.AudioFormatWAV
	case ".ogg":
		return model.AudioFormatOGG
	}
	return ""
}

func codecFromExt(ext string) model.Codec {
	switch ext {
	case ".flac":
		return model.CodecFLAC
	case ".m4a", ".alac":
		return model.CodecALAC
	case ".mp3":
		return model.CodecMP3
	case ".aiff", ".aif", ".wav":
		return model.CodecPCM
	case ".ogg":
		return model.CodecVorbis
	}
	return ""
}

func containerFromFileType(ft tag.FileType) string {
	switch ft {
	case tag.FLAC:
		return "FLAC"
	case tag.M4A, tag.ALAC:
		return "MP4"
	case tag.MP3:
		return "MP3"
	case tag.OGG:
		return "OGG"
	}
	return ""
}

func defaultBitrate(ext string) int {
	switch ext {
	case ".flac":
		return 800
	case ".mp3":
		return 256
	case ".ogg":
		return 192
	}
	return 0
}

func defaultSampleRate(ext string) int {
	return 44100
}

func extractTag(raw map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := raw[k]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func extractIntFromRaw(raw map[string]interface{}, keys ...string) (int, bool) {
	for _, k := range keys {
		v, ok := raw[k]
		if !ok {
			continue
		}
		switch val := v.(type) {
		case string:
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				return n, true
			}
			if idx := strings.IndexAny(val, "/ "); idx > 0 {
				if n, err := strconv.Atoi(strings.TrimSpace(val[:idx])); err == nil {
					return n, true
				}
			}
		case int:
			return val, true
		case int64:
			return int(val), true
		}
	}
	return 0, false
}

func parseTrackDisc(raw map[string]interface{}, tags *model.RawTrackTags) {
	for _, key := range []string{"tracknumber", "TRACKNUMBER", "track"} {
		if v, ok := raw[key]; ok {
			s := fmt.Sprintf("%v", v)
			if idx := strings.IndexAny(s, "/ "); idx > 0 {
				if n, err := strconv.Atoi(strings.TrimSpace(s[:idx])); err == nil {
					tags.TrackNumber = n
				}
			} else if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				tags.TrackNumber = n
			}
		}
	}

	for _, key := range []string{"discnumber", "DISCNUMBER", "disc"} {
		if v, ok := raw[key]; ok {
			s := fmt.Sprintf("%v", v)
			if idx := strings.IndexAny(s, "/ "); idx > 0 {
				if n, err := strconv.Atoi(strings.TrimSpace(s[:idx])); err == nil {
					tags.DiscNumber = n
				}
			} else if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				tags.DiscNumber = n
			}
		}
	}
}

func parseReplayGain(raw map[string]interface{}, tags *model.RawTrackTags) {
	rgMappings := []struct {
		key  string
		dest *float64
	}{
		{"REPLAYGAIN_TRACK_GAIN", &tags.RGTrackGain},
		{"replaygain_track_gain", &tags.RGTrackGain},
		{"REPLAYGAIN_TRACK_PEAK", &tags.RGTrackPeak},
		{"replaygain_track_peak", &tags.RGTrackPeak},
		{"REPLAYGAIN_ALBUM_GAIN", &tags.RGAlbumGain},
		{"replaygain_album_gain", &tags.RGAlbumGain},
		{"REPLAYGAIN_ALBUM_PEAK", &tags.RGAlbumPeak},
		{"replaygain_album_peak", &tags.RGAlbumPeak},
		{"R128_TRACK_GAIN", &tags.R128TrackGain},
		{"r128_track_gain", &tags.R128TrackGain},
		{"R128_ALBUM_GAIN", &tags.R128AlbumGain},
		{"r128_album_gain", &tags.R128AlbumGain},
	}

	for _, m := range rgMappings {
		if v, ok := raw[m.key]; ok {
			parseFloatToDest(v, m.dest)
		}
	}

	for rawKey, v := range raw {
		if strings.HasPrefix(rawKey, "TXXX") {
			if comm, ok := v.(*tag.Comm); ok {
				desc := strings.ToLower(comm.Description)
				var dest *float64
				switch {
				case strings.Contains(desc, "replaygain_track_gain"):
					dest = &tags.RGTrackGain
				case strings.Contains(desc, "replaygain_track_peak"):
					dest = &tags.RGTrackPeak
				case strings.Contains(desc, "replaygain_album_gain"):
					dest = &tags.RGAlbumGain
				case strings.Contains(desc, "replaygain_album_peak"):
					dest = &tags.RGAlbumPeak
				case strings.Contains(desc, "r128_track_gain"):
					dest = &tags.R128TrackGain
				case strings.Contains(desc, "r128_album_gain"):
					dest = &tags.R128AlbumGain
				case desc == "year":
					if y, err := strconv.Atoi(cleanString(comm.Text)); err == nil && tags.Year == 0 {
						tags.Year = y
					}
				}
				if dest != nil {
					parseFloatToDest(comm.Text, dest)
				}
			}
		}
	}
}

func cleanString(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "\x00")
}

func parseFloatToDest(v interface{}, dest *float64) {
	switch val := v.(type) {
	case string:
		cleaned := cleanString(val)
		cleaned = strings.TrimSuffix(cleaned, "dB")
		cleaned = strings.TrimSpace(cleaned)
		if f, err := strconv.ParseFloat(cleaned, 64); err == nil {
			*dest = f
		}
	case float64:
		*dest = val
	case *tag.Comm:
		cleaned := cleanString(val.Text)
		cleaned = strings.TrimSuffix(cleaned, "dB")
		cleaned = strings.TrimSpace(cleaned)
		if f, err := strconv.ParseFloat(cleaned, 64); err == nil {
			*dest = f
		}
	}
}
