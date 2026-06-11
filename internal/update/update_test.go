package update

import (
	"database/sql"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jamesMcMeex/waveshell/internal/config"
	"github.com/jamesMcMeex/waveshell/internal/db"
	"github.com/jamesMcMeex/waveshell/internal/messages"
	"github.com/jamesMcMeex/waveshell/internal/model"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	err = db.Migrate(d)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func TestInitialModel(t *testing.T) {
	m := InitialModel()
	assert.Equal(t, model.BrowseModeArtist, m.UI.BrowseMode)
	assert.Equal(t, model.PaneLeft, m.UI.ActivePane)
	assert.Equal(t, OverlayNone, m.UI.ActiveOverlay)
}

func TestInit(t *testing.T) {
	t.Run("returns batch cmd when scan_on_startup is true", func(t *testing.T) {
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
	})

	t.Run("returns nil when scan_on_startup is false and no db", func(t *testing.T) {
		m := Model{
			Config: &config.Config{
				Library: config.LibraryConfig{
					ScanOnStartup: false,
				},
			},
		}
		assert.Nil(t, m.Init())
	})

	t.Run("returns left pane query when no scan needed but db present", func(t *testing.T) {
		m := Model{
			Config: &config.Config{
				Library: config.LibraryConfig{
					ScanOnStartup: false,
				},
			},
			DB: openTestDB(t),
		}
		cmd := m.Init()
		require.NotNil(t, cmd)
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
			_ = cmd // cmd may be nil or re-query; either is acceptable
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
	assert.Equal(t, m.Library, updated.Library)
	assert.Equal(t, m.UI.BrowseMode, updated.UI.BrowseMode)
	assert.Equal(t, m.UI.ActivePane, updated.UI.ActivePane)
	assert.Nil(t, cmd)
}

func TestArtistListResultMsg(t *testing.T) {
	m := Model{}
	result, cmd := m.Update(messages.ArtistListResultMsg{
		Artists: []model.Artist{
			{ID: 1, Name: "Artist A"},
			{ID: 2, Name: "Artist B"},
		},
	})

	updated := result.(Model)
	require.Len(t, updated.Library.Artists, 2)
	assert.Equal(t, "Artist A", updated.Library.Artists[0].Name)
	assert.Equal(t, 0, updated.UI.LeftCursor)
	assert.Nil(t, cmd)
}

func TestTagSliceResultMsg(t *testing.T) {
	m := Model{}
	result, cmd := m.Update(messages.TagSliceResultMsg{
		Mode:   model.BrowseModeLabel,
		Values: []string{"Label X", "Label Y"},
	})

	updated := result.(Model)
	require.Len(t, updated.Library.TagSliceValues, 2)
	assert.Equal(t, "Label X", updated.Library.TagSliceValues[0])
	assert.Nil(t, cmd)
}

func TestAlbumListResultMsg(t *testing.T) {
	m := Model{}
	result, cmd := m.Update(messages.AlbumListResultMsg{
		Mode: model.BrowseModeArtist,
		Key:  "1",
		Albums: []model.Album{
			{ID: 1, Title: "Album One", Year: 2020, TrackCount: 10},
		},
	})

	updated := result.(Model)
	require.Len(t, updated.Library.Albums, 1)
	assert.Equal(t, "Album One", updated.Library.Albums[0].Title)
	assert.Equal(t, 0, updated.UI.MiddleCursor)
	assert.Nil(t, cmd)
}

func TestTrackListResultMsg(t *testing.T) {
	m := Model{}
	result, cmd := m.Update(messages.TrackListResultMsg{
		AlbumID: 1,
		Tracks: []model.Track{
			{ID: 1, Title: "Track One", DurationMs: 5000, Format: "FLAC", SampleRate: 44100},
		},
	})

	updated := result.(Model)
	require.Len(t, updated.Library.Tracks, 1)
	assert.Equal(t, "Track One", updated.Library.Tracks[0].Title)
	assert.Equal(t, 0, updated.UI.RightCursor)
	assert.Nil(t, cmd)
}
