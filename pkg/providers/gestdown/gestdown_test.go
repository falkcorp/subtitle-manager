// file: pkg/providers/gestdown/gestdown_test.go
// version: 2.0.0
// guid: bfe9a67e-a423-4db3-a78d-1d298b8507c5
// last-edited: 2026-07-23

package gestdown

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// newTestClient returns a Client pointed at srv with its transport.
func newTestClient(srv *httptest.Server) *Client {
	return &Client{
		APIURL:     srv.URL,
		HTTPClient: srv.Client(),
		UserAgent:  "subtitle-manager-test",
	}
}

func TestNewDefaults(t *testing.T) {
	c := New()
	if c.APIURL != "https://api.gestdown.info" {
		t.Fatalf("expected default gestdown.info APIURL, got %q", c.APIURL)
	}
	if c.HTTPClient == nil {
		t.Fatal("expected HTTP client")
	}
	if c.UserAgent == "" {
		t.Fatal("expected a non-empty default User-Agent")
	}
}

// router dispatches on the decoded request path. A plain handler is used
// instead of http.ServeMux because Gestdown paths contain spaces (e.g.
// "/shows/search/Breaking Bad"), which the Go 1.22+ ServeMux pattern parser
// rejects.
func router(t *testing.T, routes map[string]http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		h, ok := routes[r.URL.Path]
		if !ok {
			t.Fatalf("unexpected request path %q", r.URL.Path)
			return
		}
		h(w, r)
	}
}

func TestFetchEpisodeSuccess(t *testing.T) {
	sawUA := false
	srv := httptest.NewServer(router(t, map[string]http.HandlerFunc{
		"/shows/search/Breaking Bad": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("User-Agent") != "" {
				sawUA = true
			}
			_, _ = w.Write([]byte(`{"shows":[{"id":"show-uuid","name":"Breaking Bad","seasons":[1,2,3,4,5]}]}`))
		},
		"/subtitles/get/show-uuid/1/1/en": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"matchingSubtitles":[
				{"subtitleId":"a","version":"draft","completed":false,"downloadUri":"/subtitles/download/a","language":"English"},
				{"subtitleId":"b","version":"0tv","completed":true,"downloadUri":"/subtitles/download/b","language":"English"}
			]}`))
		},
		"/subtitles/download/b": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/srt")
			_, _ = w.Write([]byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n"))
		},
	}))
	defer srv.Close()

	c := newTestClient(srv)
	data, err := c.Fetch(context.Background(), "/media/Breaking.Bad.S01E01.1080p.WEB.mkv", "en")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !strings.Contains(string(data), "Hello") {
		t.Fatalf("expected subtitle body, got %q", string(data))
	}
	if !sawUA {
		t.Fatal("expected a User-Agent header to be sent")
	}
}

func TestFetchSkipsIncompleteSubtitle(t *testing.T) {
	// Only an incomplete subtitle is offered; Fetch must not return it.
	srv := httptest.NewServer(router(t, map[string]http.HandlerFunc{
		"/shows/search/Breaking Bad": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"shows":[{"id":"show-uuid","name":"Breaking Bad","seasons":[1]}]}`))
		},
		"/subtitles/get/show-uuid/1/1/en": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"matchingSubtitles":[{"subtitleId":"a","completed":false,"downloadUri":"/subtitles/download/a"}]}`))
		},
	}))
	defer srv.Close()

	c := newTestClient(srv)
	if _, err := c.Fetch(context.Background(), "/media/Breaking.Bad.S01E01.mkv", "en"); err == nil {
		t.Fatal("expected error when only incomplete subtitles are available")
	}
}

func TestFetchMovieUnsupported(t *testing.T) {
	// A movie file must be rejected without any HTTP call.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("no request expected for a movie, got %s", r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Fetch(context.Background(), "/media/Inception (2010).mkv", "en")
	if err == nil {
		t.Fatal("expected error for a movie file")
	}
	if !strings.Contains(err.Error(), "only TV episodes") {
		t.Fatalf("expected TV-only error, got %v", err)
	}
}

func TestFetchShowNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/shows/search/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		t.Fatalf("unexpected path %q after show 404", r.URL.Path)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	if _, err := c.Fetch(context.Background(), "/media/No.Such.Show.S01E01.mkv", "en"); err == nil {
		t.Fatal("expected error when the show is not found")
	}
}

func TestFetchPrefersSeasonMatchingShow(t *testing.T) {
	// Two shows share a name; only the second lists season 3. The
	// season-matching show is ordered first, so the other is never queried.
	srv := httptest.NewServer(router(t, map[string]http.HandlerFunc{
		"/shows/search/The Office": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"shows":[
				{"id":"uk","name":"The Office","seasons":[1,2]},
				{"id":"us","name":"The Office","seasons":[1,2,3,4,5]}
			]}`))
		},
		"/subtitles/get/us/3/2/en": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"matchingSubtitles":[{"subtitleId":"z","completed":true,"downloadUri":"/subtitles/download/z"}]}`))
		},
		"/subtitles/download/z": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("SRT"))
		},
		"/subtitles/get/uk/3/2/en": func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("season-matching show should be tried first; uk should not be queried")
		},
	}))
	defer srv.Close()

	c := newTestClient(srv)
	data, err := c.Fetch(context.Background(), "/media/The.Office.S03E02.mkv", "en")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if string(data) != "SRT" {
		t.Fatalf("expected SRT body, got %q", string(data))
	}
}

func TestFetchTransportError(t *testing.T) {
	transportErr := errors.New("transport failure")
	c := &Client{
		APIURL: "https://example.invalid",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, transportErr
			}),
		},
	}
	_, err := c.Fetch(context.Background(), "/media/Breaking.Bad.S01E01.mkv", "en")
	if !errors.Is(err, transportErr) {
		t.Fatalf("expected transport error, got %v", err)
	}
}

func TestListSubtitles404IsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	subs, err := c.listSubtitles(context.Background(), "show-uuid", 1, 1, "en")
	if err != nil {
		t.Fatalf("expected 404 to be treated as empty, got error %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("expected no subtitles, got %d", len(subs))
	}
}
