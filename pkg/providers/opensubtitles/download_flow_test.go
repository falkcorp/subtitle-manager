// file: pkg/providers/opensubtitles/download_flow_test.go
// version: 1.1.0
// guid: 4c02e7b3-58a1-4d96-b7e2-1a05f39c684d
// last-edited: 2026-08-04

package opensubtitles

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

// strictV1 implements the download contract as documented, and rejects anything
// else. That strictness is the point: the previous mock served the subtitle
// directly from GET /download, so it passed against a client that never made
// the POST the real API requires.
//
// Contract (OpenSubtitles REST API v1):
//   - POST /download with JSON body {"file_id": N} -> {"link": "..."}
//   - GET that link -> the subtitle bytes
//   - Api-Key and Authorization headers on authenticated requests
//   - file_id comes from attributes.files[].file_id, not attributes.subtitle_id
type strictV1 struct {
	mu       sync.Mutex
	gotKey   string
	gotAuth  string
	gotBody  map[string]int
	postSeen bool
}

func (s *strictV1) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/login":
		fmt.Fprint(w, `{"token":"tok","status":"200"}`)

	case strings.HasPrefix(r.URL.Path, "/subtitles"):
		// subtitle_id and file_id deliberately differ, so a client reading the
		// wrong field asks for a file that does not exist.
		fmt.Fprint(w, `{"data":[{"attributes":{"subtitle_id":"999","release":"GROUP","files":[{"file_id":4242,"file_name":"s.srt"}]}}]}`)

	case r.URL.Path == "/download":
		if r.Method != http.MethodPost {
			http.Error(w, "download requires POST", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]int
		if err := json.Unmarshal(body, &parsed); err != nil {
			http.Error(w, "download requires a JSON body", http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		s.postSeen = true
		s.gotBody = parsed
		s.gotKey = r.Header.Get("Api-Key")
		s.gotAuth = r.Header.Get("Authorization")
		s.mu.Unlock()

		if parsed["file_id"] != 4242 {
			http.Error(w, "unknown file_id", http.StatusNotFound)
			return
		}
		fmt.Fprintf(w, `{"link":"http://%s/cdn/s.srt","remaining":42}`, r.Host)

	case r.URL.Path == "/cdn/s.srt":
		fmt.Fprint(w, "1\n00:00:01,000 --> 00:00:02,000\nhello\n")

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func testMedia(t *testing.T) string {
	t.Helper()
	// At least 128 KiB: the OpenSubtitles hash reads 64 KiB from each end, so a
	// smaller file fails before any request is made.
	p := filepath.Join(t.TempDir(), "Movie.2020.mkv")
	if err := os.WriteFile(p, make([]byte, 256*1024), 0644); err != nil {
		t.Fatalf("create media: %v", err)
	}
	return p
}

// TestFetchFollowsTheDownloadLink is the regression. Before this, Fetch issued
// GET /download?file_id=<subtitle_id> and returned that response body verbatim
// — the JSON envelope at best, never a subtitle.
func TestFetchFollowsTheDownloadLink(t *testing.T) {
	h := &strictV1{}
	srv := httptest.NewServer(h)
	defer srv.Close()

	viper.Reset()
	defer viper.Reset()
	viper.Set("opensubtitles.api_url", srv.URL)
	viper.Set("opensubtitles.username", "u")
	viper.Set("opensubtitles.password", "p")
	viper.Set("opensubtitles.api_key", "secret-key")

	got, err := New("").Fetch(context.Background(), testMedia(t), "en")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !strings.Contains(string(got), "hello") {
		t.Fatalf("got %q, want the subtitle behind the download link", got)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.postSeen {
		t.Error("no POST /download was made")
	}
	// file_id, not subtitle_id: the mock returns 999 for the latter and 4242
	// for the former, so reading the wrong field 404s.
	if h.gotBody["file_id"] != 4242 {
		t.Errorf("file_id = %d, want 4242 from attributes.files[].file_id", h.gotBody["file_id"])
	}
	if h.gotKey != "secret-key" {
		t.Errorf("Api-Key header = %q, want the configured key", h.gotKey)
	}
	if h.gotAuth == "" {
		t.Error("Authorization header missing")
	}
}

// TestQuotaExhaustedIsReported covers the response a real account gets once its
// daily allowance runs out: 200, a message, and no link. Returning that body as
// if it were a subtitle would write a JSON file to the media directory.
func TestQuotaExhaustedIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			fmt.Fprint(w, `{"token":"tok","status":"200"}`)
			return
		}
		fmt.Fprint(w, `{"message":"You have downloaded your allowed subtitles for today","remaining":0}`)
	}))
	defer srv.Close()

	viper.Reset()
	defer viper.Reset()
	viper.Set("opensubtitles.api_url", srv.URL)
	viper.Set("opensubtitles.username", "u")
	viper.Set("opensubtitles.password", "p")

	c := New("")
	_, err := c.resolveDownloadLink(context.Background(), 4242)
	if err == nil {
		t.Fatal("quota exhaustion was not reported as an error")
	}
	if !strings.Contains(err.Error(), "allowed subtitles") {
		t.Errorf("error %q does not surface the API's message", err)
	}
}

// TestFileIDPrefersFilesArray pins which field the id comes from.
func TestFileIDPrefersFilesArray(t *testing.T) {
	var r SearchResult
	r.Attributes.SubtitleID = "999"
	if _, ok := fileIDForResult(r); ok {
		t.Error("returned an id with no files[]; subtitle_id is not a file id")
	}

	r.Attributes.Files = append(r.Attributes.Files, struct {
		FileID   int    `json:"file_id"`
		CDNumber int    `json:"cd_number"`
		FileName string `json:"file_name"`
	}{FileID: 4242, FileName: "s.srt"})

	id, ok := fileIDForResult(r)
	if !ok || id != 4242 {
		t.Errorf("got (%d, %t), want (4242, true)", id, ok)
	}
}

// TestBazarrImportedCredentialsAreUsed pins the fallback that makes an imported
// config work.
//
// pkg/bazarr.mapProviderSettings writes credentials under
// providers.opensubtitles.*, while this client only ever read the top-level
// opensubtitles.* spelling. Importing a Bazarr config — the documented
// migration path — therefore produced a config containing a username and
// password that nothing read, leaving the provider unauthenticated with no
// indication why. Confirmed on a real imported config before this was fixed.
func TestBazarrImportedCredentialsAreUsed(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	viper.Set("providers.opensubtitles.username", "imported-user")
	viper.Set("providers.opensubtitles.password", "imported-pass")
	viper.Set("providers.opensubtitles.api_key", "imported-key")

	c := New("")
	if c.username != "imported-user" || c.password != "imported-pass" {
		t.Errorf("credentials not adopted: username=%q", c.username)
	}
	if c.APIKey != "imported-key" {
		t.Errorf("APIKey = %q, want the imported key", c.APIKey)
	}
}

// TestTopLevelCredentialsWin is the control: the fallback must never override a
// config that already works.
func TestTopLevelCredentialsWin(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	viper.Set("opensubtitles.username", "primary")
	viper.Set("providers.opensubtitles.username", "imported")

	if got := New("").username; got != "primary" {
		t.Errorf("username = %q, want the top-level value", got)
	}
}
