// file: pkg/radarr/rescan_test.go
// version: 1.0.0
// guid: 8b3652b2-6d65-41b6-aa42-583159582761
// last-edited: 2026-07-25

package radarr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jdfalk/subtitle-manager/pkg/arr"
)

// TestMovieRefsAppliesPathMapping verifies movie folders come back in
// subtitle-manager's path space so MatchPath never has to invert MapPath.
func TestMovieRefsAppliesPathMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/movie" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `[{"id":11,"path":"/data/movies/Film (2020)"}]`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	c.Filters = arr.Filters{PathMappings: [][2]string{{"/data/movies", "/media/films"}}}

	refs, err := c.MovieRefs(context.Background())
	if err != nil {
		t.Fatalf("MovieRefs: %v", err)
	}
	if len(refs) != 1 || refs[0].ID != 11 || refs[0].Path != "/media/films/Film (2020)" {
		t.Fatalf("got %+v, want [{11 /media/films/Film (2020)}]", refs)
	}
}

// TestRescanPostsBazarrCommand pins the request shape to Bazarr's
// notify_radarr: POST /api/v3/command {"name":"RescanMovie","movieId":N}.
func TestRescanPostsBazarrCommand(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v3/command" {
			t.Errorf("got %s %s, want POST /api/v3/command", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL, "k").Rescan(context.Background(), 11); err != nil {
		t.Fatalf("Rescan: %v", err)
	}
	if got["name"] != "RescanMovie" {
		t.Errorf("name = %v, want RescanMovie", got["name"])
	}
	if id, _ := got["movieId"].(float64); int(id) != 11 {
		t.Errorf("movieId = %v, want 11", got["movieId"])
	}
}

func TestRescanReportsNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL, "k").Rescan(context.Background(), 1); err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestMatchPath(t *testing.T) {
	refs := []MovieRef{
		{ID: 1, Path: "/movies/Alien"},
		{ID: 2, Path: "/movies/Aliens"},
	}
	cases := []struct {
		name  string
		video string
		want  int
		ok    bool
	}{
		// The separator check is what stops "Alien" matching "Aliens".
		{"separator boundary", "/movies/Aliens/Aliens.mkv", 2, true},
		{"exact folder", "/movies/Alien/Alien.mkv", 1, true},
		{"no match", "/tv/Show/S01E01.mkv", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := MatchPath(refs, tc.video)
			if got != tc.want || ok != tc.ok {
				t.Errorf("MatchPath(%q) = (%d, %v), want (%d, %v)", tc.video, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestMatchPathAmbiguousReportsNone(t *testing.T) {
	refs := []MovieRef{{ID: 1, Path: "/movies/Film"}, {ID: 2, Path: "/movies/Film"}}
	if id, ok := MatchPath(refs, "/movies/Film/Film.mkv"); ok {
		t.Errorf("got (%d, true), want no match for ambiguous mapping", id)
	}
}
