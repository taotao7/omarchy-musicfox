package netease

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-musicfox/netease-music/service"
	"github.com/go-musicfox/netease-music/util"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/taotao7/omarchy-musicfox/backend/internal/catalog"
)

type Client struct {
	mu         sync.Mutex
	jar        *cookiejar.Jar
	cookiePath string
	qr         *service.LoginQRService
	qrPath     string
	quality    service.SongQualityLevel
	musicURL   *url.URL
}

type QRStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Image   string `json:"image,omitempty"`
	URL     string `json:"url,omitempty"`
}

func New(stateDir, cacheDir string) (*Client, error) {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return nil, err
	}
	cookiePath := filepath.Join(stateDir, "cookies.json")
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	musicURL, _ := url.Parse("https://music.163.com/")
	if raw, readErr := os.ReadFile(cookiePath); readErr == nil {
		var cookies []*http.Cookie
		if json.Unmarshal(raw, &cookies) == nil {
			jar.SetCookies(musicURL, cookies)
		}
	}
	util.SetGlobalCookieJar(jar)
	util.HTTPClientTimeout = 15 * time.Second
	return &Client{
		jar:        jar,
		cookiePath: cookiePath,
		qrPath:     filepath.Join(cacheDir, "login-qr.png"),
		quality:    service.Higher,
		musicURL:   musicURL,
	}, nil
}

func (client *Client) SetQuality(value string) {
	client.mu.Lock()
	defer client.mu.Unlock()
	quality := service.SongQualityLevel(value)
	if quality.IsValid() {
		client.quality = quality
	}
}

func (client *Client) HasSession() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.hasSession()
}

func (client *Client) hasSession() bool {
	for _, cookie := range client.jar.Cookies(client.musicURL) {
		if (cookie.Name == "MUSIC_U" || cookie.Name == "MUSIC_A") && cookie.Value != "" {
			return true
		}
	}
	return false
}

func (client *Client) ImportCookie(raw string) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	var cookies []*http.Cookie
	for _, part := range strings.Split(raw, ";") {
		name, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found || name == "" || value == "" {
			continue
		}
		cookies = append(cookies, &http.Cookie{
			Name: name, Value: value, Path: "/", Domain: ".music.163.com",
			Expires: time.Now().Add(365 * 24 * time.Hour), Secure: true,
		})
	}
	if len(cookies) == 0 {
		return errors.New("cookie string contains no name=value pairs")
	}
	client.jar.SetCookies(client.musicURL, cookies)
	if !client.hasSession() {
		return errors.New("cookie must include MUSIC_U or MUSIC_A")
	}
	return client.save()
}

func (client *Client) StartQR() (QRStatus, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	qr := &service.LoginQRService{}
	code, body, loginURL, err := qr.GetKey()
	if err != nil {
		return QRStatus{}, err
	}
	if code != 200 || loginURL == "" {
		return QRStatus{}, fmt.Errorf("QR key request failed (%v): %s", code, compact(body))
	}
	if err := qrcode.WriteFile(loginURL, qrcode.Medium, 360, client.qrPath); err != nil {
		return QRStatus{}, fmt.Errorf("render QR code: %w", err)
	}
	client.qr = qr
	return QRStatus{Code: 801, Message: "waiting", Image: client.qrPath, URL: loginURL}, nil
}

func (client *Client) CheckQR() (QRStatus, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.qr == nil {
		return QRStatus{}, errors.New("QR login has not been started")
	}
	_, body, err := client.qr.CheckQR()
	if err != nil {
		return QRStatus{}, err
	}
	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return QRStatus{}, fmt.Errorf("decode QR status: %w", err)
	}
	status := QRStatus{Code: response.Code, Message: response.Message, Image: client.qrPath}
	if response.Code == 803 {
		if err := client.save(); err != nil {
			return status, err
		}
		client.qr = nil
	}
	return status, nil
}

func (client *Client) Account() (catalog.Profile, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	_, raw := (&service.UserAccountService{}).AccountInfo()
	payload, err := catalog.Decode(raw)
	if err != nil {
		return catalog.Profile{}, err
	}
	if err := apiError(payload); err != nil {
		return catalog.Profile{}, err
	}
	profile := catalog.ParseProfile(payload)
	if profile.UserID == "" {
		return profile, errors.New("not logged in")
	}
	_ = client.save()
	return profile, nil
}

func (client *Client) Logout() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	_, _, _ = (&service.LogoutService{}).Logout()
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	client.jar = jar
	util.SetGlobalCookieJar(jar)
	return client.save()
}

func (client *Client) RefreshSession() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	code, body, err := (&service.LoginRefreshService{}).LoginRefresh()
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("refresh failed (%v): %s", code, compact(body))
	}
	return client.save()
}

func (client *Client) DailySongs() ([]catalog.Track, error) {
	_, raw := (&service.RecommendSongsService{}).RecommendSongs()
	return tracks(raw, "data", "dailySongs")
}

