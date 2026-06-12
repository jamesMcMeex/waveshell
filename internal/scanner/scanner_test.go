package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jamesMcMeex/waveshell/internal/model"
)

func fixturePath(t *testing.T, parts ...string) string {
	t.Helper()
	return filepath.Join(append([]string{"..", "..", "testdata", "fixtures"}, parts...)...)
}

func TestIsSupportedExt(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"song.flac", true},
		{"song.alac", true},
		{"song.m4a", true},
		{"song.aiff", true},
		{"song.aif", true},
		{"song.mp3", true},
		{"song.wav", true},
		{"song.ogg", true},
		{"song.txt", false},
		{"song.mp4", false},
		{"song", false},
		{"song.FLAC", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := model.IsSupportedExt(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSupportedExtensions(t *testing.T) {
	exts := model.SupportedExtensions()
	expected := []string{".flac", ".alac", ".m4a", ".aiff", ".aif", ".mp3", ".wav", ".ogg"}
	assert.ElementsMatch(t, expected, exts)
}

func TestScanFLAC(t *testing.T) {
	path := fixturePath(t, "flac", "sample.flac")
	_, err := os.Stat(path)
	require.NoError(t, err, "fixture file must exist")

	tags, err := ScanFile(path)
	require.NoError(t, err)
	require.NotNil(t, tags)

	assert.Equal(t, "Test Track", tags.Title)
	assert.Equal(t, "Test Artist", tags.Artist)
	assert.Equal(t, "Test Album Artist", tags.AlbumArtist)
	assert.Equal(t, "Test Album", tags.Album)
	assert.Equal(t, "Test Genre", tags.Genre)
	assert.Equal(t, 2024, tags.Year)
	assert.Equal(t, 1, tags.TrackNumber)
	assert.Equal(t, 1, tags.DiscNumber)
	assert.Equal(t, model.AudioFormatFLAC, tags.Format)
	assert.Equal(t, model.CodecFLAC, tags.Codec)
	assert.Equal(t, "FLAC", tags.Container)
	assert.Equal(t, 44100, tags.SampleRate)
	assert.Greater(t, tags.DurationMs, 0)
}

func TestScanMP3(t *testing.T) {
	path := fixturePath(t, "mp3", "sample.mp3")
	_, err := os.Stat(path)
	require.NoError(t, err)

	tags, err := ScanFile(path)
	require.NoError(t, err)
	require.NotNil(t, tags)

	assert.Equal(t, "Test Track", tags.Title)
	assert.Equal(t, "Test Artist", tags.Artist)
	assert.Equal(t, "Test Album Artist", tags.AlbumArtist)
	assert.Equal(t, "Test Album", tags.Album)
	assert.Equal(t, "Test Genre", tags.Genre)
	assert.Equal(t, 2024, tags.Year)
	assert.Equal(t, 1, tags.TrackNumber)
	assert.Equal(t, 1, tags.DiscNumber)
	assert.Equal(t, model.AudioFormatMP3, tags.Format)
}

func TestScanM4A(t *testing.T) {
	path := fixturePath(t, "m4a", "sample.m4a")
	_, err := os.Stat(path)
	require.NoError(t, err)

	tags, err := ScanFile(path)
	require.NoError(t, err)
	require.NotNil(t, tags)

	assert.Equal(t, "Test Track", tags.Title)
	assert.Equal(t, "Test Artist", tags.Artist)
	assert.Equal(t, model.AudioFormatALAC, tags.Format)
	assert.Equal(t, model.CodecALAC, tags.Codec)
}

func TestScanAIFF(t *testing.T) {
	path := fixturePath(t, "aiff", "sample.aiff")
	_, err := os.Stat(path)
	require.NoError(t, err)

	tags, err := ScanFile(path)
	require.NoError(t, err)
	require.NotNil(t, tags)

	assert.Equal(t, model.AudioFormatAIFF, tags.Format)
	assert.Equal(t, model.CodecPCM, tags.Codec)
	assert.Greater(t, tags.SampleRate, 0)
	assert.Greater(t, tags.DurationMs, 0)
}

func TestScanWAV(t *testing.T) {
	path := fixturePath(t, "wav", "sample.wav")
	_, err := os.Stat(path)
	require.NoError(t, err)

	tags, err := ScanFile(path)
	require.NoError(t, err)
	require.NotNil(t, tags)

	assert.Equal(t, model.AudioFormatWAV, tags.Format)
	assert.Equal(t, model.CodecPCM, tags.Codec)
	assert.Greater(t, tags.SampleRate, 0)
	assert.Greater(t, tags.DurationMs, 0)
}

func TestScanOGG(t *testing.T) {
	path := fixturePath(t, "ogg", "sample.ogg")
	_, err := os.Stat(path)
	require.NoError(t, err)

	tags, err := ScanFile(path)
	require.NoError(t, err)
	require.NotNil(t, tags)

	assert.Equal(t, model.AudioFormatOGG, tags.Format)
	assert.Equal(t, model.CodecVorbis, tags.Codec)
	assert.Greater(t, tags.SampleRate, 0)
}

func TestScanUnsupportedFile(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "notes.txt")
	require.NoError(t, os.WriteFile(txtPath, []byte("hello"), 0644))

	_, err := ScanFile(txtPath)
	assert.Error(t, err, "unsupported format should return error")
}

func TestWalkAndScan(t *testing.T) {
	tmpDir := t.TempDir()
	flacDir := filepath.Join(tmpDir, "music")
	require.NoError(t, os.MkdirAll(flacDir, 0755))

	src := fixturePath(t, "flac", "sample.flac")
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(flacDir, "track01.flac"), data, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(flacDir, "track02.mp3"), data, 0644))

	var results []ScanResult
	processed, skipped := WalkAndScan([]string{tmpDir}, nil, func(r ScanResult) {
		results = append(results, r)
	})

	assert.Equal(t, 2, processed, "should process both audio files")
	assert.Equal(t, 0, skipped)
	assert.Len(t, results, 2)
}

