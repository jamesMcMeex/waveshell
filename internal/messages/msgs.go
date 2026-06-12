// Package messages defines all custom tea.Msg types for the waveshell TUI.
// All async subsystems produce and consume these types, preventing import
// cycles by centralising Msg definitions in a single leaf package.
package messages

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jamesMcMeex/waveshell/internal/model"
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

// ArtistListResultMsg carries the artist list for the left pane in artist browse mode.
type ArtistListResultMsg struct {
	Artists []model.Artist
}

// TagSliceResultMsg carries distinct values for the left pane in label, genre, or year browse modes.
type TagSliceResultMsg struct {
	Mode   model.BrowseMode
	Values []string
}

// AlbumListResultMsg carries albums for the middle pane, filtered by the selected left-pane value.
type AlbumListResultMsg struct {
	Mode   model.BrowseMode
	Key    string
	Albums []model.Album
}

// TrackListResultMsg carries tracks for the tracks pane, filtered by the selected album.
type TrackListResultMsg struct {
	AlbumID int64
	Tracks  []model.Track
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

// ── mpv Process ───────────────────────────────────────────────────────────────

// MPVReadyMsg is sent after the mpv subprocess has started and the Unix socket
// connection has been established. Events is the channel for SubscribeCmd.
type MPVReadyMsg struct {
	Events <-chan tea.Msg
}

// MPVNotFoundMsg is sent when the mpv binary cannot be located on PATH.
type MPVNotFoundMsg struct{}

// MPVConnectionLostMsg is sent when the socket connection drops unexpectedly.
type MPVConnectionLostMsg struct {
	Err error
}

// ── mpv Events ────────────────────────────────────────────────────────────────

// PlaybackStateChangedMsg is sent when mpv's pause property changes.
type PlaybackStateChangedMsg struct {
	State model.PlaybackState
}

// TimePositionChangedMsg is sent when mpv's time-pos property changes.
type TimePositionChangedMsg struct {
	PositionSec float64
}

// DurationChangedMsg is sent when mpv's duration property is first available.
type DurationChangedMsg struct {
	DurationSec float64
}

// VolumeChangedMsg is sent when mpv's volume property changes.
type VolumeChangedMsg struct {
	Volume int
}

// TrackEndedMsg is sent when mpv's end-file event fires.
type TrackEndedMsg struct {
	Reason model.TrackEndReason
}
