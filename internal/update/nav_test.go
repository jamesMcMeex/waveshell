package update

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/jamesMcMeex/waveshell/internal/config"
	"github.com/jamesMcMeex/waveshell/internal/messages"
	"github.com/jamesMcMeex/waveshell/internal/model"
)

func assertUpdate(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	r, _ := m.Update(msg)
	return r.(Model)
}

func keyRune(r rune) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func keyPress(k tea.KeyType) tea.Msg {
	return tea.KeyMsg{Type: k}
}

func TestFormatBadge_lossless(t *testing.T) {
	bd := 24
	track := model.Track{Format: "FLAC", SampleRate: 44100, BitDepth: &bd, Bitrate: 800}
	badge := FormatBadge(track)
	assert.Equal(t, "FLAC 44.1k 24bit 800kbps", badge)
}

func TestFormatBadge_lossy(t *testing.T) {
	track := model.Track{Format: "MP3", SampleRate: 44100, Bitrate: 320}
	badge := FormatBadge(track)
	assert.Equal(t, "MP3 44.1k 320kbps", badge)
}

func TestFormatBadge_alac(t *testing.T) {
	bd := 16
	track := model.Track{Format: "ALAC", SampleRate: 44100, BitDepth: &bd, Bitrate: 600}
	badge := FormatBadge(track)
	assert.Equal(t, "ALAC 44.1k 16bit 600kbps", badge)
}

func TestFormatBadge_highRes(t *testing.T) {
	bd := 24
	track := model.Track{Format: "FLAC", SampleRate: 96000, BitDepth: &bd, Bitrate: 2000}
	badge := FormatBadge(track)
	assert.Equal(t, "FLAC 96.0k 24bit 2000kbps", badge)
}

func TestFormatBadge_ogg(t *testing.T) {
	track := model.Track{Format: "OGG", SampleRate: 44100, Bitrate: 192}
	badge := FormatBadge(track)
	assert.Equal(t, "OGG 44.1k 192kbps", badge)
}

func TestPaneFocus_tabCyclesLeftToMiddleToRight(t *testing.T) {
	m := Model{UI: UIState{ActivePane: model.PaneLeft}}
	result, _ := m.Update(keyPress(tea.KeyTab))
	assert.Equal(t, model.PaneMiddle, result.(Model).UI.ActivePane)

	result, _ = result.Update(keyPress(tea.KeyTab))
	assert.Equal(t, model.PaneRight, result.(Model).UI.ActivePane)

	result, _ = result.Update(keyPress(tea.KeyTab))
	assert.Equal(t, model.PaneLeft, result.(Model).UI.ActivePane)
}

func TestPaneFocus_shiftTabCyclesRightToMiddleToLeft(t *testing.T) {
	m := Model{UI: UIState{ActivePane: model.PaneRight}}
	result, _ := m.Update(keyPress(tea.KeyShiftTab))
	assert.Equal(t, model.PaneMiddle, result.(Model).UI.ActivePane)

	result, _ = result.Update(keyPress(tea.KeyShiftTab))
	assert.Equal(t, model.PaneLeft, result.(Model).UI.ActivePane)

	result, _ = result.Update(keyPress(tea.KeyShiftTab))
	assert.Equal(t, model.PaneRight, result.(Model).UI.ActivePane)
}

func TestPaneFocus_lMovesRight(t *testing.T) {
	m := Model{UI: UIState{ActivePane: model.PaneLeft}}
	result, _ := m.Update(keyRune('l'))
	assert.Equal(t, model.PaneMiddle, result.(Model).UI.ActivePane)
}

func TestPaneFocus_jMovesLeft(t *testing.T) {
	m := Model{UI: UIState{ActivePane: model.PaneRight}}
	result, _ := m.Update(keyRune('j'))
	assert.Equal(t, model.PaneMiddle, result.(Model).UI.ActivePane)
}

func TestPaneFocus_leftArrowMovesLeft(t *testing.T) {
	m := Model{UI: UIState{ActivePane: model.PaneMiddle}}
	result, _ := m.Update(keyPress(tea.KeyLeft))
	assert.Equal(t, model.PaneLeft, result.(Model).UI.ActivePane)
}

