// Package messages defines all custom tea.Msg types for the waveshell TUI.
// All async subsystems produce and consume these types, preventing import
// cycles by centralising Msg definitions in a single leaf package.
package messages

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ScanStartedMsg signals the beginning of a library scan.
// The model handles this by initialising scan state and starting the scanner Cmd.
type ScanStartedMsg struct{}

// ScanProgressMsg is emitted after each file is scanned during a library scan.
// NextCmd carries the recursive continuation of the scan Cmd.
type ScanProgressMsg struct {
	Processed   int
	Total       int // -1 until directory walk completes and total is known
	CurrentPath string
	NextCmd     tea.Cmd
}

// ScanFileErrorMsg is emitted when a single file fails to scan or persist.
// NextCmd carries the recursive continuation of the scan Cmd.
type ScanFileErrorMsg struct {
	Path    string
	Err     error
	NextCmd tea.Cmd
}

// ScanCompleteMsg signals the end of a library scan, with final counts.
type ScanCompleteMsg struct {
	Processed int
	Skipped   int
}

// DBErrorMsg is emitted when a database operation fails.
// If Fatal is true the model should shut down.
type DBErrorMsg struct {
	Op    string
	Err   error
	Fatal bool
}

// TickMsg is emitted every second by TickCmd for periodic UI updates.
type TickMsg struct {
	Time time.Time
}

func TickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg{Time: t}
	})
}
