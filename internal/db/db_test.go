package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jamesMcMeex/waveshell/internal/model"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func migrateTestDB(t *testing.T, db *sql.DB) {
	t.Helper()
	err := Migrate(db)
	require.NoError(t, err)
}

func TestOpenSetsPragmas(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	require.NoError(t, err)
	assert.Contains(t, journalMode, "wal", "WAL mode should be set")
}

func TestMigrateCreatesTables(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)

	tables := []string{"schema_version", "artists", "albums", "tracks"}
	for _, name := range tables {
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
			name,
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "table %q should exist", name)
	}

	var version int
	err := db.QueryRow("SELECT version FROM schema_version").Scan(&version)
	require.NoError(t, err)
	assert.Equal(t, 1, version)
}

func TestMigrateIdempotent(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)

	err := Migrate(db)
	require.NoError(t, err, "second migration should be idempotent")

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM schema_version`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "schema_version should have exactly one row")
}

func TestInsertArtist(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)

	id, err := InsertArtist(db, "Aphex Twin", "Aphex Twin")
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	id2, err := InsertArtist(db, "Aphex Twin", "Aphex Twin")
	require.NoError(t, err)
	assert.Equal(t, id, id2, "duplicate insert should return existing id")
}

func TestInsertAlbum(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)

	artistID := int64(1)
	a := model.Album{
		Title:       "Test Album",
		ArtistID:    &artistID,
		AlbumArtist: "Test Artist",
		Year:        2024,
		Genre:       "Electronic",
		TrackCount:  10,
		DiscCount:   1,
	}
	id, err := InsertAlbum(db, a)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))
}

func TestUpsertTrack_NewTrack(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)

	track := model.Track{
		FilePath:      "/music/test.flac",
		FileSizeBytes: 12345,
		LastModified:  1000,
		Title:         "Test Track",
		Artist:        "Test Artist",
		Album:         "Test Album",
		DurationMs:    5000,
		Format:        "FLAC",
		Codec:         "flac",
		Container:     "FLAC",
		SampleRate:    44100,
		Bitrate:       800,
	}

	id, err := UpsertTrack(db, track, 2000)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	saved, err := TrackByPath(db, "/music/test.flac")
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, "Test Track", saved.Title)
	assert.Equal(t, int64(12345), saved.FileSizeBytes)
	assert.Equal(t, int64(1000), saved.LastModified)
	assert.Equal(t, int64(2000), saved.DateAdded)
}

func TestUpsertTrack_UpdateExisting(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)

	track1 := model.Track{
		FilePath:      "/music/test.flac",
		FileSizeBytes: 12345,
		LastModified:  1000,
		Title:         "Original Title",
		Artist:        "Test Artist",
		Album:         "Test Album",
		DurationMs:    5000,
		Format:        "FLAC",
		Codec:         "flac",
		Container:     "FLAC",
		SampleRate:    44100,
		Bitrate:       800,
	}
	id1, err := UpsertTrack(db, track1, 2000)
	require.NoError(t, err)

	track2 := track1
	track2.Title = "Updated Title"
	track2.LastModified = 2000
	id2, err := UpsertTrack(db, track2, 2000)
	require.NoError(t, err)
	assert.Equal(t, id1, id2, "update should keep same id")

	saved, err := TrackByPath(db, "/music/test.flac")
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, "Updated Title", saved.Title)
	assert.Equal(t, int64(2000), saved.LastModified)
}

func TestUpsertTrack_SkipUnchanged(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)

	track := model.Track{
		FilePath:      "/music/test.flac",
		FileSizeBytes: 12345,
		LastModified:  1000,
		Title:         "Test Track",
		Artist:        "Test Artist",
		Album:         "Test Album",
		DurationMs:    5000,
		Format:        "FLAC",
		Codec:         "flac",
		Container:     "FLAC",
		SampleRate:    44100,
		Bitrate:       800,
	}
	_, err := UpsertTrack(db, track, 2000)
	require.NoError(t, err)

	id2, err := UpsertTrack(db, track, 2000)
	require.NoError(t, err)
	assert.Greater(t, id2, int64(0))
}

func TestDeleteTrack(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)

	track := model.Track{
		FilePath:      "/music/test.flac",
		FileSizeBytes: 12345,
		LastModified:  1000,
		Title:         "Test Track",
		Artist:        "Test Artist",
		Album:         "Test Album",
		DurationMs:    5000,
		Format:        "FLAC",
		Codec:         "flac",
		Container:     "FLAC",
		SampleRate:    44100,
		Bitrate:       800,
	}
	_, err := UpsertTrack(db, track, 2000)
	require.NoError(t, err)

	err = DeleteTrack(db, "/music/test.flac")
	require.NoError(t, err)

	saved, err := TrackByPath(db, "/music/test.flac")
	require.NoError(t, err)
	assert.Nil(t, saved, "deleted track should not exist")
}

func TestAllTrackPaths(t *testing.T) {
	db := openTestDB(t)
	migrateTestDB(t, db)

	assert.Len(t, []string{}, 0)

	paths, err := AllTrackPaths(db)
	require.NoError(t, err)
	assert.Empty(t, paths)
}

func TestWALModeConfirmed(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	require.NoError(t, err)
	assert.Contains(t, journalMode, "wal")
}
