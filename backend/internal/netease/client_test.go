package netease

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCookieImportPersistsPrivateSession(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	client, err := New(stateDir, cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.ImportCookie("other=value; MUSIC_U=session-token"); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(stateDir, "cookies.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("cookie permissions = %o, want 600", got)
	}
	restored, err := New(stateDir, cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	if !restored.HasSession() {
		t.Fatal("saved MUSIC_U cookie was not restored")
	}
}

func TestCookieImportRejectsNonSessionCookies(t *testing.T) {
	client, err := New(filepath.Join(t.TempDir(), "state"), filepath.Join(t.TempDir(), "cache"))
	if err != nil {
		t.Fatal(err)
	}
	if err := client.ImportCookie("foo=bar"); err == nil {
		t.Fatal("cookie without MUSIC_U or MUSIC_A was accepted")
	}
}

func TestDecodeQRResponsePreservesRiskControlDetails(t *testing.T) {
	body := []byte(`{"code":402,"message":"risk control","data":{"verifyType":50}}`)
	response, err := decodeQRResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if response.Code != 402 || response.Message != "risk control" {
		t.Fatalf("response = %#v", response)
	}
}

func TestDecodeQRResponseRejectsInvalidJSON(t *testing.T) {
	_, err := decodeQRResponse([]byte(`<html>blocked</html>`))
	if err == nil || !strings.Contains(err.Error(), "invalid character") {
		t.Fatalf("error = %v, want invalid JSON error", err)
	}
}
