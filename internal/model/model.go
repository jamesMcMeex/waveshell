// Package model defines domain types (Track, Album, Artist, enums) with zero
// internal package dependencies. It is a pure data leaf package — no IO, no
// business logic, no BubbleTea imports.
package model

import "strings"

type BrowseMode string

const (
	BrowseModeArtist   BrowseMode = "artist"
	BrowseModeLabel    BrowseMode = "label"
	BrowseModeGenre    BrowseMode = "genre"
	BrowseModeYear     BrowseMode = "year"
	BrowseModeGrouping BrowseMode = "grouping"
	BrowseModePlaylist BrowseMode = "playlist"
)

type PlaybackState int

const (
	PlaybackStateStopped PlaybackState = iota
	PlaybackStatePlaying
	PlaybackStatePaused
)

type TrackEndReason int

const (
	TrackEndedEOF TrackEndReason = iota
	TrackEndedStopped
	TrackEndedError
)

type RepeatMode string

const (
	RepeatModeStopAtEnd   RepeatMode = "stop_at_end"
	RepeatModeRepeatQueue RepeatMode = "repeat_queue"
	RepeatModeRepeatTrack RepeatMode = "repeat_track"
	RepeatModeShuffle     RepeatMode = "shuffle"
)

type SortField string

const (
	SortFieldTitle       SortField = "title"
	SortFieldArtist      SortField = "artist"
	SortFieldAlbum       SortField = "album"
	SortFieldYear        SortField = "year"
	SortFieldGenre       SortField = "genre"
	SortFieldDuration    SortField = "duration"
	SortFieldFormat      SortField = "format"
	SortFieldSampleRate  SortField = "sample_rate"
	SortFieldBitDepth    SortField = "bit_depth"
	SortFieldBitrate     SortField = "bitrate"
	SortFieldTrackNumber SortField = "track_number"
	SortFieldFileSize    SortField = "file_size"
	SortFieldDateAdded   SortField = "date_added"
	SortFieldLabel       SortField = "label"
	SortFieldGrouping    SortField = "grouping"
)

type SortDirection string

const (
	SortDirectionAsc  SortDirection = "asc"
	SortDirectionDesc SortDirection = "desc"
)

type Pane int

const (
	PaneLeft Pane = iota
	PaneMiddle
	PaneRight
)

type AudioFormat string

const (
	AudioFormatFLAC AudioFormat = "FLAC"
	AudioFormatALAC AudioFormat = "ALAC"
	AudioFormatMP3  AudioFormat = "MP3"
	AudioFormatAIFF AudioFormat = "AIFF"
	AudioFormatWAV  AudioFormat = "WAV"
	AudioFormatOGG  AudioFormat = "OGG"
)

type Codec string

const (
	CodecFLAC   Codec = "flac"
	CodecALAC   Codec = "alac"
	CodecMP3    Codec = "mp3"
	CodecPCM    Codec = "pcm"
	CodecVorbis Codec = "vorbis"
)

type Artist struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	NameSort string `json:"name_sort"`
}

type Album struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	ArtistID    *int64 `json:"artist_id,omitempty"`
	AlbumArtist string `json:"album_artist,omitempty"`
	Year        int    `json:"year,omitempty"`
	Genre       string `json:"genre,omitempty"`
	Label       string `json:"label,omitempty"`
	Grouping    string `json:"grouping,omitempty"`
	TrackCount  int    `json:"track_count"`
	DiscCount   int    `json:"disc_count"`

	RGAlbumGain   *float64 `json:"rg_album_gain,omitempty"`
	RGAlbumPeak   *float64 `json:"rg_album_peak,omitempty"`
	R128AlbumGain *float64 `json:"r128_album_gain,omitempty"`
}

