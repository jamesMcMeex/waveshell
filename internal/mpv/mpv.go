// Package mpv provides a Player interface and its mpv-backed implementation
// for controlling audio playback via the mpv JSON IPC protocol over a Unix
// domain socket. See docs/MPV_IPC.md for the full protocol reference.
package mpv

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jamesMcMeex/waveshell/internal/messages"
	"github.com/jamesMcMeex/waveshell/internal/model"
)

// Player is the interface that internal/update depends on for playback control.
// Property observation events arrive via the subscription channel (see events.go),
// not through this interface.
type Player interface {
	LoadFile(path string) error
	Play() error
	Pause() error
	Stop() error
	SeekRelative(seconds float64) error
	SeekAbsolute(seconds float64) error
	SetVolume(vol int) error
}

// Conn manages the mpv subprocess and its IPC socket connection.
type Conn struct {
	cmd        *exec.Cmd
	conn       net.Conn
	reader     *bufio.Reader
	writer     *bufio.Writer
	nextID     *atomic.Int64
	eventCh    chan tea.Msg
	socketPath string
	pending    map[int64]chan mpvResponse
	pendingMu  sync.Mutex
	closeOnce  sync.Once
	done       chan struct{}
}

// Launch starts an mpv subprocess, connects to its Unix socket, registers
// property observations, and starts the event loop goroutine. It blocks
// until the socket is ready (up to 5 seconds). Returns the Conn and the
// event channel for SubscribeCmd.
func Launch(socketPath string) (*Conn, <-chan tea.Msg, error) {
	// Check mpv is available first
	mpvPath, err := exec.LookPath("mpv")
	if err != nil {
		return nil, nil, fmt.Errorf("mpv not found on PATH: %w", err)
	}

	// Clean up any stale socket file
	_ = os.Remove(socketPath)

	socketDir := filepath.Dir(socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("create socket dir: %w", err)
	}

	cmd := exec.Command(mpvPath,
		"--input-ipc-server="+socketPath,
		"--idle=yes",
		"--no-terminal",
		"--no-config",
		"--no-video",
	)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("start mpv: %w", err)
	}

	// Wait for the socket to appear (mpv creates it asynchronously)
	var conn net.Conn
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err = net.DialTimeout("unix", socketPath, 500*time.Millisecond)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if conn == nil {
		stderrMsg := strings.TrimSpace(stderrBuf.String())
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = os.Remove(socketPath)
		if stderrMsg != "" {
			return nil, nil, fmt.Errorf("mpv socket not ready within 5s: %s: %w", stderrMsg, err)
		}
		return nil, nil, fmt.Errorf("mpv socket not ready within 5s: %w", err)
	}

	eventCh := make(chan tea.Msg, 64)
	var nextID atomic.Int64
	c := &Conn{
		cmd:        cmd,
		conn:       conn,
		reader:     bufio.NewReaderSize(conn, 4096),
		writer:     bufio.NewWriter(conn),
		nextID:     &nextID,
		eventCh:    eventCh,
		socketPath: socketPath,
		pending:    make(map[int64]chan mpvResponse),
		done:       make(chan struct{}),
	}

	// Start the event loop goroutine first so it can route command responses
	go c.eventLoop()

	// Register property observations
	if err := c.observeProperties(); err != nil {
		c.Close()
		_ = os.Remove(socketPath)
		stderrMsg := strings.TrimSpace(stderrBuf.String())
		if stderrMsg != "" {
			return nil, nil, fmt.Errorf("observe properties: %s: %w", stderrMsg, err)
		}
		return nil, nil, fmt.Errorf("observe properties: %w", err)
	}

	slog.Info("mpv launched", "socket", socketPath, "pid", cmd.Process.Pid)
	return c, eventCh, nil
}

// Close terminates the mpv subprocess, removes the socket file, and cleans up.
func (c *Conn) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		if c.conn != nil {
			_ = c.conn.Close()
		}
		if c.cmd != nil && c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
			_ = c.cmd.Wait()
		}
		_ = os.Remove(c.socketPath)
	})
}

// LoadFile sends a loadfile command to mpv, replacing the current file.
func (c *Conn) LoadFile(path string) error {
	return c.sendCommand("loadfile", path, "replace")
}

// Play resumes playback (sets pause to false).
func (c *Conn) Play() error {
	return c.sendCommand("set_property", "pause", false)
}

// Pause pauses playback (sets pause to true).
func (c *Conn) Pause() error {
	return c.sendCommand("set_property", "pause", true)
}

// Stop stops playback and unloads the current file.
func (c *Conn) Stop() error {
	return c.sendCommand("stop")
}

// SeekRelative seeks by the given number of seconds (positive or negative).
func (c *Conn) SeekRelative(seconds float64) error {
	return c.sendCommand("seek", seconds, "relative")
}

// SeekAbsolute seeks to an absolute position in seconds from the start.
func (c *Conn) SeekAbsolute(seconds float64) error {
	return c.sendCommand("seek", seconds, "absolute")
}

// SetVolume sets the volume (0–100).
func (c *Conn) SetVolume(vol int) error {
	if vol < 0 {
		vol = 0
	}
	if vol > 100 {
		vol = 100
	}
	return c.sendCommand("set_property", "volume", vol)
}

// ── internal helpers ──────────────────────────────────────────────────────────

