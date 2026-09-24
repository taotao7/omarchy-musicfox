package mpris

import (
	"testing"

	"github.com/taotao7/omarchy-musicfox/backend/internal/catalog"
)

func TestPlaybackStatusAndLoopMapping(t *testing.T) {
	if got := playbackStatus(State{}); got != "Stopped" {
		t.Fatalf("empty playback status = %q", got)
	}
	if got := playbackStatus(State{HasTrack: true}); got != "Paused" {
		t.Fatalf("paused playback status = %q", got)
	}
	if got := playbackStatus(State{HasTrack: true, Playing: true}); got != "Playing" {
		t.Fatalf("playing playback status = %q", got)
	}
	if got := loopStatus("repeat"); got != "Track" {
		t.Fatalf("repeat loop status = %q", got)
	}
	if got := loopStatus("shuffle"); got != "Playlist" {
		t.Fatalf("shuffle loop status = %q", got)
	}
}

func TestMetadataUsesTrackDurationBeforeMPVReportsIt(t *testing.T) {
	state := State{HasTrack: true, Track: catalog.Track{
		ID: "123", Name: "Title", Artists: "Artist", Album: "Album", Duration: 123456,
	}}
	values := metadata(state)
	if got := values["mpris:length"].Value(); got != int64(123456000) {
		t.Fatalf("mpris:length = %#v", got)
	}
	if got := values["xesam:title"].Value(); got != "Title" {
		t.Fatalf("xesam:title = %#v", got)
	}
}