func TestPaneFocus_rightArrowMovesRight(t *testing.T) {
	m := Model{UI: UIState{ActivePane: model.PaneMiddle}}
	result, _ := m.Update(keyPress(tea.KeyRight))
	assert.Equal(t, model.PaneRight, result.(Model).UI.ActivePane)
}

func TestCursorDown_kMovesCursorDown(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "Artist A"},
				{ID: 2, Name: "Artist B"},
				{ID: 3, Name: "Artist C"},
			},
		},
		UI: UIState{
			ActivePane: model.PaneLeft,
		},
	}

	result, _ := m.Update(keyRune('k'))
	assert.Equal(t, 1, result.(Model).UI.LeftCursor)

	result, _ = result.Update(keyRune('k'))
	assert.Equal(t, 2, result.(Model).UI.LeftCursor)
}

func TestCursorDown_kClampsAtEnd(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "Artist A"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft},
	}

	result, _ := m.Update(keyRune('k'))
	assert.Equal(t, 0, result.(Model).UI.LeftCursor)
}

func TestCursorDown_downArrowWorks(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "Artist A"},
				{ID: 2, Name: "Artist B"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft},
	}

	result, _ := m.Update(keyPress(tea.KeyDown))
	assert.Equal(t, 1, result.(Model).UI.LeftCursor)
}

func TestCursorUp_iMovesCursorUp(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "A"},
				{ID: 2, Name: "B"},
				{ID: 3, Name: "C"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft, LeftCursor: 2},
	}

	result, _ := m.Update(keyRune('i'))
	assert.Equal(t, 1, result.(Model).UI.LeftCursor)
}

func TestCursorUp_iClampsAtStart(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "A"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft, LeftCursor: 0},
	}

	result, _ := m.Update(keyRune('i'))
	assert.Equal(t, 0, result.(Model).UI.LeftCursor)
}

func TestCursorUp_upArrowWorks(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "A"},
				{ID: 2, Name: "B"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft, LeftCursor: 1},
	}

	result, _ := m.Update(keyPress(tea.KeyUp))
	assert.Equal(t, 0, result.(Model).UI.LeftCursor)
}

func TestJumpToTop_tMovesCursorToZero(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "A"},
				{ID: 2, Name: "B"},
				{ID: 2, Name: "C"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft, LeftCursor: 2},
	}

	result, _ := m.Update(keyRune('t'))
	assert.Equal(t, 0, result.(Model).UI.LeftCursor)
}

func TestJumpToBottom_gMovesCursorToEnd(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "A"},
				{ID: 2, Name: "B"},
				{ID: 3, Name: "C"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft},
	}

	result, _ := m.Update(keyRune('g'))
	assert.Equal(t, 2, result.(Model).UI.LeftCursor)
}

func TestJumpToBottom_endKeyMovesCursorToEnd(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "A"},
				{ID: 2, Name: "B"},
				{ID: 3, Name: "C"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft},
	}

	result, _ := m.Update(keyPress(tea.KeyEnd))
	assert.Equal(t, 2, result.(Model).UI.LeftCursor)
}

func TestJumpToBottom_GNowDoesLetterJump(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "Goldie"},
				{ID: 2, Name: "Boards of Canada"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft, Width: 80, Height: 40},
	}

	result, _ := m.Update(keyRune('G'))
	assert.Equal(t, 0, result.(Model).UI.LeftCursor)
}

func TestHomeKeyMovesToTop(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "A"},
				{ID: 2, Name: "B"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft, LeftCursor: 1},
	}

	result, _ := m.Update(keyPress(tea.KeyHome))
	assert.Equal(t, 0, result.(Model).UI.LeftCursor)
}

func TestEndKeyMovesToBottom(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "A"},
				{ID: 2, Name: "B"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft},
	}

	result, _ := m.Update(keyPress(tea.KeyEnd))
	assert.Equal(t, 1, result.(Model).UI.LeftCursor)
}

func TestLetterJump_jumpsToMatchingArtist(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "Aphex Twin"},
				{ID: 2, Name: "Boards of Canada"},
				{ID: 3, Name: "Goldie"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft, Width: 80, Height: 40},
	}

	result, _ := m.Update(keyRune('G'))
	assert.Equal(t, 2, result.(Model).UI.LeftCursor)
}

