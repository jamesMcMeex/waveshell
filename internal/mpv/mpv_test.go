package mpv

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jamesMcMeex/waveshell/internal/messages"
	"github.com/jamesMcMeex/waveshell/internal/model"
)

// mockServer listens on a Unix socket and responds to mpv IPC commands
// with configurable responses. It verifies that expected commands are received.
// See docs/MPV_IPC.md §7 for the pattern.
type mockServer struct {
	t         testing.TB
	path      string
	ln        net.Listener
	conn      net.Conn
	received  []map[string]any
	responses []map[string]any
	events    []map[string]any
}

// newMockServer creates a mock server on a temp socket path.
func newMockServer(t testing.TB) *mockServer {
	t.Helper()

	dir, err := os.MkdirTemp("", "ws")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "s.sock")

	ln, err := net.Listen("unix", path)
	require.NoError(t, err, "listen on test socket")

	s := &mockServer{
		t:    t,
		path: path,
		ln:   ln,
	}

	// Accept one connection in a goroutine
	go s.accept()

	// Give the goroutine time to call Accept
	time.Sleep(10 * time.Millisecond)

	return s
}

func (s *mockServer) accept() {
	conn, err := s.ln.Accept()
	if err != nil {
		return
	}
	s.conn = conn

	// Push initial events if configured
	for _, ev := range s.events {
		data, err := json.Marshal(ev)
		if err != nil {
			return
		}
		data = append(data, '\n')
		_, _ = conn.Write(data)
	}

	// Read commands and respond
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		lines := splitLines(buf[:n])
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			var req map[string]any
			if err := json.Unmarshal(line, &req); err != nil {
				continue
			}
			s.received = append(s.received, req)

			// Send response, echoing the request_id from the command
			if len(s.responses) > 0 {
				resp := make(map[string]any, len(s.responses[0])+1)
				for k, v := range s.responses[0] {
					resp[k] = v
				}
				if reqID, ok := req["request_id"]; ok {
					resp["request_id"] = reqID
				}
				s.responses = s.responses[1:]
				respData, _ := json.Marshal(resp)
				respData = append(respData, '\n')
				_, _ = conn.Write(respData)
			}
		}
	}
}

func (s *mockServer) Close() {
	if s.conn != nil {
		_ = s.conn.Close()
	}
	_ = s.ln.Close()
}

func (s *mockServer) NextCommand(n int) map[string]any {
	s.t.Helper()
	if n >= len(s.received) {
		s.t.Fatalf("NextCommand(%d): only %d commands received", n, len(s.received))
	}
	return s.received[n]
}

func (s *mockServer) CommandCount() int {
	return len(s.received)
}

func splitLines(b []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, ch := range b {
		if ch == '\n' {
			if i > start {
				lines = append(lines, b[start:i])
			}
			start = i + 1
		}
	}
	if start < len(b) {
		lines = append(lines, b[start:])
	}
	return lines
}

// testConn creates a Conn connected to the given socket, with reader/writer set up.
func testConn(t testing.TB, socketPath string) *Conn {
	t.Helper()
	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)

	var nextID atomic.Int64
	nextID.Store(1)

	return &Conn{
		conn:   conn,
		reader: bufio.NewReaderSize(conn, 4096),
		writer: bufio.NewWriter(conn),
		nextID: &nextID,
	}
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestPlayerInterface_compileCheck(t *testing.T) {
	var _ Player = (*Conn)(nil)
}

func TestConn_LoadFile(t *testing.T) {
	s := newMockServer(t)
	defer s.Close()

	s.responses = []map[string]any{
		{"error": "success", "request_id": 1},
	}

	p := testConn(t, s.path)
	defer func() { _ = p.conn.Close() }()

	err := p.LoadFile("/test/path.flac")
	assert.NoError(t, err)

	cmd := s.NextCommand(0)
	cmdArr := cmd["command"].([]any)
	assert.Equal(t, "loadfile", cmdArr[0])
	assert.Equal(t, "/test/path.flac", cmdArr[1])
	assert.Equal(t, "replace", cmdArr[2])
}

