package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/taotao7/omarchy-musicfox/backend/internal/catalog"
	"github.com/taotao7/omarchy-musicfox/backend/internal/mpris"
	"github.com/taotao7/omarchy-musicfox/backend/internal/netease"
	"github.com/taotao7/omarchy-musicfox/backend/internal/player"
)

type Command struct {
	ID      string          `json:"id"`
	Command string          `json:"command"`
	Args    json.RawMessage `json:"args"`
}

type App struct {
	mu       sync.Mutex
	emitMu   sync.Mutex
	mprisMu  sync.Mutex
	client   *netease.Client
	player   *player.MPV
	mpris    *mpris.Server
	queue    []catalog.Track
	index    int
	mode     string
	liked    bool
	likedIDs map[string]bool
	profile  catalog.Profile
	qrCancel chan struct{}
}

func main() {
	stateHome := envDefault("XDG_STATE_HOME", filepath.Join(userHome(), ".local", "state"))
	cacheHome := envDefault("XDG_CACHE_HOME", filepath.Join(userHome(), ".cache"))
	runtimeHome := envDefault("XDG_RUNTIME_DIR", filepath.Join(os.TempDir(), "omarchy-musicfox-"+strconv.Itoa(os.Getuid())))
	stateDir := filepath.Join(stateHome, "omarchy-musicfox")
	cacheDir := filepath.Join(cacheHome, "omarchy-musicfox")
	runtimeDir := filepath.Join(runtimeHome, "omarchy-musicfox")

	client, err := netease.New(stateDir, cacheDir)
	if err != nil {
		fatal(err)
	}
	app := &App{client: client, index: -1, mode: "list", likedIDs: make(map[string]bool)}
	app.player, err = player.New(runtimeDir, app.onPlayback, app.onTrackEnd)
	if err != nil {
		fatal(err)
	}
	defer app.player.Close()
	defer app.setMPRIS(false)
	_ = app.restoreAccount()
	app.emit("ready", map[string]any{
		"loggedIn": app.profile.UserID != "", "profile": app.profile,
		"playback": app.playbackPayload(), "version": "0.1.2",
	})

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		var command Command
		if err := json.Unmarshal(scanner.Bytes(), &command); err != nil {
			app.emitError("", fmt.Errorf("invalid command: %w", err))
			continue
		}
		go app.handle(command)
	}
}

func (app *App) handle(command Command) {
	args := map[string]any{}
	_ = json.Unmarshal(command.Args, &args)
	respond := func(kind string, data any, err error) {
		if err != nil {
			app.emitError(command.ID, err)
			return
		}
		app.emit("response", map[string]any{"id": command.ID, "kind": kind, "data": data})
	}

	switch command.Command {
	case "status":
		respond("status", map[string]any{"profile": app.profile, "playback": app.playbackPayload()}, nil)
	case "loginQrStart":
		status, err := app.client.StartQR()
		respond("qr", status, err)
		if err == nil {
			app.startQRPolling()
		}
	case "loginCookie":
		err := app.client.ImportCookie(stringArg(args, "cookie"))
		if err == nil {
			err = app.restoreAccount()
		}
		respond("login", app.profile, err)
	case "logout":
		err := app.client.Logout()
		app.mu.Lock()
		app.profile = catalog.Profile{}
		app.likedIDs = make(map[string]bool)
		app.mu.Unlock()
		respond("logout", map[string]any{"ok": err == nil}, err)
	case "home":
		songs, err := app.client.DailySongs()
		if err != nil {
			respond("home", nil, err)
			return
		}
		playlists, playlistErr := app.client.RecommendedPlaylists()
		respond("home", map[string]any{"songs": songs, "playlists": playlists}, playlistErr)
	case "search":
		tracks, err := app.client.Search(stringArg(args, "query"))
		respond("search", tracks, err)
	case "library":
		app.mu.Lock()
		userID := app.profile.UserID
		app.mu.Unlock()
		if userID == "" {
			respond("library", nil, errors.New("login required"))
			return
		}
		playlists, err := app.client.UserPlaylists(userID)
		respond("library", playlists, err)
	case "playlist":
		tracks, err := app.client.Playlist(stringArg(args, "id"))
		respond("playlist", tracks, err)
	case "toplists":
		playlists, err := app.client.Toplists()
		respond("toplists", playlists, err)
	case "personalFm":
		tracks, err := app.client.PersonalFM()
		if err == nil && len(tracks) > 0 {
			err = app.playQueue(tracks, 0)
		}
		respond("personalFm", tracks, err)
	case "play":
		var payload struct {
			Tracks []catalog.Track `json:"tracks"`
			Index  int             `json:"index"`
		}
		_ = json.Unmarshal(command.Args, &payload)
		err := app.playQueue(payload.Tracks, payload.Index)
		respond("play", app.playbackPayload(), err)
	case "playIndex":
		err := app.playAt(intArg(args, "index"))
		respond("play", app.playbackPayload(), err)
	case "toggle":
		err := app.player.Toggle()
		respond("playback", app.playbackPayload(), err)
	case "playbackPlay":
		err := app.player.Play()
		respond("playback", app.playbackPayload(), err)
	case "playbackPause":
		err := app.player.Pause()
		respond("playback", app.playbackPayload(), err)
	case "stop":
		err := app.player.Stop()
		respond("playback", app.playbackPayload(), err)
	case "next":
		app.next()
		respond("playback", app.playbackPayload(), nil)
	case "previous":
		app.previous()
		respond("playback", app.playbackPayload(), nil)
	case "seek":
		err := app.player.Seek(floatArg(args, "position"))
		respond("playback", app.playbackPayload(), err)
	case "volume":
		err := app.player.SetVolume(floatArg(args, "volume"))
		respond("playback", app.playbackPayload(), err)
	case "mode":
		mode := stringArg(args, "mode")
		if mode != "list" && mode != "repeat" && mode != "shuffle" {
			mode = "list"
		}
		app.setMode(mode)
		respond("playback", app.playbackPayload(), nil)
	case "like":
		liked := boolArg(args, "liked")
		app.mu.Lock()
		track := app.currentTrackLocked()
		app.mu.Unlock()
		if track.ID == "" {
			respond("like", nil, errors.New("nothing is playing"))
			return
		}
		err := app.client.Like(track.ID, liked)
		if err == nil {
			app.mu.Lock()
			app.liked = liked
			app.likedIDs[track.ID] = liked
			app.mu.Unlock()
		}
		respond("like", map[string]any{"liked": liked}, err)
	case "quality":
		app.client.SetQuality(stringArg(args, "quality"))
		respond("quality", map[string]any{"ok": true}, nil)
	case "mpris":
		enabled := boolArg(args, "enabled")
		err := app.setMPRIS(enabled)
		respond("mpris", map[string]any{"enabled": enabled && err == nil}, err)
	default:
		respond("unknown", nil, fmt.Errorf("unknown command %q", command.Command))
	}
}

