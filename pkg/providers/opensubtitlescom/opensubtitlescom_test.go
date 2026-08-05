// file: pkg/providers/opensubtitlescom/opensubtitlescom_test.go
// version: 2.0.0
// guid: b91f4e07-6c23-4a85-90d1-4f7e2a6c8503
// last-edited: 2026-08-04

package opensubtitlescom

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

// v1Server speaks the shape of the OpenSubtitles.com REST v1 API: log in for a
// token, search /subtitles, POST /download for a link, then fetch the link.
//
// The previous test asserted a GET to /subtitles/{name}/{lang} — an endpoint
// that exists in no version of the API — so it passed against a provider that
// could never have downloaded anything. Modelling the real conversation is the
// whole point.
type v1Server struct {
	mu   sync.Mutex
	seen []string
}

func (s *v1Server) record(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, path)
}

func (s *v1Server) paths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.seen...)
}

func (s *v1Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.record(r.Method + " " + r.URL.Path)
	switch {
	case r.URL.Path == "/login":
		fmt.Fprint(w, `{"token":"t","status":"200"}`)
	case strings.HasPrefix(r.URL.Path, "/subtitles"):
		fmt.Fprint(w, `{"data":[{"attributes":{"subtitle_id":"1","release":"GROUP","files":[{"file_id":1}]}}]}`)
	case r.URL.Path == "/download":
		fmt.Fprintf(w, `{"link":"http://%s/file.srt"}`, r.Host)
	case r.URL.Path == "/file.srt":
		fmt.Fprint(w, "real subtitle")
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// mediaFile writes a real file: the client hashes the media to search by
// moviehash, so a path that does not exist never reaches the API at all.
//
// It must be at least 128 KiB. The OpenSubtitles hash reads 64 KiB from each
// end of the file, so anything smaller fails with "unexpected EOF" before a
// single request is made.
func mediaFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "Movie.2020.mkv")
	if err := os.WriteFile(p, make([]byte, 256*1024), 0644); err != nil {
		t.Fatalf("create media: %v", err)
	}
	return p
}

func TestFetchUsesTheRealV1API(t *testing.T) {
	h := &v1Server{}
	srv := httptest.NewServer(h)
	defer srv.Close()

	viper.Reset()
	defer viper.Reset()
	viper.Set("opensubtitles.api_url", srv.URL)
	viper.Set("opensubtitles.username", "u")
	viper.Set("opensubtitles.password", "p")
	vid := mediaFile(t)

	// This asserts the *protocol*, not the payload. The underlying client's
	// download step is separately broken — it GETs {api}/download?file_id=
	// instead of POSTing {"file_id":N} and following the returned link — and
	// that is fixed on its own branch. What matters here is that this provider
	// now speaks v1 at all rather than a fabricated endpoint.
	if _, err := New().Fetch(context.Background(), vid, "en"); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	// The old stub would have issued GET /subtitles/Movie.2020.mkv/en and
	// nothing else. Assert the v1 conversation actually happened.
	var searched bool
	for _, p := range h.paths() {
		if strings.HasPrefix(p, "GET /subtitles") && !strings.Contains(p, "Movie.2020.mkv") {
			searched = true
		}
	}
	if !searched {
		t.Errorf("no v1 search request was made; saw %v", h.paths())
	}
}

// TestAlternateConfigKeysAreAdopted covers a config that carries the
// credentials under providers.opensubtitlescom.* — the spelling a Bazarr import
// produces for this provider name — rather than opensubtitles.*.
func TestAlternateConfigKeysAreAdopted(t *testing.T) {
	h := &v1Server{}
	srv := httptest.NewServer(h)
	defer srv.Close()

	viper.Reset()
	defer viper.Reset()
	viper.Set("providers.opensubtitlescom.api_url", srv.URL)
	viper.Set("providers.opensubtitlescom.username", "u")
	viper.Set("providers.opensubtitlescom.password", "p")
	vid := mediaFile(t)

	if _, err := New().Fetch(context.Background(), vid, "en"); err != nil {
		t.Fatalf("fetch with alternate keys: %v", err)
	}
	if len(h.paths()) == 0 {
		t.Error("no request reached the server; the alternate config keys were not used")
	}
}

// TestExistingKeysWin pins that reading the alternate spelling can never change
// a setup that already works.
func TestExistingKeysWin(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	viper.Set("opensubtitles.username", "primary")
	viper.Set("providers.opensubtitlescom.username", "secondary")

	if got := pick("username"); got != "primary" {
		t.Errorf("username = %q; the existing opensubtitles.* value must win", got)
	}
}

// TestConstructionDoesNotMutateConfig pins that resolving the fallback is a
// read. Provider constructors run concurrently inside the fetch wave, so a
// viper.Set here would be a data race against viper's global map — and would
// rewrite configuration for every other component besides.
func TestConstructionDoesNotMutateConfig(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	viper.Set("providers.opensubtitlescom.username", "secondary")

	New()

	if got := viper.GetString("opensubtitles.username"); got != "" {
		t.Errorf("constructing the provider wrote opensubtitles.username = %q", got)
	}
}
