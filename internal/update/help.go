package update

import tea "github.com/charmbracelet/bubbletea"

func initHelpOverlay(m *Model) {
	m.Help.Active = true
	m.Help.ScrollOffset = 0
}

func updateHelpScroll(m *Model, msg tea.Msg) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return
	}
	switch km.String() {
	case "i", "up":
		if m.Help.ScrollOffset > 0 {
			m.Help.ScrollOffset--
		}
	case "k", "down":
		m.Help.ScrollOffset++
	}
}
