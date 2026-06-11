package update

import (
	"database/sql"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/jamesMcMeex/waveshell/internal/db"
	"github.com/jamesMcMeex/waveshell/internal/messages"
	"github.com/jamesMcMeex/waveshell/internal/model"
)

func openTestDBInt(t *testing.T) *sql.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	err = db.Migrate(d)
	require.NoError(t, err)
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func TestUI_artistQueryAfterScan(t *testing.T) {
	database := openTestDBInt(t)

	artistID, err := db.InsertArtist(database, "Test Artist", "Test Artist")
	require.NoError(t, err)
	album := model.Album{
		Title:    "Test Album",
		ArtistID: &artistID,
	}
	albumID, err := db.InsertAlbum(database, album)
	require.NoError(t, err)
	track := model.Track{
		AlbumID:       albumID,
		ArtistID:      artistID,
		FilePath:      "/tmp/test.flac",
		FileSizeBytes: 1000,
		LastModified:  100,
		Title:         "Test Track",
		Artist:        "Test Artist",
		Album:         "Test Album",
		DurationMs:    5000,
		Format:        "FLAC",
		Codec:         "flac",
		Container:     "FLAC",
		SampleRate:    44100,
		Bitrate:       800,
		TrackNumber:   1,
	}
	_, err = db.UpsertTrack(database, track, time.Now().Unix())
	require.NoError(t, err)

	cmd := db.QueryArtistsCmd(database)
	msg := cmd()
	artistMsg, ok := msg.(messages.ArtistListResultMsg)
	require.True(t, ok)
	require.Len(t, artistMsg.Artists, 1)
	require.Equal(t, "Test Artist", artistMsg.Artists[0].Name)

	m := Model{
		DB: database,
		UI: UIState{
			BrowseMode: model.BrowseModeArtist,
			ActivePane: model.PaneLeft,
			Width:      120,
			Height:     40,
		},
	}
	r, _ := m.Update(artistMsg)
	m = r.(Model)
	require.Len(t, m.Library.Artists, 1)

	enterKey := tea.KeyMsg{Type: tea.KeyEnter}
	r, cmd2 := m.Update(enterKey)
	m = r.(Model)
	require.NotNil(t, cmd2)
	msg2 := cmd2()
	albumMsg, ok := msg2.(messages.AlbumListResultMsg)
	require.True(t, ok)
	require.Len(t, albumMsg.Albums, 1)
	require.Equal(t, "Test Album", albumMsg.Albums[0].Title)

	r, _ = m.Update(albumMsg)
	m = r.(Model)
	require.Len(t, m.Library.Albums, 1)
	require.Equal(t, model.PaneMiddle, m.UI.ActivePane, "album result should auto-advance to middle pane")

	r, cmd3 := m.Update(enterKey)
	m = r.(Model)
	require.NotNil(t, cmd3)
	msg3 := cmd3()
	trackMsg, ok := msg3.(messages.TrackListResultMsg)
	require.True(t, ok)
	require.Len(t, trackMsg.Tracks, 1)
	require.Equal(t, "Test Track", trackMsg.Tracks[0].Title)

	r, _ = m.Update(trackMsg)
	m = r.(Model)
	require.Len(t, m.Library.Tracks, 1)

	view := m.View()
	require.NotEmpty(t, view)
	t.Logf("View output:\n%s", view)
}

func TestUI_scanStateFromStartToComplete(t *testing.T) {
	database := openTestDBInt(t)

	m := Model{
		DB: database,
		UI: UIState{
			BrowseMode: model.BrowseModeArtist,
			ActivePane: model.PaneLeft,
			Width:      120,
			Height:     40,
		},
	}

	cmd := db.QueryArtistsCmd(database)
	msg := cmd()
	r, _ := m.Update(msg)
	m = r.(Model)
	require.Empty(t, m.Library.Artists, "no artists yet")

	artistID, err := db.InsertArtist(database, "Test Artist", "Test Artist")
	require.NoError(t, err)
	album := model.Album{
		Title:    "Test Album",
		ArtistID: &artistID,
	}
	albumID, err := db.InsertAlbum(database, album)
	require.NoError(t, err)
	track := model.Track{
		AlbumID:       albumID,
		ArtistID:      artistID,
		FilePath:      "/tmp/test.flac",
		FileSizeBytes: 1000,
		LastModified:  100,
		Title:         "Test Track",
		Artist:        "Test Artist",
		Album:         "Test Album",
		DurationMs:    5000,
		Format:        "FLAC",
		Codec:         "flac",
		Container:     "FLAC",
		SampleRate:    44100,
		Bitrate:       800,
		TrackNumber:   1,
	}
	_, err = db.UpsertTrack(database, track, time.Now().Unix())
	require.NoError(t, err)

	r, cmd = m.Update(messages.ScanCompleteMsg{Processed: 1, Skipped: 0})
	m = r.(Model)
	require.NotNil(t, cmd)
	msg = cmd()
	artistMsg, ok := msg.(messages.ArtistListResultMsg)
	require.True(t, ok)
	require.Len(t, artistMsg.Artists, 1, "should find artist after scan")
	require.Equal(t, "Test Artist", artistMsg.Artists[0].Name)

	r, _ = m.Update(artistMsg)
	m = r.(Model)
	require.Len(t, m.Library.Artists, 1)

	view := m.View()
	require.NotEmpty(t, view)
	t.Logf("View output:\n%s", view)
	require.Contains(t, view, "Test Artist", "view should show the artist name")
}
