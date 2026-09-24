package mpris

import (
	"fmt"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"

	"github.com/taotao7/omarchy-musicfox/backend/internal/catalog"
)

const (
	objectPath          = dbus.ObjectPath("/org/mpris/MediaPlayer2")
	rootInterface       = "org.mpris.MediaPlayer2"
	playerInterface     = "org.mpris.MediaPlayer2.Player"
	propertiesInterface = "org.freedesktop.DBus.Properties"
)

type State struct {
	Track    catalog.Track
	Playing  bool
	Position float64
	Duration float64
	Volume   float64
	Mode     string
	HasTrack bool
}

type Callbacks struct {
	State       func() State
	Next        func()
	Previous    func()
	Pause       func() error
	Play        func() error
	PlayPause   func() error
	Stop        func() error
	Seek        func(float64) error
	SetPosition func(float64) error
	SetVolume   func(float64) error
	SetMode     func(string)
}

type Server struct {
	mu        sync.Mutex
	conn      *dbus.Conn
	callbacks Callbacks
	last      State
}

func New(callbacks Callbacks) (*Server, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect MPRIS session bus: %w", err)
	}
	reply, err := conn.RequestName("org.mpris.MediaPlayer2.omarchy_musicfox", dbus.NameFlagDoNotQueue)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		conn.Close()
		return nil, fmt.Errorf("MPRIS name is already owned")
	}
	server := &Server{conn: conn, callbacks: callbacks}
	if callbacks.State != nil {
		server.last = callbacks.State()
	}
	if err := conn.ExportMethodTable(map[string]any{
		"Raise": server.Raise,
		"Quit":  server.Quit,
	}, objectPath, rootInterface); err != nil {
		conn.Close()
		return nil, err
	}
	if err := conn.ExportMethodTable(map[string]any{
		"Next": server.Next, "Previous": server.Previous,
		"Pause": server.Pause, "PlayPause": server.PlayPause,
		"Stop": server.Stop, "Play": server.Play,
		"Seek": server.SeekOffset, "SetPosition": server.SetPosition,
		"OpenUri": server.OpenUri,
	}, objectPath, playerInterface); err != nil {
		conn.Close()
		return nil, err
	}
	if err := conn.ExportMethodTable(map[string]any{
		"Get": server.Get, "GetAll": server.GetAll, "Set": server.Set,
	}, objectPath, propertiesInterface); err != nil {
		conn.Close()
		return nil, err
	}
	return server, nil
}

func (server *Server) Close() {
	if server != nil && server.conn != nil {
		server.conn.Close()
	}
}

func (server *Server) Update() {
	if server == nil || server.callbacks.State == nil {
		return
	}
	server.mu.Lock()
	state := server.callbacks.State()
	previous := server.last
	server.last = state
	server.mu.Unlock()
	changed := map[string]dbus.Variant{}
	if state.Playing != previous.Playing || state.HasTrack != previous.HasTrack {
		changed["PlaybackStatus"] = dbus.MakeVariant(playbackStatus(state))
	}
	if state.Track.ID != previous.Track.ID || state.Duration != previous.Duration {
		changed["Metadata"] = dbus.MakeVariant(metadata(state))
	}
	if state.Volume != previous.Volume {
		changed["Volume"] = dbus.MakeVariant(state.Volume / 100)
	}
	if state.Mode != previous.Mode {
		changed["LoopStatus"] = dbus.MakeVariant(loopStatus(state.Mode))
		changed["Shuffle"] = dbus.MakeVariant(state.Mode == "shuffle")
	}
	if len(changed) > 0 {
		_ = server.conn.Emit(objectPath, propertiesInterface+".PropertiesChanged",
			playerInterface, changed, []string{})
	}
}

func (server *Server) Raise() *dbus.Error { return nil }
func (server *Server) Quit() *dbus.Error  { return nil }

func (server *Server) Next() *dbus.Error {
	server.callbacks.Next()
	return nil
}

func (server *Server) Previous() *dbus.Error {
	server.callbacks.Previous()
	return nil
}

func (server *Server) Pause() *dbus.Error     { return call(server.callbacks.Pause) }
func (server *Server) PlayPause() *dbus.Error { return call(server.callbacks.PlayPause) }
func (server *Server) Stop() *dbus.Error      { return call(server.callbacks.Stop) }
func (server *Server) Play() *dbus.Error      { return call(server.callbacks.Play) }

func (server *Server) SeekOffset(offset int64) *dbus.Error {
	return callWithFloat(server.callbacks.Seek, float64(offset)/1_000_000)
}

func (server *Server) SetPosition(_ dbus.ObjectPath, position int64) *dbus.Error {
	return callWithFloat(server.callbacks.SetPosition, float64(position)/1_000_000)
}

func (server *Server) OpenUri(_ string) *dbus.Error { return nil }

