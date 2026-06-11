package update

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jamesMcMeex/waveshell/internal/model"
)

func (m Model) View() string {
	if m.UI.Width == 0 {
		slog.Debug("view: waiting for window size")
		return "Loading waveshell...\n"
	}

	base := renderBase(m)

	if m.UI.ActiveOverlay == OverlayHelp {
		base = renderHelpOverlay(m, base)
	}

	if m.UI.ShowBrowsePicker {
		base = renderBrowsePicker(m, base)
	}

	return base
}

func renderBase(m Model) string {
	theme := ResolveTheme(m.Config)

	// Account for 2 border chars per pane (left + right) = 6 total for 3 panes
	borderChars := 6
	leftWidth := (m.UI.Width - borderChars) * 22 / 100
	middleWidth := (m.UI.Width - borderChars) * 28 / 100
	tracksWidth := (m.UI.Width - borderChars) - leftWidth - middleWidth

	if leftWidth < 12 {
		leftWidth = 12
	}
	if middleWidth < 14 {
		middleWidth = 14
	}
	if tracksWidth < 22 {
		tracksWidth = 22
	}

	contentLines := paneHeight(m)

	left := renderLeftPane(m, theme, leftWidth, contentLines)
	middle := renderMiddlePane(m, theme, middleWidth, contentLines)
	tracks := renderTracksPane(m, theme, tracksWidth, contentLines)

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, middle, tracks)

	statusBar := renderStatusBar(m, theme)
	hints := renderKeyHints(m, theme)

	return lipgloss.JoinVertical(lipgloss.Left, panes, statusBar, hints)
}

