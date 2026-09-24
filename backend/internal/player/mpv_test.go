package player

import (
	"encoding/json"
	"net"
	"testing"
)

func TestPlaybackStateIsIndependentOfPropertyOrder(t *testing.T) {
	tests := []struct {
		name   string
		events []Event
		want   bool
	}{
		{
			name: "load reports unpaused before becoming active",
			events: []Event{
				{Name: "core-idle", Data: true},
				{Name: "pause", Data: false},
				{Name: "core-idle", Data: false},
			},
			want: true,
		},
		{
			name: "paused remains stopped when idle changes",
			events: []Event{
				{Name: "pause", Data: true},
				{Name: "core-idle", Data: false},
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mpv := &MPV{snapshot: Snapshot{Idle: true, Paused: true}}
			for _, event := range test.events {
				mpv.applyProperty(event.Name, event.Data)
			}
			if got := mpv.Snapshot().Playing; got != test.want {
				t.Fatalf("Playing = %v, want %v", got, test.want)
			}
		})
	}
}

func TestToggleUsesPausedStateWhileIdle(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	mpv := &MPV{
		conn: client,
		snapshot: Snapshot{
			Playing: false,
			Paused:  false,
			Idle:    true,
		},
	}

	done := make(chan error, 1)
	go func() { done <- mpv.Toggle() }()
	var message struct {
		Command []any `json:"command"`
	}
	if err := json.NewDecoder(server).Decode(&message); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(message.Command) != 3 || message.Command[2] != true {
		t.Fatalf("toggle command = %#v, want pause=true", message.Command)
	}
}
