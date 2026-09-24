package player

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Snapshot struct {
	Playing  bool    `json:"playing"`
	Position float64 `json:"position"`
	Duration float64 `json:"duration"`
	Volume   float64 `json:"volume"`
	Idle     bool    `json:"idle"`
	Paused   bool    `json:"-"`
}

type Event struct {
	Name string
	Data any
}

type MPV struct {
	mu         sync.Mutex
	command    *exec.Cmd
	conn       net.Conn
	socket     string
	snapshot   Snapshot
	lastNotify time.Time
	onChange   func(Snapshot)
	onEnd      func()
}

func New(runtimeDir string, onChange func(Snapshot), onEnd func()) (*MPV, error) {
	if _, err := exec.LookPath("mpv"); err != nil {
		return nil, errors.New("mpv is required for audio playback")
	}
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return nil, err
	}
	return &MPV{
		socket:   filepath.Join(runtimeDir, "mpv.sock"),
		onChange: onChange,
		onEnd:    onEnd,
		snapshot: Snapshot{Volume: 70, Idle: true, Paused: true},
	}, nil
}

func (mpv *MPV) Start() error {
	mpv.mu.Lock()
	defer mpv.mu.Unlock()
	if mpv.command != nil {
		return nil
	}
	_ = os.Remove(mpv.socket)
	mpv.command = exec.Command("mpv",
		"--idle=yes", "--no-video", "--audio-display=no", "--terminal=no",
		"--input-terminal=no", "--load-scripts=no", "--really-quiet", "--volume=70",
		"--input-ipc-server="+mpv.socket,
		"--http-header-fields=Referer: https://music.163.com/,User-Agent: Mozilla/5.0",
	)
	if err := mpv.command.Start(); err != nil {
		mpv.command = nil
		return fmt.Errorf("start mpv: %w", err)
	}
	var connection net.Conn
	var err error
	for attempt := 0; attempt < 60; attempt++ {
		connection, err = net.Dial("unix", mpv.socket)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		_ = mpv.command.Process.Kill()
		mpv.command = nil
		return fmt.Errorf("connect to mpv: %w", err)
	}
	mpv.conn = connection
	go mpv.readLoop(connection)
	go mpv.waitProcess(mpv.command)
	for id, property := range []string{"pause", "time-pos", "duration", "volume", "core-idle"} {
		_ = mpv.sendLocked([]any{"observe_property", id + 1, property})
	}
	return nil
}

func (mpv *MPV) Load(streamURL string) error {
	if err := mpv.Start(); err != nil {
		return err
	}
	return mpv.Send("loadfile", streamURL, "replace")
}

func (mpv *MPV) Toggle() error {
	mpv.mu.Lock()
	defer mpv.mu.Unlock()
	return mpv.setPausedLocked(!mpv.snapshot.Paused)
}

func (mpv *MPV) Pause() error { return mpv.setPaused(true) }
func (mpv *MPV) Play() error  { return mpv.setPaused(false) }
func (mpv *MPV) Stop() error  { return mpv.Send("stop") }

func (mpv *MPV) setPaused(paused bool) error {
	mpv.mu.Lock()
	defer mpv.mu.Unlock()
	return mpv.setPausedLocked(paused)
}

func (mpv *MPV) setPausedLocked(paused bool) error {
	if mpv.conn == nil {
		return errors.New("mpv is not running")
	}
	if err := mpv.sendLocked([]any{"set_property", "pause", paused}); err != nil {
		return err
	}
	mpv.snapshot.Paused = paused
	mpv.snapshot.Playing = playing(mpv.snapshot.Paused, mpv.snapshot.Idle)
	return nil
}

func (mpv *MPV) Seek(seconds float64) error {
	return mpv.Send("seek", seconds, "absolute", "exact")
}

func (mpv *MPV) SetVolume(volume float64) error {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	return mpv.Send("set_property", "volume", volume)
}

func (mpv *MPV) Send(command ...any) error {
	mpv.mu.Lock()
	defer mpv.mu.Unlock()
	if mpv.conn == nil {
		return errors.New("mpv is not running")
	}
	return mpv.sendLocked(command)
}

func (mpv *MPV) sendLocked(command []any) error {
	payload, _ := json.Marshal(map[string]any{"command": command})
	payload = append(payload, '\n')
	_, err := mpv.conn.Write(payload)
	return err
}

func (mpv *MPV) Snapshot() Snapshot {
	mpv.mu.Lock()
	defer mpv.mu.Unlock()
	return mpv.snapshot
}

func (mpv *MPV) Close() {
	mpv.mu.Lock()
	defer mpv.mu.Unlock()
	if mpv.conn != nil {
		_ = mpv.sendLocked([]any{"quit"})
		_ = mpv.conn.Close()
	}
	if mpv.command != nil && mpv.command.Process != nil {
		_ = mpv.command.Process.Kill()
	}
	mpv.conn = nil
	mpv.command = nil
	_ = os.Remove(mpv.socket)
}

func (mpv *MPV) readLoop(connection net.Conn) {
	scanner := bufio.NewScanner(connection)
	for scanner.Scan() {
		var message map[string]any
		if json.Unmarshal(scanner.Bytes(), &message) != nil {
			continue
		}
		event, _ := message["event"].(string)
		if event == "property-change" {
			name, _ := message["name"].(string)
			mpv.applyProperty(name, message["data"])
		} else if event == "end-file" {
			reason, _ := message["reason"].(string)
			if reason == "eof" && mpv.onEnd != nil {
				go mpv.onEnd()
			}
		}
	}
}

func (mpv *MPV) applyProperty(name string, value any) {
	mpv.mu.Lock()
	switch name {
	case "pause":
		mpv.snapshot.Paused, _ = value.(bool)
	case "time-pos":
		mpv.snapshot.Position, _ = value.(float64)
	case "duration":
		mpv.snapshot.Duration, _ = value.(float64)
	case "volume":
		mpv.snapshot.Volume, _ = value.(float64)
	case "core-idle":
		mpv.snapshot.Idle, _ = value.(bool)
	}
	mpv.snapshot.Playing = playing(mpv.snapshot.Paused, mpv.snapshot.Idle)
	snapshot := mpv.snapshot
	shouldNotify := name != "time-pos" || time.Since(mpv.lastNotify) >= 250*time.Millisecond
	if shouldNotify {
		mpv.lastNotify = time.Now()
	}
	mpv.mu.Unlock()
	if shouldNotify && mpv.onChange != nil {
		mpv.onChange(snapshot)
	}
}

func (mpv *MPV) waitProcess(command *exec.Cmd) {
	_ = command.Wait()
	mpv.mu.Lock()
	if mpv.command != command {
		mpv.mu.Unlock()
		return
	}
	if mpv.conn != nil {
		_ = mpv.conn.Close()
	}
	mpv.conn = nil
	mpv.command = nil
	mpv.snapshot.Playing = false
	mpv.snapshot.Idle = true
	mpv.snapshot.Paused = true
	snapshot := mpv.snapshot
	mpv.mu.Unlock()
	if mpv.onChange != nil {
		mpv.onChange(snapshot)
	}
}

func playing(paused, idle bool) bool {
	return !paused && !idle
}
