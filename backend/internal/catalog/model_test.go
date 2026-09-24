package catalog

import "testing"

func TestParseTracksSupportsDailyAndSearchShapes(t *testing.T) {
	payload, err := Decode([]byte(`{
		"data":{"dailySongs":[{
			"id":123,"name":"First","ar":[{"name":"Alpha"},{"name":"Beta"}],
			"al":{"name":"Album A","picUrl":"https://img/a.jpg"},"dt":181234,"fee":8
		}]},
		"result":{"songs":[{
			"id":456,"name":"Second","artists":[{"name":"Gamma"}],
			"album":{"name":"Album B","picUrl":"https://img/b.jpg"},"duration":92000
		}]}
	}`))
	if err != nil {
		t.Fatal(err)
	}

	daily := ParseTracks(payload, "data", "dailySongs")
	if len(daily) != 1 || daily[0].ID != "123" || daily[0].Artists != "Alpha / Beta" ||
		daily[0].Album != "Album A" || daily[0].Duration != 181234 || daily[0].Fee != 8 {
		t.Fatalf("unexpected daily track: %#v", daily)
	}
	search := ParseTracks(payload, "result", "songs")
	if len(search) != 1 || search[0].ID != "456" || search[0].Artists != "Gamma" ||
		search[0].Album != "Album B" || search[0].Duration != 92000 {
		t.Fatalf("unexpected search track: %#v", search)
	}
}

func TestParsePlaylistsSupportsRecommendationAndLibraryCovers(t *testing.T) {
	payload, err := Decode([]byte(`{
		"recommend":[{"id":11,"name":"Recommended","picUrl":"rec.jpg","trackCount":20}],
		"playlist":[{"id":22,"name":"Mine","coverImgUrl":"mine.jpg","trackCount":7,
			"creator":{"nickname":"Listener"}}]
	}`))
	if err != nil {
		t.Fatal(err)
	}

	recommended := ParsePlaylists(payload, "recommend")
	owned := ParsePlaylists(payload, "playlist")
	if len(recommended) != 1 || recommended[0].CoverURL != "rec.jpg" {
		t.Fatalf("unexpected recommendation: %#v", recommended)
	}
	if len(owned) != 1 || owned[0].CoverURL != "mine.jpg" || owned[0].Creator != "Listener" {
		t.Fatalf("unexpected library playlist: %#v", owned)
	}
}