func (server *Server) Get(iface, property string) (dbus.Variant, *dbus.Error) {
	properties, dbusErr := server.GetAll(iface)
	if dbusErr != nil {
		return dbus.Variant{}, dbusErr
	}
	value, ok := properties[property]
	if !ok {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", []any{property})
	}
	return value, nil
}

func (server *Server) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	if iface == rootInterface {
		return map[string]dbus.Variant{
			"CanQuit":             dbus.MakeVariant(false),
			"CanRaise":            dbus.MakeVariant(false),
			"HasTrackList":        dbus.MakeVariant(false),
			"Identity":            dbus.MakeVariant("Omarchy Musicfox"),
			"DesktopEntry":        dbus.MakeVariant("omarchy-musicfox"),
			"SupportedUriSchemes": dbus.MakeVariant([]string{}),
			"SupportedMimeTypes":  dbus.MakeVariant([]string{}),
		}, nil
	}
	if iface != playerInterface {
		return nil, dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", []any{iface})
	}
	state := server.callbacks.State()
	return map[string]dbus.Variant{
		"PlaybackStatus": dbus.MakeVariant(playbackStatus(state)),
		"LoopStatus":     dbus.MakeVariant(loopStatus(state.Mode)),
		"Rate":           dbus.MakeVariant(1.0),
		"Shuffle":        dbus.MakeVariant(state.Mode == "shuffle"),
		"Metadata":       dbus.MakeVariant(metadata(state)),
		"Volume":         dbus.MakeVariant(state.Volume / 100),
		"Position":       dbus.MakeVariant(int64(state.Position * 1_000_000)),
		"MinimumRate":    dbus.MakeVariant(1.0),
		"MaximumRate":    dbus.MakeVariant(1.0),
		"CanGoNext":      dbus.MakeVariant(state.HasTrack),
		"CanGoPrevious":  dbus.MakeVariant(state.HasTrack),
		"CanPlay":        dbus.MakeVariant(state.HasTrack),
		"CanPause":       dbus.MakeVariant(state.HasTrack),
		"CanSeek":        dbus.MakeVariant(state.HasTrack),
		"CanControl":     dbus.MakeVariant(true),
	}, nil
}

func (server *Server) Set(iface, property string, value dbus.Variant) *dbus.Error {
	if iface != playerInterface {
		return dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", []any{iface})
	}
	switch property {
	case "Volume":
		volume, ok := value.Value().(float64)
		if !ok {
			return dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs", nil)
		}
		return callWithFloat(server.callbacks.SetVolume, volume*100)
	case "LoopStatus":
		loop, ok := value.Value().(string)
		if !ok {
			return dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs", nil)
		}
		mode := "list"
		if loop == "Track" {
			mode = "repeat"
		}
		server.callbacks.SetMode(mode)
		return nil
	case "Shuffle":
		shuffle, ok := value.Value().(bool)
		if !ok {
			return dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs", nil)
		}
		if shuffle {
			server.callbacks.SetMode("shuffle")
		} else {
			server.callbacks.SetMode("list")
		}
		return nil
	default:
		return dbus.NewError("org.freedesktop.DBus.Error.PropertyReadOnly", []any{property})
	}
}

func metadata(state State) map[string]dbus.Variant {
	if !state.HasTrack {
		return map[string]dbus.Variant{}
	}
	id := strings.Map(func(character rune) rune {
		if character >= '0' && character <= '9' {
			return character
		}
		return '_'
	}, state.Track.ID)
	duration := state.Duration
	if duration <= 0 {
		duration = float64(state.Track.Duration) / 1000
	}
	return map[string]dbus.Variant{
		"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/org/mpris/MediaPlayer2/track/" + id)),
		"mpris:length":  dbus.MakeVariant(int64(duration * 1_000_000)),
		"xesam:title":   dbus.MakeVariant(state.Track.Name),
		"xesam:artist":  dbus.MakeVariant([]string{state.Track.Artists}),
		"xesam:album":   dbus.MakeVariant(state.Track.Album),
		"mpris:artUrl":  dbus.MakeVariant(state.Track.CoverURL),
	}
}

func playbackStatus(state State) string {
	if !state.HasTrack {
		return "Stopped"
	}
	if state.Playing {
		return "Playing"
	}
	return "Paused"
}

func loopStatus(mode string) string {
	if mode == "repeat" {
		return "Track"
	}
	return "Playlist"
}

func call(callback func() error) *dbus.Error {
	if callback == nil {
		return nil
	}
	if err := callback(); err != nil {
		return dbus.MakeFailedError(err)
	}
	return nil
}

func callWithFloat(callback func(float64) error, value float64) *dbus.Error {
	if callback == nil {
		return nil
	}
	if err := callback(value); err != nil {
		return dbus.MakeFailedError(err)
	}
	return nil
}
