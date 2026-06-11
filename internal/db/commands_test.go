package db

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jamesMcMeex/waveshell/internal/messages"
	"github.com/jamesMcMeex/waveshell/internal/model"
)

func seedTestLibrary(t *testing.T, db *sql.DB) {
	artistAID, err := InsertArtist(db, "Artist A", "Artist A")
	require.NoError(t, err)
	artistBID, err := InsertArtist(db, "Artist B", "Artist B")
	require.NoError(t, err)

	album := model.Album{
		Title:       "Album One",
		ArtistID:    &artistAID,
		AlbumArtist: "Artist A",
		Year:        2020,
		Genre:       "Electronic",
		Label:       "Label X",
		TrackCount:  2,
		DiscCount:   1,
	}
	albumID1, err := InsertAlbum(db, album)
	require.NoError(t, err)

	album2 := model.Album{
		Title:       "Album Two",
		ArtistID:    &artistBID,
		AlbumArtist: "Artist B",
		Year:        2022,
		Genre:       "Rock",
		Label:       "Label Y",
		TrackCount:  1,
		DiscCount:   1,
	}
	albumID2, err := InsertAlbum(db, album2)
	require.NoError(t, err)

	track := model.Track{
		AlbumID:       albumID1,
		FilePath:      "/music/a1.flac",
		FileSizeBytes: 1000,
		LastModified:  100,
		Title:         "Track One",
		Artist:        "Artist A",
		Album:         "Album One",
		AlbumArtist:   "Artist A",
		DurationMs:    5000,
		Format:        "FLAC",
		Codec:         "flac",
		Container:     "FLAC",
		SampleRate:    44100,
		Bitrate:       800,
		Genre:         "Electronic",
		Label:         "Label X",
		Year:          2020,
		TrackNumber:   1,
	}
	_, err = UpsertTrack(db, track, 1000)
	require.NoError(t, err)

	track2 := model.Track{
		AlbumID:       albumID1,
		FilePath:      "/music/a2.flac",
		FileSizeBytes: 2000,
		LastModified:  101,
		Title:         "Track Two",
		Artist:        "Artist A",
		Album:         "Album One",
		AlbumArtist:   "Artist A",
		DurationMs:    6000,
		Format:        "FLAC",
		Codec:         "flac",
		Container:     "FLAC",
		SampleRate:    48000,
		Bitrate:       900,
		Genre:         "Electronic",
		Label:         "Label X",
		Year:          2020,
		TrackNumber:   2,
	}
	_, err = UpsertTrack(db, track2, 1000)
	require.NoError(t, err)

	track3 := model.Track{
		AlbumID:       albumID2,
		FilePath:      "/music/b1.mp3",
		FileSizeBytes: 1500,
		LastModified:  200,
		Title:         "Track One",
		Artist:        "Artist B",
		Album:         "Album Two",
		AlbumArtist:   "Artist B",
		DurationMs:    4000,
		Format:        "MP3",
		Codec:         "mp3",
		Container:     "MP3",
		SampleRate:    44100,
		Bitrate:       320,
		Genre:         "Rock",
		Label:         "Label Y",
		Year:          2022,
		TrackNumber:   1,
	}
	_, err = UpsertTrack(db, track3, 1000)
	require.NoError(t, err)
}

func TestQueryArtistsCmd(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)
	seedTestLibrary(t, db)

	cmd := QueryArtistsCmd(db)
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(messages.ArtistListResultMsg)
	require.True(t, ok, "expected ArtistListResultMsg, got %T", msg)
	require.Len(t, result.Artists, 2)
	assert.Equal(t, "Artist A", result.Artists[0].Name)
	assert.Equal(t, "Artist B", result.Artists[1].Name)
}

func TestQueryTagSliceCmd_label(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)
	seedTestLibrary(t, db)

	cmd := QueryTagSliceCmd(db, model.BrowseModeLabel)
	msg := cmd()
	result, ok := msg.(messages.TagSliceResultMsg)
	require.True(t, ok)
	assert.Equal(t, model.BrowseModeLabel, result.Mode)
	assert.Equal(t, []string{"Label X", "Label Y"}, result.Values)
}

func TestQueryTagSliceCmd_genre(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)
	seedTestLibrary(t, db)

	cmd := QueryTagSliceCmd(db, model.BrowseModeGenre)
	msg := cmd()
	result, ok := msg.(messages.TagSliceResultMsg)
	require.True(t, ok)
	assert.Equal(t, model.BrowseModeGenre, result.Mode)
	assert.Equal(t, []string{"Electronic", "Rock"}, result.Values)
}

func TestQueryTagSliceCmd_year(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)
	seedTestLibrary(t, db)

	cmd := QueryTagSliceCmd(db, model.BrowseModeYear)
	msg := cmd()
	result, ok := msg.(messages.TagSliceResultMsg)
	require.True(t, ok)
	assert.Equal(t, []string{"2020", "2022"}, result.Values)
}

func TestQueryAlbumsForArtistCmd(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)
	seedTestLibrary(t, db)

	// Find artist A's ID
	var artistID int64
	err := db.QueryRow(`SELECT id FROM artists WHERE name = ?`, "Artist A").Scan(&artistID)
	require.NoError(t, err)

	cmd := QueryAlbumsForArtistCmd(db, artistID)
	msg := cmd()
	result, ok := msg.(messages.AlbumListResultMsg)
	require.True(t, ok)
	require.Len(t, result.Albums, 1)
	assert.Equal(t, "Album One", result.Albums[0].Title)
	assert.Equal(t, 2020, result.Albums[0].Year)
	assert.Equal(t, 2, result.Albums[0].TrackCount)
}

func TestQueryAlbumsForTagCmd(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)
	seedTestLibrary(t, db)

	cmd := QueryAlbumsForTagCmd(db, model.BrowseModeLabel, "Label X")
	msg := cmd()
	result, ok := msg.(messages.AlbumListResultMsg)
	require.True(t, ok)
	require.Len(t, result.Albums, 1)
	assert.Equal(t, "Album One", result.Albums[0].Title)
}

func TestQueryAlbumsForTagCmd_noResults(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)
	seedTestLibrary(t, db)

	cmd := QueryAlbumsForTagCmd(db, model.BrowseModeLabel, "Nonexistent")
	msg := cmd()
	result, ok := msg.(messages.AlbumListResultMsg)
	require.True(t, ok)
	assert.Empty(t, result.Albums)
}

func TestQueryTracksCmd(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)
	seedTestLibrary(t, db)

	var albumID int64
	err := db.QueryRow(`SELECT id FROM albums WHERE title = ?`, "Album One").Scan(&albumID)
	require.NoError(t, err)

	cmd := QueryTracksCmd(db, albumID)
	msg := cmd()
	result, ok := msg.(messages.TrackListResultMsg)
	require.True(t, ok)
	require.Len(t, result.Tracks, 2)
	assert.Equal(t, "Track One", result.Tracks[0].Title)
	assert.Equal(t, "Track Two", result.Tracks[1].Title)
	assert.Equal(t, "FLAC", result.Tracks[0].Format)
}

func TestQueryTracksCmd_empty(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)

	cmd := QueryTracksCmd(db, 999)
	msg := cmd()
	result, ok := msg.(messages.TrackListResultMsg)
	require.True(t, ok)
	assert.Empty(t, result.Tracks)
}
