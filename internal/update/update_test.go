package update

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jamesMcMeex/waveshell/internal/config"
	"github.com/jamesMcMeex/waveshell/internal/messages"
)

func TestInitialModel(t *testing.T) {
	m := InitialModel()
	assert.Zero(t, m)
}

func TestInit(t *testing.T) {
	t.Run("returns scan cmd when scan_on_startup is true", func(t *testing.T) {
		m := Model{
			Config: &config.Config{
				Library: config.LibraryConfig{
					ScanOnStartup: true,
					Paths:         []string{"/music"},
				},
			},
		}
		cmd := m.Init()
		require.NotNil(t, cmd)
		msg := cmd()
		_, ok := msg.(messages.ScanStartedMsg)
		assert.True(t, ok, "expected ScanStartedMsg")
	})

	t.Run("returns nil when scan_on_startup is false", func(t *testing.T) {
		m := Model{
			Config: &config.Config{
				Library: config.LibraryConfig{
					ScanOnStartup: false,
				},
			},
		}
		assert.Nil(t, m.Init())
	})

	t.Run("returns nil when no library paths configured", func(t *testing.T) {
		m := Model{
			Config: &config.Config{
				Library: config.LibraryConfig{
					ScanOnStartup: true,
					Paths:         []string{},
				},
			},
		}
		assert.Nil(t, m.Init())
	})

	t.Run("returns nil when config is nil", func(t *testing.T) {
		m := Model{}
		assert.Nil(t, m.Init())
	})
}

func TestScanStartedMsg(t *testing.T) {
	m := Model{Config: &config.Config{Library: config.LibraryConfig{Paths: []string{"/music"}}}}
	result, cmd := m.Update(messages.ScanStartedMsg{})
	require.NotNil(t, result)

	updated := result.(Model)
	assert.True(t, updated.Library.Scanning)
	assert.False(t, updated.Library.ScanComplete)
	assert.Equal(t, 0, updated.Library.ScanProcessed)
	assert.Equal(t, 0, updated.Library.ScanSkipped)
	assert.Equal(t, -1, updated.Library.ScanTotal)
	require.NotNil(t, cmd, "should start scan cmd")
}

func TestScanStartedMsgIgnoredWhenAlreadyScanning(t *testing.T) {
	m := Model{Config: &config.Config{Library: config.LibraryConfig{Paths: []string{"/music"}}}}
	m.Library.Scanning = true

	result, cmd := m.Update(messages.ScanStartedMsg{})

	updated := result.(Model)
	assert.True(t, updated.Library.Scanning, "should remain scanning")
	assert.Nil(t, cmd, "should not start another scan")
}

func TestScanProgressMsg(t *testing.T) {
	nextCmd := func() tea.Msg { return messages.ScanCompleteMsg{} }
	m := Model{}
	m.Library.ScanTotal = 10

	result, cmd := m.Update(messages.ScanProgressMsg{
		Processed:   5,
		Total:       10,
		CurrentPath: "/music/track.flac",
		NextCmd:     nextCmd,
	})

	updated := result.(Model)
	assert.True(t, updated.Library.Scanning)
	assert.Equal(t, 5, updated.Library.ScanProcessed)
	assert.Equal(t, 10, updated.Library.ScanTotal)
	assert.Equal(t, "/music/track.flac", updated.Library.ScanCurrent)
	assert.NotNil(t, cmd)
}

func TestScanFileErrorMsg(t *testing.T) {
	nextCmd := func() tea.Msg { return messages.ScanCompleteMsg{} }
	m := Model{}

	result, cmd := m.Update(messages.ScanFileErrorMsg{
		Path:    "/music/bad.flac",
		Err:     assert.AnError,
		NextCmd: nextCmd,
	})

	updated := result.(Model)
	assert.Equal(t, 1, updated.Library.ScanSkipped)
	assert.NotNil(t, cmd)
}

func TestScanCompleteMsg(t *testing.T) {
	tests := []struct {
		name    string
		msg     messages.ScanCompleteMsg
		preScan bool
	}{
		{
			name: "stops scanning",
			msg: messages.ScanCompleteMsg{
				Processed: 100,
				Skipped:   2,
			},
			preScan: true,
		},
		{
			name: "stops scanning with zero values",
			msg: messages.ScanCompleteMsg{
				Processed: 0,
				Skipped:   0,
			},
			preScan: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{}
			m.Library.Scanning = tt.preScan

			result, cmd := m.Update(tt.msg)

			updated := result.(Model)
			assert.False(t, updated.Library.Scanning)
			assert.True(t, updated.Library.ScanComplete)
			assert.Equal(t, tt.msg.Processed, updated.Library.ScanProcessed)
			assert.Equal(t, tt.msg.Skipped, updated.Library.ScanSkipped)
			assert.Nil(t, cmd)
		})
	}
}

func TestWindowSizeMsg(t *testing.T) {
	m := Model{}
	result, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	updated := result.(Model)
	assert.Equal(t, 120, updated.UI.Width)
	assert.Equal(t, 40, updated.UI.Height)
	assert.Nil(t, cmd)
}

func TestUnknownMsgNoOp(t *testing.T) {
	m := Model{Library: LibraryState{Scanning: true}}
	result, cmd := m.Update(nil)

	updated := result.(Model)
	assert.Equal(t, m, updated)
	assert.Nil(t, cmd)
}
