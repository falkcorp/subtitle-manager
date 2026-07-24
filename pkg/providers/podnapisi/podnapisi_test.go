// file: pkg/providers/podnapisi/podnapisi_test.go
// version: 2.0.0
// guid: 807c1ff7-3f08-41a0-9fc1-ceb5f3584e3b
// last-edited: 2026-07-24

package podnapisi

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient returns a Client pointed at srv with its transport.
func newTestClient(srv *httptest.Server) *Client {
	return &Client{
		APIURL:     srv.URL,
		HTTPClient: srv.Client(),
		UserAgent:  "subtitle-manager-test",
	}
}

// zipWith builds an in-memory ZIP archive containing the given name→content
// files, mirroring how Podnapisi serves downloads.
func zipWith(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %q: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func TestNewDefaults(t *testing.T) {
	c := New()
	if c.APIURL != "https://www.podnapisi.net/subtitles" {
		t.Fatalf("unexpected default APIURL %q", c.APIURL)
	}
	if c.HTTPClient == nil {
		t.Fatal("expected HTTP client")
	}
	if c.UserAgent == "" {
		t.Fatal("expected a non-empty default User-Agent")
	}
}

func TestFetchEpisode(t *testing.T) {
	const srt = "1\n00:00:01,000 --> 00:00:02,000\nhello\n"
	var gotSearch, gotDownload *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search/advanced":
			gotSearch = r
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"page":1,"all_pages":1,"data":[
				{"id":"2581","language":"en","movie":{"type":"tv-series"}}
			]}`))
		case strings.HasSuffix(r.URL.Path, "/download"):
			gotDownload = r
			_, _ = w.Write(zipWith(t, map[string]string{"the.big.bang.theory.s07e05.srt": srt}))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv)
	data, err := c.Fetch(context.Background(), "The.Big.Bang.Theory.S07E05.1080p.BluRay.x264.mkv", "en")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(data) != srt {
		t.Fatalf("subtitle bytes = %q, want %q", data, srt)
	}

	// Verify the episode search parameters.
	q := gotSearch.URL.Query()
	if !strings.Contains(strings.ToLower(q.Get("keywords")), "big bang") {
		t.Fatalf("keywords = %q, want the series title", q.Get("keywords"))
	}
	if q.Get("language") != "en" {
		t.Fatalf("language = %q, want en", q.Get("language"))
	}
	if q.Get("seasons") != "7" || q.Get("episodes") != "5" {
		t.Fatalf("seasons/episodes = %q/%q, want 7/5", q.Get("seasons"), q.Get("episodes"))
	}
	if types := q["movie_type"]; len(types) != 2 || types[0] != "tv-series" || types[1] != "mini-series" {
		t.Fatalf("movie_type = %v, want [tv-series mini-series]", types)
	}
	if gotSearch.Header.Get("Accept") != "application/json" {
		t.Fatalf("Accept = %q, want application/json", gotSearch.Header.Get("Accept"))
	}

	// Verify the download requested a zip container for the selected id.
	if !strings.Contains(gotDownload.URL.Path, "2581") {
		t.Fatalf("download path %q missing subtitle id", gotDownload.URL.Path)
	}
	if gotDownload.URL.Query().Get("container") != "zip" {
		t.Fatalf("download container = %q, want zip", gotDownload.URL.Query().Get("container"))
	}
}

func TestFetchMovie(t *testing.T) {
	const srt = "movie subtitle"
	var gotSearch *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search/advanced":
			gotSearch = r
			_, _ = w.Write([]byte(`{"page":1,"all_pages":1,"data":[
				{"id":"99","language":"en","movie":{"type":"movie"}}
			]}`))
		case strings.HasSuffix(r.URL.Path, "/download"):
			_, _ = w.Write(zipWith(t, map[string]string{"inception.srt": srt}))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv)
	data, err := c.Fetch(context.Background(), "Inception (2010).mkv", "en")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(data) != srt {
		t.Fatalf("subtitle bytes = %q, want %q", data, srt)
	}
	q := gotSearch.URL.Query()
	if q.Get("movie_type") != "movie" {
		t.Fatalf("movie_type = %q, want movie", q.Get("movie_type"))
	}
	if q.Get("year") != "2010" {
		t.Fatalf("year = %q, want 2010", q.Get("year"))
	}
	if q.Get("seasons") != "" || q.Get("episodes") != "" {
		t.Fatal("movie query must not set seasons/episodes")
	}
}

func TestFetchNoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"page":1,"all_pages":1,"data":[]}`))
	}))
	defer srv.Close()

	_, err := newTestClient(srv).Fetch(context.Background(), "Show.S01E01.mkv", "en")
	if err == nil || !strings.Contains(err.Error(), "no subtitle found") {
		t.Fatalf("expected no-subtitle error, got %v", err)
	}
}

func TestFetchTypeMismatchSkipped(t *testing.T) {
	// An episode query that only returns a movie result must find no match
	// rather than downloading the wrong media type.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/advanced" {
			t.Fatalf("download must not be attempted; got path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"page":1,"all_pages":1,"data":[
			{"id":"7","language":"en","movie":{"type":"movie"}}
		]}`))
	}))
	defer srv.Close()

	_, err := newTestClient(srv).Fetch(context.Background(), "Show.S01E01.mkv", "en")
	if err == nil || !strings.Contains(err.Error(), "no subtitle found") {
		t.Fatalf("expected no-subtitle error for type mismatch, got %v", err)
	}
}

func TestFetchSearchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newTestClient(srv).Fetch(context.Background(), "Show.S01E01.mkv", "en")
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("expected status 500 error, got %v", err)
	}
}

func TestExtractSubtitlePrefersSrt(t *testing.T) {
	archive := zipWith(t, map[string]string{
		"readme.txt": "ignore me",
		"movie.srt":  "the real subtitle",
		"movie.nfo":  "not a subtitle",
	})
	data, err := extractSubtitle(archive)
	if err != nil {
		t.Fatalf("extractSubtitle: %v", err)
	}
	if string(data) != "the real subtitle" {
		t.Fatalf("got %q, want the .srt content", data)
	}
}

func TestExtractSubtitleNone(t *testing.T) {
	archive := zipWith(t, map[string]string{"cover.jpg": "binary"})
	if _, err := extractSubtitle(archive); err == nil {
		t.Fatal("expected error when archive holds no subtitle file")
	}
}
