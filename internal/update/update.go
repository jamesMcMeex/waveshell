package update

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jamesMcMeex/waveshell/internal/messages"
	"github.com/jamesMcMeex/waveshell/internal/scanner"
)

func InitialModel() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	if m.Config != nil && m.Config.Library.ScanOnStartup && len(m.Config.Library.Paths) > 0 {
		return func() tea.Msg {
			return messages.ScanStartedMsg{}
		}
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.ScanStartedMsg:
		if m.Library.Scanning {
			return m, nil
		}
		m.Library.Scanning = true
		m.Library.ScanComplete = false
		m.Library.ScanProcessed = 0
		m.Library.ScanSkipped = 0
		m.Library.ScanTotal = -1
		return m, scanner.StartScanCmd(m.Config.Library.Paths, m.DB)

	case messages.ScanProgressMsg:
		m.Library.Scanning = true
		m.Library.ScanProcessed = msg.Processed
		m.Library.ScanTotal = msg.Total
		m.Library.ScanCurrent = msg.CurrentPath
		return m, msg.NextCmd

	case messages.ScanFileErrorMsg:
		m.Library.ScanSkipped++
		return m, msg.NextCmd

	case messages.ScanCompleteMsg:
		m.Library.Scanning = false
		m.Library.ScanComplete = true
		m.Library.ScanProcessed = msg.Processed
		m.Library.ScanSkipped = msg.Skipped
		return m, nil

	case tea.WindowSizeMsg:
		m.UI.Width = msg.Width
		m.UI.Height = msg.Height
		return m, nil
	}

	return m, nil
}
