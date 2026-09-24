package catalog

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Track struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Artists  string `json:"artists"`
	Album    string `json:"album"`
	CoverURL string `json:"coverUrl"`
	Duration int64  `json:"duration"`
	Fee      int    `json:"fee"`
}

type Playlist struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CoverURL   string `json:"coverUrl"`
	TrackCount int    `json:"trackCount"`
	Creator    string `json:"creator"`
}

type Profile struct {
	UserID    string `json:"userId"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}

func Decode(raw []byte) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode NetEase response: %w", err)
	}
	return payload, nil
}

func At(value any, path ...string) any {
	current := value
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = object[key]
	}
	return current
}

func Slice(value any, path ...string) []any {
	items, _ := At(value, path...).([]any)
	return items
}

func String(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int:
		return strconv.Itoa(typed)
	default:
		return ""
	}
}

func Int64(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case json.Number:
		result, _ := typed.Int64()
		return result
	case string:
		result, _ := strconv.ParseInt(typed, 10, 64)
		return result
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func ParseTracks(value any, path ...string) []Track {
	items := Slice(value, path...)
	tracks := make([]Track, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		track := ParseTrack(object)
		if track.ID != "" {
			tracks = append(tracks, track)
		}
	}
	return tracks
}

func ParseTrack(object map[string]any) Track {
	album, _ := object["al"].(map[string]any)
	if album == nil {
		album, _ = object["album"].(map[string]any)
	}
	artists := object["ar"]
	if artists == nil {
		artists = object["artists"]
	}
	duration := Int64(object["dt"])
	if duration == 0 {
		duration = Int64(object["duration"])
	}
	return Track{
		ID:       String(object["id"]),
		Name:     String(object["name"]),
		Artists:  artistNames(artists),
		Album:    String(album["name"]),
		CoverURL: firstNonEmpty(String(album["picUrl"]), String(object["albumPicUrl"])),
		Duration: duration,
		Fee:      int(Int64(object["fee"])),
	}
}

func ParsePlaylists(value any, path ...string) []Playlist {
	items := Slice(value, path...)
	playlists := make([]Playlist, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		creator, _ := object["creator"].(map[string]any)
		playlist := Playlist{
			ID:         String(object["id"]),
			Name:       String(object["name"]),
			CoverURL:   firstNonEmpty(String(object["picUrl"]), String(object["coverImgUrl"])),
			TrackCount: int(Int64(object["trackCount"])),
			Creator:    String(creator["nickname"]),
		}
		if playlist.ID != "" {
			playlists = append(playlists, playlist)
		}
	}
	return playlists
}

func ParseProfile(value any) Profile {
	object, _ := At(value, "profile").(map[string]any)
	return Profile{
		UserID:    String(object["userId"]),
		Nickname:  String(object["nickname"]),
		AvatarURL: String(object["avatarUrl"]),
	}
}

func artistNames(value any) string {
	items, _ := value.([]any)
	names := make([]string, 0, len(items))
	for _, item := range items {
		artist, _ := item.(map[string]any)
		if name := String(artist["name"]); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, " / ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