func TestLetterJump_caseInsensitive(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "amon tobin"},
				{ID: 2, Name: "aphex twin"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft, Width: 80, Height: 40},
	}

	result, _ := m.Update(keyRune('A'))
	assert.Equal(t, 0, result.(Model).UI.LeftCursor)
}

func TestLetterJump_noMatchLeavesCursor(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Artists: []model.Artist{
				{ID: 1, Name: "Aphex Twin"},
				{ID: 2, Name: "Boards of Canada"},
			},
		},
		UI: UIState{ActivePane: model.PaneLeft, Width: 80, Height: 40, LeftCursor: 1},
	}

	result, _ := m.Update(keyRune('Z'))
	assert.Equal(t, 1, result.(Model).UI.LeftCursor)
}

func TestLetterJump_middlePane(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Albums: []model.Album{
				{ID: 1, Title: "Come to Daddy"},
				{ID: 2, Title: "Drukqs"},
				{ID: 3, Title: "Selected Ambient Works"},
			},
		},
		UI: UIState{ActivePane: model.PaneMiddle, Width: 80, Height: 40},
	}

	result, _ := m.Update(keyRune('S'))
	assert.Equal(t, 2, result.(Model).UI.MiddleCursor)
}

func TestLetterJump_tracksPane(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Tracks: []model.Track{
				{ID: 1, Title: "Flim"},
				{ID: 2, Title: "Girl/Boy Song"},
				{ID: 3, Title: "Windowlicker"},
			},
		},
		UI: UIState{ActivePane: model.PaneRight, Width: 80, Height: 40},
	}

	result, _ := m.Update(keyRune('W'))
	assert.Equal(t, 2, result.(Model).UI.RightCursor)
}

func TestCtrlD_scrollsHalfPage(t *testing.T) {
	artists := make([]model.Artist, 50)
	for i := range artists {
		artists[i] = model.Artist{ID: int64(i), Name: fmt.Sprintf("Artist %d", i)}
	}

	m := Model{
		Library: LibraryState{Artists: artists},
		UI:      UIState{ActivePane: model.PaneLeft, Width: 80, Height: 30},
	}

	result, _ := m.Update(keyPress(tea.KeyCtrlD))

	updated := result.(Model)
	assert.Greater(t, updated.UI.LeftCursor, 0)
}

func TestCtrlU_scrollsHalfPageUp(t *testing.T) {
	artists := make([]model.Artist, 50)
	for i := range artists {
		artists[i] = model.Artist{ID: int64(i), Name: fmt.Sprintf("Artist %d", i)}
	}

	m := Model{
		Library: LibraryState{Artists: artists},
		UI:      UIState{ActivePane: model.PaneLeft, Width: 80, Height: 30, LeftCursor: 30, LeftOffset: 20},
	}

	result, _ := m.Update(keyPress(tea.KeyCtrlU))

	updated := result.(Model)
	assert.Less(t, updated.UI.LeftCursor, 30)
}

func TestThemeResolution_builtinSlate(t *testing.T) {
	cfg := config.Default()
	th := ResolveTheme(&cfg)
	assert.Contains(t, string(th.BG), "#")
	assert.Contains(t, string(th.FG), "#")
	assert.Contains(t, string(th.Accent), "#")
	assert.Contains(t, string(th.Muted), "#")
}

func TestThemeResolution_gameboyPreset(t *testing.T) {
	cfg := config.Default()
	cfg.UI.Theme = "gameboy"
	th := ResolveTheme(&cfg)
	assert.Equal(t, "#0f380f", string(th.BG))
	assert.Equal(t, "#9bbc0f", string(th.FG))
}

func TestBrowseModePicker_openWithB(t *testing.T) {
	m := Model{
		UI: UIState{ActivePane: model.PaneLeft},
	}

	result, _ := m.Update(keyRune('b'))
	updated := result.(Model)
	assert.True(t, updated.UI.ShowBrowsePicker)
	assert.Equal(t, 0, updated.UI.BrowsePickerCursor)
}

func TestBrowseModePicker_escCloses(t *testing.T) {
	m := Model{
		UI: UIState{
			ActivePane:       model.PaneLeft,
			ShowBrowsePicker: true,
		},
	}

	result, _ := m.Update(keyPress(tea.KeyEsc))
	assert.False(t, result.(Model).UI.ShowBrowsePicker)
}