func TestConn_Play(t *testing.T) {
	s := newMockServer(t)
	defer s.Close()

	s.responses = []map[string]any{
		{"error": "success", "request_id": 1},
	}

	p := testConn(t, s.path)
	defer func() { _ = p.conn.Close() }()

	err := p.Play()
	assert.NoError(t, err)

	cmd := s.NextCommand(0)
	cmdArr := cmd["command"].([]any)
	assert.Equal(t, "set_property", cmdArr[0])
	assert.Equal(t, "pause", cmdArr[1])
	assert.Equal(t, false, cmdArr[2])
}

func TestConn_Pause(t *testing.T) {
	s := newMockServer(t)
	defer s.Close()

	s.responses = []map[string]any{
		{"error": "success", "request_id": 1},
	}

	p := testConn(t, s.path)
	defer func() { _ = p.conn.Close() }()

	err := p.Pause()
	assert.NoError(t, err)

	cmd := s.NextCommand(0)
	cmdArr := cmd["command"].([]any)
	assert.Equal(t, "set_property", cmdArr[0])
	assert.Equal(t, "pause", cmdArr[1])
	assert.Equal(t, true, cmdArr[2])
}

func TestConn_SeekRelative(t *testing.T) {
	s := newMockServer(t)
	defer s.Close()

	s.responses = []map[string]any{
		{"error": "success", "request_id": 1},
	}

	p := testConn(t, s.path)
	defer func() { _ = p.conn.Close() }()

	err := p.SeekRelative(15.5)
	assert.NoError(t, err)

	cmd := s.NextCommand(0)
	cmdArr := cmd["command"].([]any)
	assert.Equal(t, "seek", cmdArr[0])
	assert.Equal(t, 15.5, cmdArr[1])
	assert.Equal(t, "relative", cmdArr[2])
}

func TestConn_SeekAbsolute(t *testing.T) {
	s := newMockServer(t)
	defer s.Close()

	s.responses = []map[string]any{
		{"error": "success", "request_id": 1},
	}

	p := testConn(t, s.path)
	defer func() { _ = p.conn.Close() }()

	err := p.SeekAbsolute(120.0)
	assert.NoError(t, err)

	cmd := s.NextCommand(0)
	cmdArr := cmd["command"].([]any)
	assert.Equal(t, "seek", cmdArr[0])
	assert.Equal(t, 120.0, cmdArr[1])
	assert.Equal(t, "absolute", cmdArr[2])
}

func TestConn_SetVolume(t *testing.T) {
	s := newMockServer(t)
	defer s.Close()

	s.responses = []map[string]any{
		{"error": "success", "request_id": 1},
	}

	p := testConn(t, s.path)
	defer func() { _ = p.conn.Close() }()

	err := p.SetVolume(75)
	assert.NoError(t, err)

	cmd := s.NextCommand(0)
	cmdArr := cmd["command"].([]any)
	assert.Equal(t, "set_property", cmdArr[0])
	assert.Equal(t, "volume", cmdArr[1])
	assert.Equal(t, float64(75), cmdArr[2])
}

func TestConn_Stop(t *testing.T) {
	s := newMockServer(t)
	defer s.Close()

	s.responses = []map[string]any{
		{"error": "success", "request_id": 1},
	}

	p := testConn(t, s.path)
	defer func() { _ = p.conn.Close() }()

	err := p.Stop()
	assert.NoError(t, err)

	cmd := s.NextCommand(0)
	cmdArr := cmd["command"].([]any)
	assert.Equal(t, "stop", cmdArr[0])
}

func TestConn_SetVolume_clamping(t *testing.T) {
	s := newMockServer(t)
	defer s.Close()

	s.responses = []map[string]any{
		{"error": "success", "request_id": 1},
	}

	p := testConn(t, s.path)
	defer func() { _ = p.conn.Close() }()

	err := p.SetVolume(150)
	assert.NoError(t, err)

	cmd := s.NextCommand(0)
	cmdArr := cmd["command"].([]any)
	assert.Equal(t, float64(100), cmdArr[2])
}

