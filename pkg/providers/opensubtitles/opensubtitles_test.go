// file: pkg/providers/opensubtitles/opensubtitles_test.go
// version: 1.1.0
// guid: 6bd3fcef-66ea-425f-be25-cbcea87859fe
// last-edited: 2026-08-04

package opensubtitles

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

type mockHandler struct{}

func (mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle both old and new API endpoints for compatibility
	if strings.HasPrefix(r.URL.Path, "/search") || strings.HasPrefix(r.URL.Path, "/subtitles") {
		// Support both old format and new OpenSubtitles API format
		if strings.HasPrefix(r.URL.Path, "/subtitles") {
			fmt.Fprint(w, `{"data":[{"attributes":{"subtitle_id":"1","files":[{"file_id":1}]}}]}`)
		} else {
			fmt.Fprintf(w, `[{"SubDownloadLink":"http://%s/download"}]`, r.Host)
		}
		return
	}
	if r.URL.Path == "/download" {
		if r.Method == http.MethodPost {
			// New API: POST to /download returns a link
			fmt.Fprintf(w, `{"link":"http://%s/file.srt"}`, r.Host)
		} else {
			// Old API: GET /download returns file directly
			fmt.Fprint(w, "sub data")
		}
		return
	}
	if r.URL.Path == "/file.srt" {
		fmt.Fprint(w, "sub data")
		return
	}
	w.WriteHeader(404)
}

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(mockHandler{})
	defer srv.Close()
	viper.Set("opensubtitles.username", "u")
	viper.Set("opensubtitles.password", "p")
	defer viper.Reset()
	c := New("")
	c.token = "t"
	c.tokenExp = time.Now().Add(time.Hour)
	c.APIURL = srv.URL
	c.HTTPClient = srv.Client()
	// override fileHash to avoid reading a real file
	orig := fileHashFunc
	fileHashFunc = func(string) (uint64, int64, error) { return 1, 1, nil }
	defer func() { fileHashFunc = orig }()

	data, err := c.Fetch(context.Background(), "dummy.mkv", "en")
	if err != nil {
		t.Fatalf("fetch error: %v", err)
	}
	if string(data) != "sub data" {
		t.Fatalf("unexpected data: %s", data)
	}
}

// TestSearch lists download links without downloading.
func TestSearch(t *testing.T) {
	srv := httptest.NewServer(mockHandler{})
	defer srv.Close()
	viper.Set("opensubtitles.username", "u")
	viper.Set("opensubtitles.password", "p")
	defer viper.Reset()
	c := New("")
	c.token = "t"
	c.tokenExp = time.Now().Add(time.Hour)
	c.APIURL = srv.URL
	c.HTTPClient = srv.Client()
	orig := fileHashFunc
	fileHashFunc = func(string) (uint64, int64, error) { return 1, 1, nil }
	defer func() { fileHashFunc = orig }()

	urls, err := c.Search(context.Background(), "dummy.mkv", "en")
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(urls) != 1 || !strings.Contains(urls[0], "/download") {
		t.Fatalf("unexpected urls: %v", urls)
	}
}

// TestFetchByResult downloads a specific candidate rather than the first match.
func TestFetchByResult(t *testing.T) {
	srv := httptest.NewServer(mockHandler{})
	defer srv.Close()
	c := New("")
	c.APIURL = srv.URL
	c.HTTPClient = srv.Client()

	// The download id comes from attributes.files[].file_id. This test used to
	// set only attributes.subtitle_id, which encoded the very assumption that
	// was wrong — they are different values, and the API 404s on the latter.
	var result SearchResult
	result.Attributes.SubtitleID = "42"
	result.Attributes.Files = append(result.Attributes.Files, struct {
		FileID   int    `json:"file_id"`
		CDNumber int    `json:"cd_number"`
		FileName string `json:"file_name"`
	}{FileID: 42, FileName: "s.srt"})

	data, err := c.FetchByResult(context.Background(), result)
	if err != nil {
		t.Fatalf("fetch by result error: %v", err)
	}
	if string(data) != "sub data" {
		t.Fatalf("unexpected data: %s", data)
	}

	// A result with no file id cannot be downloaded.
	if _, err := c.FetchByResult(context.Background(), SearchResult{}); err == nil {
		t.Fatal("expected error for result with no file id")
	}
}

// TestNewUsesConfig verifies that viper settings override defaults.
func TestNewUsesConfig(t *testing.T) {
	viper.Set("opensubtitles.api_url", "http://api")
	viper.Set("opensubtitles.user_agent", "ua")
	defer viper.Reset()
	c := New("k")
	if c.APIURL != "http://api" {
		t.Fatalf("expected api_url http://api, got %s", c.APIURL)
	}
	if c.UserAgent != "ua" {
		t.Fatalf("expected user_agent ua, got %s", c.UserAgent)
	}
}

// TestResolveAPIURLCorrectsLegacyHosts pins that a config carried over from the
// old XML-RPC or legacy REST API does not silently break every request.
//
// This client only speaks REST v1 (POST /login, GET /subtitles?moviehash=...,
// POST /download). Verified 2026-08-04: rest.opensubtitles.org answers the v1
// search path with 400, while api.opensubtitles.com/api/v1 answers 403 —
// i.e. the endpoint exists and only wants credentials.
func TestResolveAPIURLCorrectsLegacyHosts(t *testing.T) {
	for _, legacy := range []string{
		"https://rest.opensubtitles.org",
		"https://api.opensubtitles.org/xml-rpc",
		"http://www.opensubtitles.org",
	} {
		if got := resolveAPIURL(legacy); got != DefaultAPIURL {
			t.Errorf("resolveAPIURL(%q) = %q, want %q", legacy, got, DefaultAPIURL)
		}
	}
}

// TestResolveAPIURLHonoursDeliberateOverrides is the control. Overriding the
// base URL is how these tests point the client at httptest, and second-guessing
// an operator's deliberate value would be worse than a clear failure — so only
// hosts that provably cannot serve this client are replaced.
func TestResolveAPIURLHonoursDeliberateOverrides(t *testing.T) {
	for _, keep := range []string{
		"http://127.0.0.1:8080",
		"https://api.opensubtitles.com/api/v1",
		"https://mirror.example.com/api/v1",
	} {
		if got := resolveAPIURL(keep); got != keep {
			t.Errorf("resolveAPIURL(%q) = %q, want it unchanged", keep, got)
		}
	}
	if got := resolveAPIURL(""); got != DefaultAPIURL {
		t.Errorf("empty config = %q, want the default", got)
	}
}
