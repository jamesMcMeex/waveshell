// Package update defines the BubbleTea Model and its Init, Update, and View
// methods. It is the hub that all subsystems feed into — every Msg type is
// handled here and the full TUI is rendered from this package.
package update

import (
	"database/sql"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jamesMcMeex/waveshell/internal/config"
	"github.com/jamesMcMeex/waveshell/internal/model"
	"github.com/jamesMcMeex/waveshell/internal/mpv"
)

type Model struct {
	Library    LibraryState
	Player     PlayerState
	Queue      QueueState
	Search     SearchState
	UI         UIState
	Help       HelpState
	Config     *config.Config
	ConfigPath string
	DB         *sql.DB
	MPV        mpv.Player
	MPVErr     error
}

type LibraryState struct {
	Scanning      bool
	ScanProcessed int
	ScanSkipped   int
	ScanTotal     int
	ScanCurrent   string
	ScanComplete  bool

	Artists []model.Artist
	Albums  []model.Album
	Tracks  []model.Track

	TagSliceValues []string

	SelectedArtistID int64
	SelectedAlbumID  int64
	SelectedTagKey   string
}

type PlayerState struct {
	State              model.PlaybackState
	CurrentTrack       *model.Track
	PositionSec        float64
	DisplayPositionSec float64
	DurationSec        float64
	Volume             int
	Events             <-chan tea.Msg
	MPVReady           bool
}

type QueueState struct {
	Tracks       []model.Track
	CurrentIndex int
	Mode         model.RepeatMode
}

type SearchState struct {
	Active  bool
	Query   string
	Results any
}

type ActiveOverlay int

const (
	OverlayNone ActiveOverlay = iota
	OverlayHelp
)

type UIState struct {
	Width  int
	Height int

	BrowseMode    model.BrowseMode
	ActivePane    model.Pane
	ActiveOverlay ActiveOverlay

	LeftCursor   int
	LeftOffset   int
	MiddleCursor int
	MiddleOffset int
	RightCursor  int
	RightOffset  int

	ShowBrowsePicker   bool
	BrowsePickerCursor int
}

type HelpState struct {
	Active       bool
	ScrollOffset int
}
