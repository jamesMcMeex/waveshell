package mpv

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jamesMcMeex/waveshell/internal/messages"
)

// SubscribeCmd returns a Cmd that waits for the next event from the mpv IPC
// reader and returns it as a Msg. Update re-schedules this Cmd after every
// event to keep the subscription loop alive.
func SubscribeCmd(events <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			// Channel closed — connection is gone
			return messages.MPVConnectionLostMsg{Err: nil}
		}
		return msg
	}
}