func TestBrowseModePicker_selectMode(t *testing.T) {
	m := Model{
		UI: UIState{
			ShowBrowsePicker:   true,
			BrowsePickerCursor: 1, // Label mode
			BrowseMode:         model.BrowseModeArtist,
		},
	}

	result, _ := m.Update(keyPress(tea.KeyEnter))
	updated := result.(Model)
	assert.Equal(t, model.BrowseModeLabel, updated.UI.BrowseMode)
	assert.False(t, updated.UI.ShowBrowsePicker)
}

func TestBrowseModePicker_sameModeIsNoop(t *testing.T) {
	m := Model{
		UI: UIState{
			ShowBrowsePicker:   true,
			BrowsePickerCursor: 0, // Artist mode
			BrowseMode:         model.BrowseModeArtist,
		},
	}

	result, _ := m.Update(keyPress(tea.KeyEnter))
	updated := result.(Model)
	assert.Equal(t, model.BrowseModeArtist, updated.UI.BrowseMode)
	assert.False(t, updated.UI.ShowBrowsePicker)
}

func TestBrowsePicker_iNavigatesUp(t *testing.T) {
	m := Model{
		UI: UIState{
			ShowBrowsePicker:   true,
			BrowsePickerCursor: 2,
		},
	}

	result, _ := m.Update(keyRune('i'))
	assert.Equal(t, 1, result.(Model).UI.BrowsePickerCursor)
}

func TestBrowsePicker_kNavigatesDown(t *testing.T) {
	m := Model{
		UI: UIState{
			ShowBrowsePicker:   true,
			BrowsePickerCursor: 1,
		},
	}

	result, _ := m.Update(keyRune('k'))
	assert.Equal(t, 2, result.(Model).UI.BrowsePickerCursor)
}

func TestHelpOverlay_hOpensHelp(t *testing.T) {
	m := Model{
		UI: UIState{Width: 80, Height: 40},
	}

	result, _ := m.Update(keyRune('h'))
	updated := result.(Model)
	assert.Equal(t, OverlayHelp, updated.UI.ActiveOverlay)
	assert.True(t, updated.Help.Active)
}

func TestHelpOverlay_escCloses(t *testing.T) {
	m := Model{
		UI: UIState{
			Width:         80,
			Height:        40,
			ActiveOverlay: OverlayHelp,
		},
		Help: HelpState{Active: true},
	}

	result, _ := m.Update(keyPress(tea.KeyEsc))
	updated := result.(Model)
	assert.Equal(t, OverlayNone, updated.UI.ActiveOverlay)
	assert.False(t, updated.Help.Active)
}

func TestHelpOverlay_hClosesWhenActive(t *testing.T) {
	m := Model{
		UI: UIState{
			Width:         80,
			Height:        40,
			ActiveOverlay: OverlayHelp,
		},
		Help: HelpState{Active: true},
	}

	result, _ := m.Update(keyRune('h'))
	assert.Equal(t, OverlayNone, result.(Model).UI.ActiveOverlay)
}

func TestMiddlePaneCursor_navigatesIndependently(t *testing.T) {
	albums := []model.Album{
		{ID: 1, Title: "Album One"},
		{ID: 2, Title: "Album Two"},
		{ID: 3, Title: "Album Three"},
	}
	m := Model{
		Library: LibraryState{Albums: albums},
		UI:      UIState{ActivePane: model.PaneMiddle},
	}

	result, _ := m.Update(keyRune('k'))
	assert.Equal(t, 1, result.(Model).UI.MiddleCursor)

	result, _ = result.Update(keyRune('i'))
	assert.Equal(t, 0, result.(Model).UI.MiddleCursor)
}

func TestRightPaneCursor_navigatesIndependently(t *testing.T) {
	tracks := []model.Track{
		{ID: 1, Title: "Track One"},
		{ID: 2, Title: "Track Two"},
	}
	m := Model{
		Library: LibraryState{Tracks: tracks},
		UI:      UIState{ActivePane: model.PaneRight},
	}

	result, _ := m.Update(keyRune('k'))
	assert.Equal(t, 1, result.(Model).UI.RightCursor)
}