type mpvRequest struct {
	Command   []any `json:"command"`
	RequestID int64 `json:"request_id"`
}

type mpvResponse struct {
	Error     string `json:"error"`
	Data      any    `json:"data,omitempty"`
	RequestID int64  `json:"request_id"`
}

type mpvEvent struct {
	Event     string `json:"event"`
	ID        int    `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Data      any    `json:"data,omitempty"`
	Reason    string `json:"reason,omitempty"`
	FileError string `json:"file_error,omitempty"`
}

func (c *Conn) sendCommand(args ...any) error {
	id := c.nextID.Add(1)
	req := mpvRequest{
		Command:   args,
		RequestID: id,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal command: %w", err)
	}

	data = append(data, '\n')

	c.pendingMu.Lock()
	hasPending := c.pending != nil
	c.pendingMu.Unlock()

	if hasPending {
		return c.sendCommandAsync(id, data)
	}
	return c.sendCommandSync(id, data)
}

func (c *Conn) sendCommandSync(id int64, data []byte) error {
	if _, err := c.writer.Write(data); err != nil {
		return fmt.Errorf("write command: %w", err)
	}
	if err := c.writer.Flush(); err != nil {
		return fmt.Errorf("flush command: %w", err)
	}

	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}

		var ev mpvEvent
		if err := json.Unmarshal(line, &ev); err == nil && ev.Event != "" {
			c.dispatchEvent(ev)
			continue
		}

		var resp mpvResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}

		if resp.RequestID != id {
			continue
		}

		if resp.Error != "success" {
			return fmt.Errorf("mpv error: %s", resp.Error)
		}
		return nil
	}
}

func (c *Conn) sendCommandAsync(id int64, data []byte) error {
	ch := make(chan mpvResponse, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	if _, err := c.writer.Write(data); err != nil {
		return fmt.Errorf("write command: %w", err)
	}
	if err := c.writer.Flush(); err != nil {
		return fmt.Errorf("flush command: %w", err)
	}

	select {
	case resp := <-ch:
		if resp.Error != "success" {
			return fmt.Errorf("mpv error: %s", resp.Error)
		}
		return nil
	case <-c.done:
		return fmt.Errorf("connection closed")
	}
}

func (c *Conn) observeProperties() error {
	observations := []struct {
		id   int
		prop string
	}{
		{1, "time-pos"},
		{2, "pause"},
		{3, "duration"},
		{4, "volume"},
	}

	for _, obs := range observations {
		if err := c.sendCommand("observe_property", obs.id, obs.prop); err != nil {
			return fmt.Errorf("observe %s: %w", obs.prop, err)
		}
	}
	return nil
}

func (c *Conn) eventLoop() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("mpv event loop panicked", "recover", r)
		}
	}()

	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			select {
			case <-c.done:
				return
			default:
			}
			c.eventCh <- messages.MPVConnectionLostMsg{Err: err}
			return
		}

		var ev mpvEvent
		if err := json.Unmarshal(line, &ev); err == nil && ev.Event != "" {
			c.dispatchEvent(ev)
			continue
		}

		var resp mpvResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			slog.Debug("mpv unparsed", "error", err, "line", string(line))
			continue
		}

		if resp.RequestID == 0 {
			continue
		}

		c.pendingMu.Lock()
		ch, ok := c.pending[resp.RequestID]
		c.pendingMu.Unlock()
		if ok {
			select {
			case ch <- resp:
			default:
			}
		}
	}
}

func (c *Conn) dispatchEvent(ev mpvEvent) {
	if c.eventCh == nil {
		return
	}
	switch ev.Event {
	case "property-change":
		c.handlePropertyChange(ev)
	case "end-file":
		c.handleEndFile(ev)
	case "start-file":
		// No action needed currently
	default:
		slog.Debug("mpv unhandled event", "event", ev.Event)
	}
}

func (c *Conn) handlePropertyChange(ev mpvEvent) {
	switch ev.ID {
	case 1: // time-pos
		if pos, ok := toFloat64(ev.Data); ok {
			c.eventCh <- messages.TimePositionChangedMsg{PositionSec: pos}
		}
	case 2: // pause
		if paused, ok := ev.Data.(bool); ok {
			state := model.PlaybackStatePlaying
			if paused {
				state = model.PlaybackStatePaused
			}
			c.eventCh <- messages.PlaybackStateChangedMsg{State: state}
		}
	case 3: // duration
		if dur, ok := toFloat64(ev.Data); ok {
			c.eventCh <- messages.DurationChangedMsg{DurationSec: dur}
		}
	case 4: // volume
		if vol, ok := toFloat64(ev.Data); ok {
			c.eventCh <- messages.VolumeChangedMsg{Volume: int(vol)}
		}
	}
}

func (c *Conn) handleEndFile(ev mpvEvent) {
	var reason model.TrackEndReason
	switch ev.Reason {
	case "eof":
		reason = model.TrackEndedEOF
	case "stop":
		reason = model.TrackEndedStopped
	case "error":
		reason = model.TrackEndedError
	default:
		reason = model.TrackEndedError
	}
	slog.Debug("mpv end-file", "reason", ev.Reason, "file_error", ev.FileError)
	c.eventCh <- messages.TrackEndedMsg{Reason: reason}
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
