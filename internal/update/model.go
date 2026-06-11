// Package update defines the BubbleTea Model and its Init, Update, and View
// methods. It is the hub that all subsystems feed into — every Msg type is
// handled here and the full TUI is rendered from this package.
package update

import (
	"database/sql"

	"github.com/jamesMcMeex/waveshell/internal/config"
	"github.com/jamesMcMeex/waveshell/internal/model"
)

type Model struct {
	Library LibraryState
	Player  PlayerState
	Queue   QueueState
	Search  SearchState
	UI      UIState
	Config  *config.Config
	DB      *sql.DB
}

type LibraryState struct {
	Scanning      bool
	ScanProcessed int
	ScanSkipped   int
	ScanTotal     int
	ScanCurrent   string
	ScanComplete  bool
	Tracks        []model.Track
	Artists       []model.Artist
	Albums        []model.Album
}

type PlayerState struct {
	State              model.PlaybackState
	CurrentTrack       *model.Track
	PositionSec        float64
	DisplayPositionSec float64
	DurationSec        float64
	Volume             int
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

type UIState struct {
	Width  int
	Height int
}