func (app *App) playQueue(tracks []catalog.Track, index int) error {
	if len(tracks) == 0 {
		return errors.New("queue is empty")
	}
	if index < 0 || index >= len(tracks) {
		return errors.New("queue index is out of range")
	}
	app.mu.Lock()
	app.queue = append([]catalog.Track(nil), tracks...)
	app.index = index
	app.mu.Unlock()
	return app.playAt(index)
}

func (app *App) playAt(index int) error {
	app.mu.Lock()
	if index < 0 || index >= len(app.queue) {
		app.mu.Unlock()
		return errors.New("queue index is out of range")
	}
	app.index = index
	track := app.queue[index]
	app.liked = app.likedIDs[track.ID]
	app.mu.Unlock()
	streamURL, err := app.client.StreamURL(track.ID)
	if err != nil {
		return err
	}
	if err := app.player.Load(streamURL); err != nil {
		return err
	}
	app.emit("playback", app.playbackPayload())
	go func() {
		lyrics, lyricErr := app.client.Lyrics(track.ID)
		if lyricErr != nil {
			app.emitError("", lyricErr)
			return
		}
		lyrics["trackId"] = track.ID
		app.emit("lyrics", lyrics)
	}()
	return nil
}

func (app *App) next() {
	app.advance(false)
}

func (app *App) onTrackEnd() {
	app.advance(true)
}

func (app *App) advance(automatic bool) {
	app.mu.Lock()
	if len(app.queue) == 0 {
		app.mu.Unlock()
		return
	}
	next := app.index + 1
	if automatic && app.mode == "repeat" {
		next = app.index
	} else if app.mode == "shuffle" {
		next = rand.Intn(len(app.queue))
	} else if next >= len(app.queue) {
		next = 0
	}
	app.mu.Unlock()
	if err := app.playAt(next); err != nil {
		app.emitError("", err)
	}
}

func (app *App) previous() {
	app.mu.Lock()
	if len(app.queue) == 0 {
		app.mu.Unlock()
		return
	}
	previous := app.index - 1
	if previous < 0 {
		previous = len(app.queue) - 1
	}
	app.mu.Unlock()
	if err := app.playAt(previous); err != nil {
		app.emitError("", err)
	}
}

func (app *App) onPlayback(snapshot player.Snapshot) {
	app.emit("playback", app.playbackPayloadWith(snapshot))
	app.updateMPRIS()
}