type Track struct {
	ID       int64 `json:"id"`
	AlbumID  int64 `json:"album_id"`
	ArtistID int64 `json:"artist_id"`

	FilePath      string `json:"file_path"`
	FileSizeBytes int64  `json:"file_size_bytes"`
	LastModified  int64  `json:"last_modified"`
	DateAdded     int64  `json:"date_added"`

	Title       string `json:"title"`
	Artist      string `json:"artist"`
	AlbumArtist string `json:"album_artist,omitempty"`
	Album       string `json:"album"`
	TrackNumber int    `json:"track_number,omitempty"`
	DiscNumber  int    `json:"disc_number,omitempty"`
	Year        int    `json:"year,omitempty"`
	Genre       string `json:"genre,omitempty"`
	Grouping    string `json:"grouping,omitempty"`
	Label       string `json:"label,omitempty"`

	DurationMs int    `json:"duration_ms"`
	Format     string `json:"format"`
	Codec      string `json:"codec"`
	Container  string `json:"container"`
	SampleRate int    `json:"sample_rate"`
	BitDepth   *int   `json:"bit_depth,omitempty"`
	Bitrate    int    `json:"bitrate"`

	RGTrackGain   *float64 `json:"rg_track_gain,omitempty"`
	RGTrackPeak   *float64 `json:"rg_track_peak,omitempty"`
	RGAlbumGain   *float64 `json:"rg_album_gain,omitempty"`
	RGAlbumPeak   *float64 `json:"rg_album_peak,omitempty"`
	R128TrackGain *float64 `json:"r128_track_gain,omitempty"`
	R128AlbumGain *float64 `json:"r128_album_gain,omitempty"`

	HasArtwork       bool   `json:"has_artwork"`
	ArtworkWidth     int    `json:"artwork_width,omitempty"`
	ArtworkHeight    int    `json:"artwork_height,omitempty"`
	ArtworkFormat    string `json:"artwork_format,omitempty"`
	ArtworkSizeBytes int64  `json:"artwork_size_bytes,omitempty"`
}

type RawTrackTags struct {
	Title       string
	Artist      string
	AlbumArtist string
	Album       string
	TrackNumber int
	DiscNumber  int
	Year        int
	Genre       string
	Grouping    string
	Label       string
	Composer    string

	Format    AudioFormat
	Codec     Codec
	Container string

	SampleRate int
	BitDepth   int
	Bitrate    int
	Channels   int
	DurationMs int

	RGTrackGain   float64
	RGTrackPeak   float64
	RGAlbumGain   float64
	RGAlbumPeak   float64
	R128TrackGain float64
	R128AlbumGain float64

	HasArtwork    bool
	ArtworkFormat string
	ArtworkWidth  int
	ArtworkHeight int
	ArtworkData   []byte
}

func SupportedExtensions() []string {
	return []string{".flac", ".alac", ".m4a", ".aiff", ".aif", ".mp3", ".wav", ".ogg"}
}

func IsSupportedExt(path string) bool {
	lower := strings.ToLower(path)
	for _, ext := range SupportedExtensions() {
		if len(lower) >= len(ext) && lower[len(lower)-len(ext):] == ext {
			return true
		}
	}
	return false
}

func FormatFromExtension(ext string) (AudioFormat, Codec, string) {
	switch ext {
	case ".flac":
		return AudioFormatFLAC, CodecFLAC, "FLAC"
	case ".m4a", ".alac":
		return AudioFormatALAC, CodecALAC, "MP4"
	case ".mp3":
		return AudioFormatMP3, CodecMP3, "MP3"
	case ".aiff", ".aif":
		return AudioFormatAIFF, CodecPCM, "AIFF"
	case ".wav":
		return AudioFormatWAV, CodecPCM, "WAV"
	case ".ogg":
		return AudioFormatOGG, CodecVorbis, "OGG"
	}
	return "", "", ""
}

func FormatContainer(format AudioFormat) string {
	switch format {
	case AudioFormatFLAC:
		return "FLAC"
	case AudioFormatALAC:
		return "MP4"
	case AudioFormatMP3:
		return "MP3"
	case AudioFormatAIFF:
		return "AIFF"
	case AudioFormatWAV:
		return "WAV"
	case AudioFormatOGG:
		return "OGG"
	}
	return ""
}