func TestEscQuit_doesNotQuit(t *testing.T) {
	m := Model{}
	result, cmd := m.Update(keyPress(tea.KeyEsc))
	assert.Nil(t, cmd)
	assert.Equal(t, m.UI.BrowseMode, result.(Model).UI.BrowseMode)
}

func TestQuit_qReturnsQuitCmd(t *testing.T) {
	m := Model{}
	_, cmd := m.Update(keyRune('q'))
	assert.NotNil(t, cmd)
	// tea.Quit is a special cmd
}

func TestDBResult_clearsRightPanes(t *testing.T) {
	m := Model{
		Library: LibraryState{
			Albums:  []model.Album{{ID: 1, Title: "Old Album"}},
			Tracks:  []model.Track{{ID: 1, Title: "Old Track"}},
			Artists: []model.Artist{{ID: 1, Name: "Old Artist"}},
		},
	}

	result, _ := m.Update(messages.ArtistListResultMsg{
		Artists: []model.Artist{{ID: 2, Name: "New Artist"}},
	})
	updated := result.(Model)
	assert.Len(t, updated.Library.Artists, 1)
	assert.Equal(t, "New Artist", updated.Library.Artists[0].Name)
	assert.Empty(t, updated.Library.Albums)
	assert.Empty(t, updated.Library.Tracks)
}

func TestThemeCycle_cyclesThroughBuiltins(t *testing.T) {
	cfg := config.Default()
	cfg.UI.Theme = "slate"
	m := Model{Config: &cfg}

	m = assertUpdate(t, m, keyRune('c'))
	assert.Equal(t, "phosphor", m.Config.UI.Theme)

	m = assertUpdate(t, m, keyRune('c'))
	assert.Equal(t, "amber", m.Config.UI.Theme)

	m = assertUpdate(t, m, keyRune('c'))
	assert.Equal(t, "gameboy", m.Config.UI.Theme)

	m = assertUpdate(t, m, keyRune('c'))
	assert.Equal(t, "slate", m.Config.UI.Theme)
}

func TestThemeCycle_unknownThemeDefaultsToSlate(t *testing.T) {
	cfg := config.Default()
	cfg.UI.Theme = "nonexistent"
	m := Model{Config: &cfg}

	m = assertUpdate(t, m, keyRune('c'))
	assert.Equal(t, "slate", m.Config.UI.Theme)
}

func TestThemeCycle_nilConfigIsNoop(t *testing.T) {
	m := Model{Config: nil}
	r, cmd := m.Update(keyRune('c'))
	assert.Nil(t, cmd)
	assert.Nil(t, r.(Model).Config)
}

func TestNextTheme_cyclesCorrectly(t *testing.T) {
	assert.Equal(t, "phosphor", nextTheme("slate"))
	assert.Equal(t, "amber", nextTheme("phosphor"))
	assert.Equal(t, "gameboy", nextTheme("amber"))
	assert.Equal(t, "slate", nextTheme("gameboy"))
	assert.Equal(t, "slate", nextTheme(""))
	assert.Equal(t, "slate", nextTheme("unknown"))
}

func TestFormatDuration(t *testing.T) {
	assert.Equal(t, "0:05", formatDuration(5000))
	assert.Equal(t, "1:00", formatDuration(60000))
	assert.Equal(t, "1:30", formatDuration(90000))
	assert.Equal(t, "0:00", formatDuration(0))
}

func TestTrackColumnValue(t *testing.T) {
	bd := 24
	track := model.Track{
		TrackNumber: 1,
		Title:       "Test Track",
		DurationMs:  5000,
		Format:      "FLAC",
		SampleRate:  44100,
		BitDepth:    &bd,
		Bitrate:     800,
		Artist:      "Test Artist",
		Album:       "Test Album",
		Year:        2024,
		Genre:       "Electronic",
	}
	assert.Equal(t, "01", trackColumnValue(track, "track_number"))
	assert.Equal(t, "Test Track", trackColumnValue(track, "title"))
	assert.Equal(t, "0:05", trackColumnValue(track, "duration"))
	assert.Equal(t, "2024", trackColumnValue(track, "year"))
	assert.Equal(t, "Electronic", trackColumnValue(track, "genre"))
}

func TestTrackColumnValue_zeroTrackNumber(t *testing.T) {
	track := model.Track{}
	assert.Equal(t, "--", trackColumnValue(track, "track_number"))
}