func (client *Client) RecommendedPlaylists() ([]catalog.Playlist, error) {
	_, raw := (&service.RecommendResourceService{}).RecommendResource()
	payload, err := catalog.Decode(raw)
	if err != nil {
		return nil, err
	}
	if err := apiError(payload); err != nil {
		return nil, err
	}
	return catalog.ParsePlaylists(payload, "recommend"), nil
}

func (client *Client) Search(query string) ([]catalog.Track, error) {
	_, raw := (&service.SearchService{S: query, Type: "1", Limit: "50"}).Search()
	return tracks(raw, "result", "songs")
}

func (client *Client) UserPlaylists(userID string) ([]catalog.Playlist, error) {
	_, raw := (&service.UserPlaylistService{Uid: userID, Limit: "100"}).UserPlaylist()
	payload, err := catalog.Decode(raw)
	if err != nil {
		return nil, err
	}
	if err := apiError(payload); err != nil {
		return nil, err
	}
	return catalog.ParsePlaylists(payload, "playlist"), nil
}

func (client *Client) Playlist(id string) ([]catalog.Track, error) {
	_, raw := (&service.PlaylistTrackAllService{Id: id}).AllTracks()
	return tracks(raw, "playlist", "tracks")
}

func (client *Client) Toplists() ([]catalog.Playlist, error) {
	_, raw := (&service.ToplistService{}).Toplist()
	payload, err := catalog.Decode(raw)
	if err != nil {
		return nil, err
	}
	if err := apiError(payload); err != nil {
		return nil, err
	}
	return catalog.ParsePlaylists(payload, "list"), nil
}

func (client *Client) PersonalFM() ([]catalog.Track, error) {
	_, raw := (&service.PersonalFmService{}).PersonalFm()
	return tracks(raw, "data")
}

func (client *Client) StreamURL(id string) (string, error) {
	client.mu.Lock()
	quality := client.quality
	client.mu.Unlock()
	_, raw, err := (&service.SongUrlV1Service{ID: id, Level: quality, SkipUNM: true}).SongUrl()
	if err != nil {
		return "", err
	}
	payload, err := catalog.Decode(raw)
	if err != nil {
		return "", err
	}
	items := catalog.Slice(payload, "data")
	if len(items) == 0 {
		return "", errors.New("no playback URL returned")
	}
	item, _ := items[0].(map[string]any)
	streamURL := catalog.String(item["url"])
	if streamURL == "" {
		return "", errors.New("track is unavailable for this account or region")
	}
	return streamURL, nil
}

func (client *Client) Lyrics(id string) (map[string]any, error) {
	_, raw := (&service.LyricService{ID: id}).Lyric()
	payload, err := catalog.Decode(raw)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"lrc":        catalog.String(catalog.At(payload, "lrc", "lyric")),
		"translated": catalog.String(catalog.At(payload, "tlyric", "lyric")),
	}, nil
}

func (client *Client) Like(id string, liked bool) error {
	value := "false"
	if liked {
		value = "true"
	}
	code, raw := (&service.LikeService{ID: id, L: value}).Like()
	if code != 200 {
		return fmt.Errorf("like request failed (%v): %s", code, compact(raw))
	}
	return nil
}

func (client *Client) LikeIDs(userID string) (map[string]bool, error) {
	_, raw := (&service.LikeListService{UID: userID}).LikeList()
	payload, err := catalog.Decode(raw)
	if err != nil {
		return nil, err
	}
	if err := apiError(payload); err != nil {
		return nil, err
	}
	liked := make(map[string]bool)
	for _, value := range catalog.Slice(payload, "ids") {
		if id := catalog.String(value); id != "" {
			liked[id] = true
		}
	}
	return liked, nil
}

func tracks(raw []byte, path ...string) ([]catalog.Track, error) {
	payload, err := catalog.Decode(raw)
	if err != nil {
		return nil, err
	}
	if err := apiError(payload); err != nil {
		return nil, err
	}
	return catalog.ParseTracks(payload, path...), nil
}

func apiError(payload map[string]any) error {
	code := catalog.Int64(payload["code"])
	if code == 0 || code == 200 {
		return nil
	}
	message := catalog.String(payload["message"])
	if message == "" {
		message = catalog.String(payload["msg"])
	}
	if message == "" {
		message = "request failed"
	}
	return fmt.Errorf("NetEase API %d: %s", code, message)
}

func (client *Client) save() error {
	cookies := client.jar.Cookies(client.musicURL)
	raw, err := json.Marshal(cookies)
	if err != nil {
		return fmt.Errorf("encode login session: %w", err)
	}
	temporary := client.cookiePath + ".tmp"
	if err := os.WriteFile(temporary, raw, 0o600); err != nil {
		return fmt.Errorf("save login session: %w", err)
	}
	if err := os.Rename(temporary, client.cookiePath); err != nil {
		return fmt.Errorf("save login session: %w", err)
	}
	return os.Chmod(client.cookiePath, 0o600)
}

func compact(raw []byte) string {
	text := strings.TrimSpace(string(raw))
	if len(text) > 240 {
		return text[:240]
	}
	return text
}
