package update

import (
	"fmt"
	"strings"

	"github.com/jamesMcMeex/waveshell/internal/model"
)

func FormatBadge(t model.Track) string {
	rate := float64(t.SampleRate) / 1000.0
	if t.BitDepth != nil {
		return fmt.Sprintf("%s %.1fk %dbit %dkbps", t.Format, rate, *t.BitDepth, t.Bitrate)
	}
	return fmt.Sprintf("%s %.1fk %dkbps", t.Format, rate, t.Bitrate)
}

func leftPaneLabel(m Model) string {
	switch m.UI.BrowseMode {
	case model.BrowseModeLabel:
		return "LABELS"
	case model.BrowseModeGenre:
		return "GENRES"
	case model.BrowseModeYear:
		return "YEARS"
	default:
		return "ARTISTS"
	}
}

func leftPaneItems(m Model) []string {
	mode := m.UI.BrowseMode
	if mode == "" {
		mode = model.BrowseModeArtist
	}
	switch mode {
	case model.BrowseModeArtist:
		items := make([]string, len(m.Library.Artists))
		for i, a := range m.Library.Artists {
			items[i] = a.Name
		}
		return items
	case model.BrowseModeLabel, model.BrowseModeGenre, model.BrowseModeYear:
		return m.Library.TagSliceValues
	default:
		return nil
	}
}

func leftPaneSelectedKey(m Model) string {
	switch m.UI.BrowseMode {
	case model.BrowseModeArtist:
		if m.UI.LeftCursor >= 0 && m.UI.LeftCursor < len(m.Library.Artists) {
			a := m.Library.Artists[m.UI.LeftCursor]
			return fmt.Sprintf("%d", a.ID)
		}
	case model.BrowseModeLabel, model.BrowseModeGenre, model.BrowseModeYear:
		if m.UI.LeftCursor >= 0 && m.UI.LeftCursor < len(m.Library.TagSliceValues) {
			return m.Library.TagSliceValues[m.UI.LeftCursor]
		}
	}
	return ""
}

func selectedArtistID(m Model) int64 {
	if m.UI.BrowseMode == model.BrowseModeArtist {
		if m.UI.LeftCursor >= 0 && m.UI.LeftCursor < len(m.Library.Artists) {
			return m.Library.Artists[m.UI.LeftCursor].ID
		}
	}
	return 0
}

func selectedAlbumID(m Model) int64 {
	if m.UI.MiddleCursor >= 0 && m.UI.MiddleCursor < len(m.Library.Albums) {
		return m.Library.Albums[m.UI.MiddleCursor].ID
	}
	return 0
}

func middlePaneLabel() string {
	return "ALBUMS"
}

func middlePaneItems(m Model) []string {
	items := make([]string, len(m.Library.Albums))
	for i, a := range m.Library.Albums {
		label := a.Title
		if a.Year > 0 {
			label = fmt.Sprintf("%s (%d)", label, a.Year)
		}
		if a.TrackCount > 0 {
			label = fmt.Sprintf("%s · %d tracks", label, a.TrackCount)
		}
		items[i] = label
	}
	return items
}

func trackList(m Model) []model.Track {
	return m.Library.Tracks
}

func renderTrackRow(t model.Track, columns []string, colWidths map[string]int) string {
	var parts []string
	for _, col := range columns {
		val := trackColumnValue(t, col)
		w := colWidths[col]
		if len(val) > w {
			val = val[:w]
		}
		val = fmt.Sprintf("%-*s", w, val)
		parts = append(parts, val)
	}
	return strings.Join(parts, " ")
}

func trackColumnValue(t model.Track, col string) string {
	switch col {
	case "track_number":
		if t.TrackNumber > 0 {
			return fmt.Sprintf("%02d", t.TrackNumber)
		}
		return "--"
	case "title":
		return t.Title
	case "duration":
		return formatDuration(t.DurationMs)
	case "format":
		return FormatBadge(t)
	case "sample_rate":
		return fmt.Sprintf("%.1fk", float64(t.SampleRate)/1000.0)
	case "bit_depth":
		if t.BitDepth != nil {
			return fmt.Sprintf("%dbit", *t.BitDepth)
		}
		return "—"
	case "bitrate":
		return fmt.Sprintf("%d", t.Bitrate)
	case "artist":
		return t.Artist
	case "album":
		return t.Album
	case "year":
		if t.Year > 0 {
			return fmt.Sprintf("%d", t.Year)
		}
		return "—"
	case "genre":
		return t.Genre
	case "label":
		return t.Label
	case "grouping":
		return t.Grouping
	default:
		return ""
	}
}

func formatDuration(ms int) string {
	secs := ms / 1000
	m := secs / 60
	s := secs % 60
	return fmt.Sprintf("%d:%02d", m, s)
}