func TestWalkAndScan_SkipsNonAudio(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("hello"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "cover.jpg"), []byte("image"), 0644))

	var results []ScanResult
	processed, skipped := WalkAndScan([]string{tmpDir}, nil, func(r ScanResult) {
		results = append(results, r)
	})

	assert.Equal(t, 0, processed)
	assert.Equal(t, 0, skipped)
	assert.Empty(t, results)
}

func TestWalkAndScan_WithMtimeCheck(t *testing.T) {
	tmpDir := t.TempDir()
	src := fixturePath(t, "flac", "sample.flac")
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	testFile := filepath.Join(tmpDir, "track.flac")
	require.NoError(t, os.WriteFile(testFile, data, 0644))

	info, err := os.Stat(testFile)
	require.NoError(t, err)
	modTime := info.ModTime().Unix()

	var results []ScanResult
	processed, skipped := WalkAndScan(
		[]string{tmpDir},
		func(path string, mt int64) (bool, error) {
			return mt == modTime, nil
		},
		func(r ScanResult) {
			results = append(results, r)
		},
	)

	assert.Equal(t, 1, processed)
	assert.Equal(t, 0, skipped)
	assert.Len(t, results, 1)
}

func TestWalkAndScan_WithMtimeSkipsUnchanged(t *testing.T) {
	tmpDir := t.TempDir()
	src := fixturePath(t, "flac", "sample.flac")
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	testFile := filepath.Join(tmpDir, "track.flac")
	require.NoError(t, os.WriteFile(testFile, data, 0644))

	var results []ScanResult
	processed, skipped := WalkAndScan(
		[]string{tmpDir},
		func(path string, mt int64) (bool, error) {
			return false, nil
		},
		func(r ScanResult) {
			results = append(results, r)
		},
	)

	assert.Equal(t, 1, processed, "mtime-skipped files are counted as processed")
	assert.Equal(t, 0, skipped)
	assert.Empty(t, results)
}

func TestTrackFromTags(t *testing.T) {
	tags := &model.RawTrackTags{
		Title:      "Test Song",
		Artist:     "Test Artist",
		Album:      "Test Album",
		DurationMs: 5000,
		Format:     model.AudioFormatFLAC,
		Codec:      model.CodecFLAC,
		SampleRate: 44100,
		BitDepth:   24,
		Bitrate:    800,
	}

	track := TrackFromTags(tags, "/music/test.flac", 100000, 1234567890)
	assert.Equal(t, "/music/test.flac", track.FilePath)
	assert.Equal(t, "Test Song", track.Title)
	assert.Equal(t, "Test Artist", track.Artist)
	assert.Equal(t, int64(100000), track.FileSizeBytes)
	assert.Equal(t, int64(1234567890), track.LastModified)
	assert.Equal(t, 5000, track.DurationMs)
	assert.Equal(t, "FLAC", track.Format)
	assert.Equal(t, "flac", track.Codec)
	assert.Equal(t, 44100, track.SampleRate)
	require.NotNil(t, track.BitDepth)
	assert.Equal(t, 24, *track.BitDepth)
}

func TestInspectFLAC(t *testing.T) {
	path := fixturePath(t, "flac", "sample.flac")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	info, err := inspectFLAC(f)
	require.NoError(t, err)
	assert.Equal(t, 44100, info.SampleRate)
	assert.Greater(t, info.DurationMs, 4000)
}

func TestInspectMP3(t *testing.T) {
	path := fixturePath(t, "mp3", "sample.mp3")
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	info, err := inspectMP3(f)
	require.NoError(t, err)
	assert.Equal(t, 44100, info.SampleRate)
	assert.Greater(t, info.Bitrate, 0)
}

func TestSortName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Aphex Twin", "Aphex Twin"},
		{"The Orb", "Orb, The"},
		{"A Tribe Called Quest", "Tribe Called Quest, A"},
		{"An Awesome Band", "Awesome Band, An"},
		{"Boards of Canada", "Boards of Canada"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sortName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