func TestConn_multiCommand(t *testing.T) {
	s := newMockServer(t)
	defer s.Close()

	s.responses = []map[string]any{
		{"error": "success", "request_id": 1},
		{"error": "success", "request_id": 2},
		{"error": "success", "request_id": 3},
	}

	p := testConn(t, s.path)
	defer func() { _ = p.conn.Close() }()

	assert.NoError(t, p.LoadFile("/a.flac"))
	assert.NoError(t, p.Play())
	assert.NoError(t, p.SetVolume(50))

	assert.Equal(t, 3, s.CommandCount())
	assert.Equal(t, "loadfile", s.NextCommand(0)["command"].([]any)[0])
	assert.Equal(t, "set_property", s.NextCommand(1)["command"].([]any)[0])
	assert.Equal(t, "set_property", s.NextCommand(2)["command"].([]any)[0])
}

func TestConn_sendCommand_error(t *testing.T) {
	s := newMockServer(t)
	defer s.Close()

	s.responses = []map[string]any{
		{"error": "invalid parameter", "request_id": 1},
	}

	p := testConn(t, s.path)
	defer func() { _ = p.conn.Close() }()

	err := p.LoadFile("/bad/path.flac")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid parameter")
}

func TestEventDispatch_timePosition(t *testing.T) {
	c := &Conn{
		eventCh: make(chan tea.Msg, 10),
	}

	c.dispatchEvent(mpvEvent{
		Event: "property-change",
		ID:    1,
		Name:  "time-pos",
		Data:  42.5,
	})

	select {
	case msg := <-c.eventCh:
		m, ok := msg.(messages.TimePositionChangedMsg)
		assert.True(t, ok, "expected TimePositionChangedMsg")
		assert.Equal(t, 42.5, m.PositionSec)
	default:
		t.Fatal("expected event on channel")
	}
}

func TestEventDispatch_pause(t *testing.T) {
	c := &Conn{
		eventCh: make(chan tea.Msg, 10),
	}

	c.dispatchEvent(mpvEvent{
		Event: "property-change",
		ID:    2,
		Name:  "pause",
		Data:  true,
	})

	select {
	case msg := <-c.eventCh:
		m, ok := msg.(messages.PlaybackStateChangedMsg)
		assert.True(t, ok, "expected PlaybackStateChangedMsg")
		assert.Equal(t, model.PlaybackStatePaused, m.State)
	default:
		t.Fatal("expected event on channel")
	}
}

func TestEventDispatch_duration(t *testing.T) {
	c := &Conn{
		eventCh: make(chan tea.Msg, 10),
	}

	c.dispatchEvent(mpvEvent{
		Event: "property-change",
		ID:    3,
		Name:  "duration",
		Data:  180.0,
	})

	select {
	case msg := <-c.eventCh:
		m, ok := msg.(messages.DurationChangedMsg)
		assert.True(t, ok, "expected DurationChangedMsg")
		assert.Equal(t, 180.0, m.DurationSec)
	default:
		t.Fatal("expected event on channel")
	}
}

func TestEventDispatch_volume(t *testing.T) {
	c := &Conn{
		eventCh: make(chan tea.Msg, 10),
	}

	c.dispatchEvent(mpvEvent{
		Event: "property-change",
		ID:    4,
		Name:  "volume",
		Data:  float64(75),
	})

	select {
	case msg := <-c.eventCh:
		m, ok := msg.(messages.VolumeChangedMsg)
		assert.True(t, ok, "expected VolumeChangedMsg")
		assert.Equal(t, 75, m.Volume)
	default:
		t.Fatal("expected event on channel")
	}
}

