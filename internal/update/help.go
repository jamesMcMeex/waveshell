package update

import tea "github.com/charmbracelet/bubbletea"

func initHelpOverlay(m *Model) {
	m.Help.Active = true
}

func updateHelpScroll(m *Model, msg tea.Msg) {
	_ = msg
}
