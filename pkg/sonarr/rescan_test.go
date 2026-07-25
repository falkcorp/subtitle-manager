// file: pkg/sonarr/rescan_test.go
// version: 1.0.0
// guid: 513902a4-c46e-4d65-978f-3983bd9d7c4a
// last-edited: 2026-07-25

package sonarr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jdfalk/subtitle-manager/pkg/arr"
)

// TestSeriesAppliesPathMapping verifies that series folders come back in
// subtitle-manager's path space, which is what makes MatchPath work without
// having to invert Filters.MapPath.
func TestSeriesAppliesPathMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/series" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Api-Key"); got != "key" {
			t.Errorf("api key = %q, want %q", got, "key")
		}
		_, _ = io.WriteString(w, `[{"id":7,"path":"/data/tv/Show"},{"id":8,"path":"/data/tv/Other"}]`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	c.Filters = arr.Filters{PathMappings: [][2]string{{"/data/tv", "/media/television"}}}

	refs, err := c.Series(context.Background())
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("got %d refs, want 2", len(refs))
	}
	if refs[0].ID != 7 || refs[0].Path != "/media/television/Show" {
		t.Errorf("ref[0] = %+v, want {7 /media/television/Show}", refs[0])
	}
}

// TestSeriesSkipsIncompleteEntries guards against a rescan of series id 0.
func TestSeriesSkipsIncompleteEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"id":0,"path":"/tv/A"},{"id":3,"path":""},{"id":4,"path":"/tv/B"}]`)
	}))
	defer srv.Close()

	refs, err := NewClient(srv.URL, "k").Series(context.Background())
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if len(refs) != 1 || refs[0].ID != 4 {
		t.Fatalf("got %+v, want only series 4", refs)
	}
}

// TestRescanPostsBazarrCommand pins the request shape to Bazarr's
// notify_sonarr: POST /api/v3/command {"name":"RescanSeries","seriesId":N}.
func TestRescanPostsBazarrCommand(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v3/command" {
			t.Errorf("got %s %s, want POST /api/v3/command", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q", ct)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL, "k").Rescan(context.Background(), 42); err != nil {
		t.Fatalf("Rescan: %v", err)
	}
	if got["name"] != "RescanSeries" {
		t.Errorf("name = %v, want RescanSeries", got["name"])
	}
	if id, _ := got["seriesId"].(float64); int(id) != 42 {
		t.Errorf("seriesId = %v, want 42", got["seriesId"])
	}
}

// TestRescanReportsNon2xx ensures a rejected command surfaces as an error so
// the caller can log it, rather than silently appearing to succeed.
func TestRescanReportsNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL, "k").Rescan(context.Background(), 1); err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestMatchPath(t *testing.T) {
	refs := []SeriesRef{
		{ID: 1, Path: "/tv/Show"},
		{ID: 2, Path: "/tv/Show Name"},
		{ID: 3, Path: "/tv/Nested/Inner"},
		{ID: 4, Path: "/tv/Nested"},
	}
	cases := []struct {
		name  string
		video string
		want  int
		ok    bool
	}{
		// "/tv/Show" must not swallow a file under "/tv/Show Name".
		{"separator boundary", "/tv/Show Name/S01E01.mkv", 2, true},
		{"exact folder", "/tv/Show/S01E01.mkv", 1, true},
		// Longest folder wins so a nested library picks the inner series.
		{"longest prefix wins", "/tv/Nested/Inner/S01E01.mkv", 3, true},
		{"outer when not in inner", "/tv/Nested/Loose.mkv", 4, true},
		{"no match", "/movies/Film/Film.mkv", 0, false},
		// The folder itself is not a file inside the folder.
		{"folder is not its own child", "/tv/Show", 0, false},
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

// TestMatchPathAmbiguousReportsNone documents the deliberate choice to refuse a
// guess when two series map onto the same folder: rescanning the wrong series
// is a silent no-op that is painful to debug.
func TestMatchPathAmbiguousReportsNone(t *testing.T) {
	refs := []SeriesRef{{ID: 1, Path: "/tv/Show"}, {ID: 2, Path: "/tv/Show"}}
	if id, ok := MatchPath(refs, "/tv/Show/S01E01.mkv"); ok {
		t.Errorf("got (%d, true), want no match for ambiguous mapping", id)
	}
}

// TestMatchPathTrailingSlashFolder covers *arr instances that report folders
// with a trailing separator.
func TestMatchPathTrailingSlashFolder(t *testing.T) {
	refs := []SeriesRef{{ID: 9, Path: "/tv/Show/"}}
	if id, ok := MatchPath(refs, "/tv/Show/S01E01.mkv"); !ok || id != 9 {
		t.Errorf("got (%d, %v), want (9, true)", id, ok)
	}
}
