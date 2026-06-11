package db

import (
	"database/sql"
	"fmt"

	"github.com/jamesMcMeex/waveshell/internal/model"
)

func InsertArtist(db *sql.DB, name string, nameSort string) (int64, error) {
	res, err := db.Exec(
		`INSERT OR IGNORE INTO artists (name, name_sort) VALUES (?, ?)`,
		name, nameSort,
	)
	if err != nil {
		return 0, fmt.Errorf("insert artist: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	if id == 0 {
		err = db.QueryRow(`SELECT id FROM artists WHERE name = ?`, name).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("find existing artist: %w", err)
		}
	}
	return id, nil
}

func InsertAlbum(db *sql.DB, a model.Album) (int64, error) {
	res, err := db.Exec(`
		INSERT INTO albums (title, artist_id, album_artist, year, genre, label, grouping,
			track_count, disc_count, rg_album_gain, rg_album_peak, r128_album_gain)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(title, artist_id) DO UPDATE SET
			year=excluded.year, genre=excluded.genre, label=excluded.label,
			grouping=excluded.grouping, track_count=excluded.track_count,
			disc_count=excluded.disc_count, rg_album_gain=excluded.rg_album_gain,
			rg_album_peak=excluded.rg_album_peak, r128_album_gain=excluded.r128_album_gain`,
		a.Title, a.ArtistID, sqlNullString(a.AlbumArtist),
		sqlNullInt(a.Year), sqlNullString(a.Genre),
		sqlNullString(a.Label), sqlNullString(a.Grouping),
		a.TrackCount, a.DiscCount,
		sqlNullFloat64(a.RGAlbumGain), sqlNullFloat64(a.RGAlbumPeak),
		sqlNullFloat64(a.R128AlbumGain),
	)
	if err != nil {
		return 0, fmt.Errorf("insert album: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	if id == 0 {
		err = db.QueryRow(
			`SELECT id FROM albums WHERE title = ? AND (artist_id = ? OR (artist_id IS NULL AND ? IS NULL))`,
			a.Title, a.ArtistID, a.ArtistID,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("find existing album: %w", err)
		}
	}
	return id, nil
}

func UpsertTrack(db *sql.DB, t model.Track, nowUnix int64) (int64, error) {
	var existingID int64
	var existingMod int64
	err := db.QueryRow(
		`SELECT id, last_modified FROM tracks WHERE file_path = ?`, t.FilePath,
	).Scan(&existingID, &existingMod)

	if err == sql.ErrNoRows {
		t.DateAdded = nowUnix
		return insertTrack(db, t)
	}
	if err != nil {
		return 0, fmt.Errorf("check existing track: %w", err)
	}

	if existingMod == t.LastModified {
		return existingID, nil
	}

	t.ID = existingID
	return updateTrack(db, t)
}

func TrackByPath(db *sql.DB, filePath string) (*model.Track, error) {
	row := db.QueryRow(`
		SELECT id, file_path, file_size_bytes, last_modified, date_added,
			album_id, artist_id, title, artist,
			COALESCE(album_artist,''), COALESCE(album,''),
			track_number, disc_number, year,
			COALESCE(genre,''), COALESCE(grouping,''), COALESCE(label,''),
			duration_ms, format, codec, container, sample_rate, bit_depth, bitrate,
			rg_track_gain, rg_track_peak, rg_album_gain, rg_album_peak,
			r128_track_gain, r128_album_gain,
			has_artwork, artwork_width, artwork_height,
			COALESCE(artwork_format,''), artwork_size_bytes
		FROM tracks WHERE file_path = ?`, filePath,
	)

	var t model.Track
	var tn, dn, yr sql.NullInt64
	var bd sql.NullInt64
	var rtg, rtp, rag, rap, r128tg, r128ag sql.NullFloat64
	var aw, ah, asb sql.NullInt64

	err := row.Scan(
		&t.ID, &t.FilePath, &t.FileSizeBytes, &t.LastModified, &t.DateAdded,
		&t.AlbumID, &t.ArtistID, &t.Title, &t.Artist,
		&t.AlbumArtist, &t.Album,
		&tn, &dn, &yr,
		&t.Genre, &t.Grouping, &t.Label,
		&t.DurationMs, &t.Format, &t.Codec, &t.Container, &t.SampleRate, &bd, &t.Bitrate,
		&rtg, &rtp, &rag, &rap, &r128tg, &r128ag,
		&t.HasArtwork, &aw, &ah, &t.ArtworkFormat, &asb,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query track by path: %w", err)
	}

	if tn.Valid {
		t.TrackNumber = int(tn.Int64)
	}
	if dn.Valid {
		t.DiscNumber = int(dn.Int64)
	}
	if yr.Valid {
		t.Year = int(yr.Int64)
	}
	if bd.Valid {
		v := int(bd.Int64)
		t.BitDepth = &v
	}
	if rtg.Valid {
		v := rtg.Float64
		t.RGTrackGain = &v
	}
	if rtp.Valid {
		v := rtp.Float64
		t.RGTrackPeak = &v
	}
	if rag.Valid {
		v := rag.Float64
		t.RGAlbumGain = &v
	}
	if rap.Valid {
		v := rap.Float64
		t.RGAlbumPeak = &v
	}
	if r128tg.Valid {
		v := r128tg.Float64
		t.R128TrackGain = &v
	}
	if r128ag.Valid {
		v := r128ag.Float64
		t.R128AlbumGain = &v
	}
	if aw.Valid {
		t.ArtworkWidth = int(aw.Int64)
	}
	if ah.Valid {
		t.ArtworkHeight = int(ah.Int64)
	}
	if asb.Valid {
		t.ArtworkSizeBytes = asb.Int64
	}

	return &t, nil
}

func DeleteTrack(db *sql.DB, filePath string) error {
	_, err := db.Exec(`DELETE FROM tracks WHERE file_path = ?`, filePath)
	return err
}

func AllTrackPaths(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT file_path FROM tracks`)
	if err != nil {
		return nil, fmt.Errorf("query all paths: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan path: %w", err)
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

func insertTrack(db *sql.DB, t model.Track) (int64, error) {
	res, err := db.Exec(`
		INSERT INTO tracks (
			file_path, file_size_bytes, last_modified, date_added,
			album_id, artist_id, title, artist, album_artist, album,
			track_number, disc_number, year, genre, grouping, label,
			duration_ms, format, codec, container, sample_rate, bit_depth, bitrate,
			rg_track_gain, rg_track_peak, rg_album_gain, rg_album_peak,
			r128_track_gain, r128_album_gain,
			has_artwork, artwork_width, artwork_height, artwork_format, artwork_size_bytes
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.FilePath, t.FileSizeBytes, t.LastModified, t.DateAdded,
		t.AlbumID, t.ArtistID, t.Title, t.Artist,
		sqlNullString(t.AlbumArtist), sqlNullString(t.Album),
		sqlNullInt(t.TrackNumber), sqlNullInt(t.DiscNumber),
		sqlNullInt(t.Year), sqlNullString(t.Genre),
		sqlNullString(t.Grouping), sqlNullString(t.Label),
		t.DurationMs, t.Format, t.Codec, t.Container, t.SampleRate,
		sqlNullIntPtr(t.BitDepth), t.Bitrate,
		sqlNullFloat64(t.RGTrackGain), sqlNullFloat64(t.RGTrackPeak),
		sqlNullFloat64(t.RGAlbumGain), sqlNullFloat64(t.RGAlbumPeak),
		sqlNullFloat64(t.R128TrackGain), sqlNullFloat64(t.R128AlbumGain),
		t.HasArtwork,
		sqlNullInt(t.ArtworkWidth), sqlNullInt(t.ArtworkHeight),
		sqlNullString(t.ArtworkFormat), sqlNullInt64(t.ArtworkSizeBytes),
	)
	if err != nil {
		return 0, fmt.Errorf("insert track: %w", err)
	}
	return res.LastInsertId()
}

func updateTrack(db *sql.DB, t model.Track) (int64, error) {
	_, err := db.Exec(`
		UPDATE tracks SET
			file_size_bytes=?, last_modified=?, album_id=?, artist_id=?,
			title=?, artist=?, album_artist=?, album=?,
			track_number=?, disc_number=?, year=?, genre=?, grouping=?, label=?,
			duration_ms=?, format=?, codec=?, container=?, sample_rate=?, bit_depth=?, bitrate=?,
			rg_track_gain=?, rg_track_peak=?, rg_album_gain=?, rg_album_peak=?,
			r128_track_gain=?, r128_album_gain=?,
			has_artwork=?, artwork_width=?, artwork_height=?, artwork_format=?, artwork_size_bytes=?
		WHERE id=?`,
		t.FileSizeBytes, t.LastModified, t.AlbumID, t.ArtistID,
		t.Title, t.Artist, sqlNullString(t.AlbumArtist), sqlNullString(t.Album),
		sqlNullInt(t.TrackNumber), sqlNullInt(t.DiscNumber),
		sqlNullInt(t.Year), sqlNullString(t.Genre),
		sqlNullString(t.Grouping), sqlNullString(t.Label),
		t.DurationMs, t.Format, t.Codec, t.Container, t.SampleRate,
		sqlNullIntPtr(t.BitDepth), t.Bitrate,
		sqlNullFloat64(t.RGTrackGain), sqlNullFloat64(t.RGTrackPeak),
		sqlNullFloat64(t.RGAlbumGain), sqlNullFloat64(t.RGAlbumPeak),
		sqlNullFloat64(t.R128TrackGain), sqlNullFloat64(t.R128AlbumGain),
		t.HasArtwork,
		sqlNullInt(t.ArtworkWidth), sqlNullInt(t.ArtworkHeight),
		sqlNullString(t.ArtworkFormat), sqlNullInt64(t.ArtworkSizeBytes),
		t.ID,
	)
	if err != nil {
		return 0, fmt.Errorf("update track: %w", err)
	}
	return t.ID, nil
}

func sqlNullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func sqlNullInt(n int) interface{} {
	if n == 0 {
		return nil
	}
	return n
}

func sqlNullInt64(n int64) interface{} {
	if n == 0 {
		return nil
	}
	return n
}

func sqlNullIntPtr(p *int) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func sqlNullFloat64(f *float64) interface{} {
	if f == nil {
		return nil
	}
	return *f
}