func renderLeftPane(m Model, th Theme, width int, height int) string {
	title := leftPaneLabel(m)
	items := leftPaneItems(m)
	inner := width - 2

	var lines []string
	start := m.UI.LeftOffset
	for i := start; i < start+height && i < len(items); i++ {
		line := items[i]
		if len(line) > inner {
			line = line[:inner]
		}
		if i == m.UI.LeftCursor {
			lines = append(lines, th.CursorLine().Render(fmt.Sprintf("▶ %-*s", inner-2, line)))
		} else {
			lines = append(lines, th.NormalLine().Render(fmt.Sprintf("  %-*s", inner-2, line)))
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", inner))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	titleLine := th.PaneTitle().Render(fmt.Sprintf(" %-*s ", inner-2, title))
	paddedTitle := fmt.Sprintf("%-*s", width, titleLine)
	border := th.PaneBorder(m.UI.ActivePane == model.PaneLeft)
	return lipgloss.JoinVertical(lipgloss.Left, paddedTitle, border.Width(width).Render(content))
}

func renderMiddlePane(m Model, th Theme, width int, height int) string {
	title := middlePaneLabel()
	items := middlePaneItems(m)
	inner := width - 2

	var lines []string
	start := m.UI.MiddleOffset
	for i := start; i < start+height && i < len(items); i++ {
		line := items[i]
		if len(line) > inner {
			line = line[:inner]
		}
		if i == m.UI.MiddleCursor {
			lines = append(lines, th.CursorLine().Render(fmt.Sprintf("▶ %-*s", inner-2, line)))
		} else {
			lines = append(lines, th.NormalLine().Render(fmt.Sprintf("  %-*s", inner-2, line)))
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", inner))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	titleLine := th.PaneTitle().Render(fmt.Sprintf(" %-*s ", inner-2, title))
	paddedTitle := fmt.Sprintf("%-*s", width, titleLine)
	border := th.PaneBorder(m.UI.ActivePane == model.PaneMiddle)
	return lipgloss.JoinVertical(lipgloss.Left, paddedTitle, border.Width(width).Render(content))
}

func renderTracksPane(m Model, th Theme, width int, height int) string {
	cols := visibleColumns(m)
	inner := width - 2
	colWidths := columnWidths(m, cols, inner)
	header := renderColumnHeader(cols, colWidths, th)

	paddedHeader := fmt.Sprintf("%-*s", width, header)

	tracks := trackList(m)
	var lines []string
	start := m.UI.RightOffset
	for i := start; i < start+height && i < len(tracks); i++ {
		row := renderTrackRow(tracks[i], cols, colWidths)
		if len(row) > inner {
			row = row[:inner]
		}
		line := fmt.Sprintf(" %-*s", inner-1, row)
		if i == m.UI.RightCursor {
			lines = append(lines, th.CursorLine().Render(line))
		} else {
			lines = append(lines, th.NormalLine().Render(line))
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", inner))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	border := th.PaneBorder(m.UI.ActivePane == model.PaneRight)
	return lipgloss.JoinVertical(lipgloss.Left, paddedHeader, border.Width(width).Render(content))
}

func renderColumnHeader(cols []string, colWidths map[string]int, th Theme) string {
	labels := map[string]string{
		"track_number": "#",
		"title":        "TITLE",
		"duration":     "DUR",
		"format":       "FORMAT",
		"sample_rate":  "RATE",
		"bit_depth":    "DEPTH",
		"bitrate":      "KBPS",
		"artist":       "ARTIST",
		"album":        "ALBUM",
		"year":         "YEAR",
		"genre":        "GENRE",
		"label":        "LABEL",
		"grouping":     "GROUP",
	}
	var parts []string
	for _, col := range cols {
		label := labels[col]
		if label == "" {
			label = col
		}
		w := colWidths[col]
		if len(label) > w {
			label = label[:w]
		}
		parts = append(parts, fmt.Sprintf("%-*s", w, label))
	}
	return th.PaneTitle().Render(fmt.Sprintf(" %-*s", totalWidth(cols, colWidths), strings.Join(parts, " ")))
}

func totalWidth(cols []string, colWidths map[string]int) int {
	total := 0
	for _, col := range cols {
		total += colWidths[col]
	}
	return total + len(cols) - 1
}

func visibleColumns(m Model) []string {
	browseMode := string(m.UI.BrowseMode)
	if m.Config != nil {
		if cc, ok := m.Config.UI.Columns[browseMode]; ok && len(cc.Visible) > 0 {
			return cc.Visible
		}
	}
	return defaultColumns(browseMode)
}

func defaultColumns(mode string) []string {
	switch mode {
	case string(model.BrowseModeArtist):
		return []string{"track_number", "title", "format", "sample_rate", "bit_depth", "duration"}
	case string(model.BrowseModeLabel):
		return []string{"track_number", "title", "artist", "format", "duration"}
	case string(model.BrowseModeGenre):
		return []string{"track_number", "title", "artist", "format", "duration"}
	case string(model.BrowseModeYear):
		return []string{"track_number", "title", "artist", "format", "duration"}
	default:
		return []string{"track_number", "title", "format", "duration"}
	}
}

func columnWidths(m Model, cols []string, innerWidth int) map[string]int {
	widths := make(map[string]int)
	fixedWidths := map[string]int{
		"track_number": 3,
		"duration":     6,
		"format":       16,
		"sample_rate":  6,
		"bit_depth":    5,
		"bitrate":      5,
		"year":         5,
		"label":        12,
		"grouping":     8,
		"genre":        10,
		"artist":       12,
		"album":        12,
	}

	avail := innerWidth - len(cols)

	for _, col := range cols {
		if col == "title" {
			continue
		}
		w, ok := fixedWidths[col]
		if !ok {
			w = 8
		}
		avail -= w
		widths[col] = w
	}
	if avail < 5 {
		avail = 5
	}
	widths["title"] = avail

	return widths
}

func renderStatusBar(m Model, th Theme) string {
	var parts []string

	if m.UI.BrowseMode != model.BrowseModeArtist {
		parts = append(parts, fmt.Sprintf("By %s", browseModeLabel(m.UI.BrowseMode)))
	}

	selectedKey := leftPaneSelectedKey(m)
	if selectedKey != "" {
		parts = append(parts, selectedKey)
	}

	trackCount := len(m.Library.Tracks)
	if trackCount > 0 {
		parts = append(parts, fmt.Sprintf("%d tracks", trackCount))
	} else if len(m.Library.Albums) > 0 {
		parts = append(parts, fmt.Sprintf("%d albums", len(m.Library.Albums)))
	}

	if len(parts) == 0 {
		parts = append(parts, "waveshell")
	}

	return th.StatusBar().Render(strings.Join(parts, " · "))
}

func browseModeLabel(m model.BrowseMode) string {
	switch m {
	case model.BrowseModeLabel:
		return "Label"
	case model.BrowseModeGenre:
		return "Genre"
	case model.BrowseModeYear:
		return "Year"
	default:
		return "Artist"
	}
}

func renderKeyHints(m Model, th Theme) string {
	return th.KeyHintBar().Render("[i/k] navigate  [j/l] pane  [b] mode  [c] theme  [h] help  [q] quit")
}

func renderBrowsePicker(m Model, base string) string {
	th := ResolveTheme(m.Config)
	modes := browseModes()

	var lines []string
	lines = append(lines, th.PaneTitle().Render("BROWSE BY"))
	lines = append(lines, strings.Repeat("─", 24))
	for i, mode := range modes {
		label := browseModeLabel(mode)
		if mode == model.BrowseModeArtist {
			label += " [default]"
		}
		if i == m.UI.BrowsePickerCursor {
			lines = append(lines, th.CursorLine().Render("▶ "+label))
		} else {
			lines = append(lines, th.NormalLine().Render("  "+label))
		}
	}

	picker := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Accent).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	extraHints := th.KeyHintBar().Render("  [i/k] navigate  [enter] select  [esc] cancel")

	return lipgloss.JoinVertical(lipgloss.Left, base, "\n", picker, extraHints)
}

// ── Help overlay ─────────────────────────────────────────────────────

func renderHelpOverlay(m Model, base string) string {
	th := ResolveTheme(m.Config)
	helpBox := buildHelpBox(m, th)
	helpLines := strings.Split(helpBox, "\n")
	baseLines := strings.Split(base, "\n")

	helpHeight := len(helpLines)
	helpWidth := longestLine(helpLines)
	baseHeight := len(baseLines)

	startY := (baseHeight - helpHeight) / 2
	if startY < 0 {
		startY = 0
	}
	startX := (m.UI.Width - helpWidth) / 2
	if startX < 0 {
		startX = 0
	}

	for i, hl := range helpLines {
		lineIdx := startY + i
		if lineIdx >= len(baseLines) {
			break
		}
		original := baseLines[lineIdx]
		if startX >= len(original) {
			baseLines[lineIdx] = original + strings.Repeat(" ", startX-len(original)) + hl
		} else {
			baseLines[lineIdx] = original[:startX] + hl
			if len(original) > startX+len(hl) {
				baseLines[lineIdx] += original[startX+len(hl):]
			}
		}
	}

	return strings.Join(baseLines, "\n")
}

func buildHelpBox(m Model, th Theme) string {
	var b strings.Builder
	header := th.HelpTitle().Render("HELP")
	items := []helpItem{
		{s: "NAVIGATION", header: true},
		{k: "i / ↑", d: "Cursor up"},
		{k: "k / ↓", d: "Cursor down"},
		{k: "j / ←", d: "Focus previous pane"},
		{k: "l / →", d: "Focus next pane"},
		{k: "Tab / Shift+Tab", d: "Cycle pane focus"},
		{k: "t / Home", d: "Jump to top"},
		{k: "g / End", d: "Jump to bottom"},
		{k: "Ctrl+D / Ctrl+U", d: "Scroll half page"},
		{k: "A–Z", d: "Jump to first matching item"},
		{s: "", header: false},
		{s: "LIBRARY", header: true},
		{k: "b", d: "Browse mode picker"},
		{k: "Enter", d: "Select item"},
		{s: "", header: false},
		{s: "APPLICATION", header: true},
		{k: "c", d: "Cycle theme"},
		{k: "h", d: "Help overlay"},
		{k: "q", d: "Quit"},
		{k: "Esc", d: "Dismiss overlay / cancel"},
		{s: "", header: false},
		{s: "COMING SOON", header: true},
		{k: "p", d: "Play / pause"},
		{k: "/", d: "Search"},
		{k: "i", d: "Info panel"},
		{k: "Space", d: "Add to queue"},
	}

	b.WriteString(header + "\n")
	b.WriteString(th.MutedText().Render(strings.Repeat("─", 30)) + "\n")
	for _, item := range items {
		if item.header {
			b.WriteString("\n" + th.HelpTitle().Render(item.s) + "\n")
			continue
		}
		if item.s == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("  %-18s %s", th.HelpKey().Render(item.k), th.HelpContent().Render(item.d)) + "\n")
	}

	content := b.String()
	boxLines := strings.Split(content, "\n")
	boxWidth := longestLine(boxLines)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Accent).
		Width(boxWidth + 2).
		Render(content)
}

type helpItem struct {
	s      string
	k      string
	d      string
	header bool
}

func longestLine(lines []string) int {
	max := 0
	for _, l := range lines {
		if len(l) > max {
			max = len(l)
		}
	}
	return max
}
