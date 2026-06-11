package messages

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanStartedMsg_isMsg(t *testing.T) {
	var msg tea.Msg = ScanStartedMsg{}
	_ = msg
}

func TestScanProgressMsg_isMsg(t *testing.T) {
	var msg tea.Msg = ScanProgressMsg{}
	_ = msg
}

func TestScanFileErrorMsg_isMsg(t *testing.T) {
	var msg tea.Msg = ScanFileErrorMsg{}
	_ = msg
}

func TestScanCompleteMsg_isMsg(t *testing.T) {
	var msg tea.Msg = ScanCompleteMsg{}
	_ = msg
}

func TestDBErrorMsg_isMsg(t *testing.T) {
	var msg tea.Msg = DBErrorMsg{}
	_ = msg
}

func TestTickMsg_isMsg(t *testing.T) {
	var msg tea.Msg = TickMsg{}
	_ = msg
}

func TestArtistListResultMsg_isMsg(t *testing.T) {
	var msg tea.Msg = ArtistListResultMsg{}
	_ = msg
}

func TestTagSliceResultMsg_isMsg(t *testing.T) {
	var msg tea.Msg = TagSliceResultMsg{}
	_ = msg
}

func TestAlbumListResultMsg_isMsg(t *testing.T) {
	var msg tea.Msg = AlbumListResultMsg{}
	_ = msg
}

func TestTrackListResultMsg_isMsg(t *testing.T) {
	var msg tea.Msg = TrackListResultMsg{}
	_ = msg
}

func TestTickCmd_returnsTickMsg(t *testing.T) {
	cmd := TickCmd()
	require.NotNil(t, cmd)

	msg := cmd()
	require.NotNil(t, msg)

	tick, ok := msg.(TickMsg)
	require.True(t, ok, "expected TickMsg, got %T", msg)

	assert.False(t, tick.Time.IsZero(), "TickMsg.Time should not be zero")
}

func TestTickCmd_timeIsRecent(t *testing.T) {
	before := time.Now()
	cmd := TickCmd()
	msg := cmd()
	tick := msg.(TickMsg)
	assert.WithinDuration(t, before, tick.Time, time.Second+100*time.Millisecond)
}

func TestScanProgressMsg_TotalSentinel(t *testing.T) {
	msg := ScanProgressMsg{Total: -1}
	assert.Equal(t, -1, msg.Total, "Total == -1 means walk in progress")
}