func (app *App) mprisState() mpris.State {
	snapshot := app.player.Snapshot()
	app.mu.Lock()
	defer app.mu.Unlock()
	track := app.currentTrackLocked()
	return mpris.State{
		Track: track, Playing: snapshot.Playing, Position: snapshot.Position,
		Duration: snapshot.Duration, Volume: snapshot.Volume, Mode: app.mode,
		HasTrack: track.ID != "",
	}
}

func (app *App) setMode(mode string) {
	app.mu.Lock()
	app.mode = mode
	app.mu.Unlock()
	app.emit("playback", app.playbackPayload())
	app.updateMPRIS()
}

func (app *App) setMPRIS(enabled bool) error {
	app.mprisMu.Lock()
	defer app.mprisMu.Unlock()
	if !enabled {
		if app.mpris != nil {
			app.mpris.Close()
			app.mpris = nil
		}
		return nil
	}
	if app.mpris != nil {
		return nil
	}
	server, err := mpris.New(mpris.Callbacks{
		State: app.mprisState,
		Next:  app.next, Previous: app.previous,
		Pause: app.player.Pause, Play: app.player.Play, PlayPause: app.player.Toggle,
		Stop: app.player.Stop,
		Seek: func(offset float64) error {
			return app.player.Seek(app.player.Snapshot().Position + offset)
		},
		SetPosition: app.player.Seek, SetVolume: app.player.SetVolume,
		SetMode: app.setMode,
	})
	if err != nil {
		return err
	}
	app.mpris = server
	return nil
}

func (app *App) updateMPRIS() {
	app.mprisMu.Lock()
	defer app.mprisMu.Unlock()
	if app.mpris != nil {
		app.mpris.Update()
	}
}

func (app *App) playbackPayload() map[string]any {
	return app.playbackPayloadWith(app.player.Snapshot())
}

func (app *App) playbackPayloadWith(snapshot player.Snapshot) map[string]any {
	app.mu.Lock()
	defer app.mu.Unlock()
	return map[string]any{
		"track": app.currentTrackLocked(), "queue": app.queue, "index": app.index,
		"mode": app.mode, "liked": app.liked, "playing": snapshot.Playing,
		"position": snapshot.Position, "duration": snapshot.Duration,
		"volume": snapshot.Volume, "idle": snapshot.Idle,
	}
}

func (app *App) currentTrackLocked() catalog.Track {
	if app.index < 0 || app.index >= len(app.queue) {
		return catalog.Track{}
	}
	return app.queue[app.index]
}

func (app *App) restoreAccount() error {
	if !app.client.HasSession() {
		return errors.New("no saved session")
	}
	profile, err := app.client.Account()
	if err != nil {
		return err
	}
	likedIDs, _ := app.client.LikeIDs(profile.UserID)
	if likedIDs == nil {
		likedIDs = make(map[string]bool)
	}
	app.mu.Lock()
	app.profile = profile
	app.likedIDs = likedIDs
	app.mu.Unlock()
	return nil
}

func (app *App) startQRPolling() {
	app.mu.Lock()
	if app.qrCancel != nil {
		close(app.qrCancel)
	}
	cancel := make(chan struct{})
	app.qrCancel = cancel
	app.mu.Unlock()
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-cancel:
				return
			case <-ticker.C:
				status, err := app.client.CheckQR()
				if err != nil {
					app.emitError("", err)
					return
				}
				app.emit("qr", status)
				if status.Code == 803 {
					if err := app.restoreAccount(); err != nil {
						app.emitError("", err)
					} else {
						app.emit("login", app.profile)
					}
					return
				}
				if status.Code == 800 {
					return
				}
			}
		}
	}()
}

func (app *App) emit(event string, data any) {
	app.emitMu.Lock()
	defer app.emitMu.Unlock()
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"event": event, "data": data})
}

func (app *App) emitError(id string, err error) {
	app.emit("error", map[string]any{"id": id, "message": err.Error()})
}

func fatal(err error) {
	payload, _ := json.Marshal(map[string]any{"event": "fatal", "data": map[string]any{"message": err.Error()}})
	fmt.Println(string(payload))
	os.Exit(1)
}

func envDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func userHome() string {
	home, _ := os.UserHomeDir()
	return home
}

func stringArg(args map[string]any, key string) string {
	return strings.TrimSpace(catalog.String(args[key]))
}

func intArg(args map[string]any, key string) int {
	return int(catalog.Int64(args[key]))
}

func floatArg(args map[string]any, key string) float64 {
	value, _ := args[key].(float64)
	return value
}

func boolArg(args map[string]any, key string) bool {
	value, _ := args[key].(bool)
	return value
}