func TestEventDispatch_endFile_eof(t *testing.T) {
	c := &Conn{
		eventCh: make(chan tea.Msg, 10),
	}

	c.dispatchEvent(mpvEvent{
		Event:  "end-file",
		Reason: "eof",
	})

	select {
	case msg := <-c.eventCh:
		m, ok := msg.(messages.TrackEndedMsg)
		assert.True(t, ok, "expected TrackEndedMsg")
		assert.Equal(t, model.TrackEndedEOF, m.Reason)
	default:
		t.Fatal("expected event on channel")
	}
}

func TestEventDispatch_endFile_stop(t *testing.T) {
	c := &Conn{
		eventCh: make(chan tea.Msg, 10),
	}

	c.dispatchEvent(mpvEvent{
		Event:  "end-file",
		Reason: "stop",
	})

	select {
	case msg := <-c.eventCh:
		m, ok := msg.(messages.TrackEndedMsg)
		assert.True(t, ok, "expected TrackEndedMsg")
		assert.Equal(t, model.TrackEndedStopped, m.Reason)
	default:
		t.Fatal("expected event on channel")
	}
}

func TestEventDispatch_endFile_error(t *testing.T) {
	c := &Conn{
		eventCh: make(chan tea.Msg, 10),
	}

	c.dispatchEvent(mpvEvent{
		Event:  "end-file",
		Reason: "error",
	})

	select {
	case msg := <-c.eventCh:
		m, ok := msg.(messages.TrackEndedMsg)
		assert.True(t, ok, "expected TrackEndedMsg")
		assert.Equal(t, model.TrackEndedError, m.Reason)
	default:
		t.Fatal("expected event on channel")
	}
}

func TestEventDispatch_startFile_noCrash(t *testing.T) {
	c := &Conn{
		eventCh: make(chan tea.Msg, 10),
	}

	// start-file should be silently ignored (no event dispatched)
	c.dispatchEvent(mpvEvent{
		Event: "start-file",
	})

	select {
	case <-c.eventCh:
		t.Fatal("expected no event for start-file")
	default:
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		input   any
		wantOk  bool
		wantVal float64
	}{
		{float64(42.5), true, 42.5},
		{int(42), true, 42},
		{int64(42), true, 42},
		{"42.5", false, 0},
		{nil, false, 0},
		{true, false, 0},
	}

	for _, tt := range tests {
		val, ok := toFloat64(tt.input)
		assert.Equal(t, tt.wantOk, ok, "toFloat64(%v)", tt.input)
		if ok {
			assert.Equal(t, tt.wantVal, val, "toFloat64(%v)", tt.input)
		}
	}
}

func TestMockServer_acceptAndRespond(t *testing.T) {
	s := newMockServer(t)
	defer s.Close()

	s.responses = []map[string]any{
		{"error": "success", "request_id": 1},
	}

	p := testConn(t, s.path)
	defer func() { _ = p.conn.Close() }()

	err := p.LoadFile("/test.flac")
	assert.NoError(t, err)

	assert.Equal(t, 1, s.CommandCount())
	cmd := s.NextCommand(0)
	cmdArr := cmd["command"].([]any)
	assert.Equal(t, "loadfile", cmdArr[0])
	assert.Equal(t, "/test.flac", cmdArr[1])
}

func TestMPVNotFound(t *testing.T) {
	t.Setenv("PATH", "") // ensure mpv is not found regardless of host installation
	_, _, err := Launch("/tmp/_not_used.sock")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mpv not found")
}

func TestLaunch_mpvPresent(t *testing.T) {
	mpvPath, err := exec.LookPath("mpv")
	if err != nil {
		t.Skip("mpv not available on this host")
	}
	t.Setenv("PATH", filepath.Dir(mpvPath))

	socketPath := filepath.Join(t.TempDir(), "ws.sock")
	conn, events, err := Launch(socketPath)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, events)

	err = conn.Play()
	assert.NoError(t, err)

	conn.Close()
	_, err = os.Stat(socketPath)
	assert.True(t, os.IsNotExist(err), "socket file should be cleaned up on Close")
}
