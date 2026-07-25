// file: pkg/webserver/wanted_test.go
// version: 1.0.0
// guid: 2d90f4a7-6b13-4c85-a0e2-9f38c714b6de
// last-edited: 2026-07-25

package webserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
)

// useWantedStore points the handler at a throwaway Pebble store and a media
// file it will accept.
//
// Pebble rather than SQLite so these run without the sqlite build tag, which CI
// does not set. Returns the path of a real media file: addToMonitoring skips
// paths that do not exist on disk, so a fabricated one would silently no-op and
// the test would assert nothing.
func useWantedStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	prev := map[string]any{
		"db_backend":      viper.Get("db_backend"),
		"db_path":         viper.Get("db_path"),
		"monitor.enabled": viper.Get("monitor.enabled"),
	}
	t.Cleanup(func() {
		if err := database.CloseSharedStores(); err != nil {
			t.Errorf("CloseSharedStores: %v", err)
		}
		for k, v := range prev {
			viper.Set(k, v)
		}
	})
	viper.Set("db_backend", "pebble")
	viper.Set("db_path", filepath.Join(dir, "db"))

	media := filepath.Join(dir, "Some.Movie.2021.1080p.mkv")
	if err := os.WriteFile(media, []byte("not really a video"), 0o644); err != nil {
		t.Fatalf("write media file: %v", err)
	}
	return media
}

func decodeWanted(t *testing.T, body []byte) []wantedItem {
	t.Helper()
	var got []wantedItem
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode wanted list: %v (body: %s)", err, body)
	}
	return got
}

// TestWantedRoundTrip covers add, list and remove through the handler.
func TestWantedRoundTrip(t *testing.T) {
	media := useWantedStore(t)
	h := wantedHandler()

	// Empty list must be [] rather than null: the UI iterates it directly.
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/wanted", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET empty: got %d (body %s)", rr.Code, rr.Body.String())
	}
	if strings.TrimSpace(rr.Body.String()) == "null" {
		t.Error("GET returned JSON null for an empty list; the UI would have to guard against it")
	}
	if n := len(decodeWanted(t, rr.Body.Bytes())); n != 0 {
		t.Fatalf("expected an empty list, got %d items", n)
	}

	// Add.
	body := `{"path":` + jsonStr(media) + `,"languages":["en","es"]}`
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/wanted", strings.NewReader(body)))
	if rr.Code != http.StatusCreated {
		t.Fatalf("POST: got %d (body %s)", rr.Code, rr.Body.String())
	}

	// List reflects it, with languages decoded into a real array.
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/wanted", nil))
	items := decodeWanted(t, rr.Body.Bytes())
	if len(items) != 1 {
		t.Fatalf("expected 1 wanted item, got %d", len(items))
	}
	if items[0].Path != media {
		t.Errorf("path = %q, want %q", items[0].Path, media)
	}
	if len(items[0].Languages) != 2 {
		t.Errorf("languages = %v, want two entries decoded from the stored JSON string",
			items[0].Languages)
	}

	// Remove.
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodDelete, "/api/wanted",
		strings.NewReader(`{"path":`+jsonStr(media)+`}`)))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("DELETE: got %d (body %s)", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/wanted", nil))
	if n := len(decodeWanted(t, rr.Body.Bytes())); n != 0 {
		t.Errorf("expected the list to be empty after DELETE, got %d items", n)
	}
}

// TestWantedRejectsBadInput verifies the handler validates before touching the
// store.
func TestWantedRejectsBadInput(t *testing.T) {
	useWantedStore(t)
	h := wantedHandler()

	for name, tc := range map[string]struct {
		method, body string
		want         int
	}{
		"empty path":         {http.MethodPost, `{"path":""}`, http.StatusBadRequest},
		"relative path":      {http.MethodPost, `{"path":"relative/file.mkv"}`, http.StatusBadRequest},
		"malformed json":     {http.MethodPost, `{`, http.StatusBadRequest},
		"bad language":       {http.MethodPost, `{"path":"/tmp/a.mkv","languages":["../etc"]}`, http.StatusBadRequest},
		"delete empty path":  {http.MethodDelete, `{"path":""}`, http.StatusBadRequest},
		"unsupported method": {http.MethodPatch, ``, http.StatusMethodNotAllowed},
	} {
		t.Run(name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest(tc.method, "/api/wanted", strings.NewReader(tc.body)))
			if rr.Code != tc.want {
				t.Errorf("got %d, want %d (body %s)", rr.Code, tc.want, rr.Body.String())
			}
		})
	}
}

// TestStartMonitoringDefaultsOff is a safety test, not a behaviour test.
//
// The monitoring loop calls scanner.ProcessFile with upgrade enabled, which
// contacts subtitle providers and writes files into the operator's media
// directories on a timer. If it ever started without an explicit opt-in, a
// server upgrade would silently begin modifying a media library. This asserts
// the default stays off.
func TestStartMonitoringDefaultsOff(t *testing.T) {
	useWantedStore(t)
	viper.Set("monitor.enabled", nil)

	if viper.GetBool("monitor.enabled") {
		t.Fatal("monitor.enabled defaults to true; the monitoring loop would " +
			"download subtitles into the user's media directories without being asked")
	}

	// Must return promptly without starting anything.
	startMonitoring(context.Background())
}

// TestMonitorIntervalNeverZero guards against a startup panic: time.NewTicker
// panics on a non-positive duration, and monitor.interval is routinely unset.
func TestMonitorIntervalNeverZero(t *testing.T) {
	prev := viper.Get("monitor.interval")
	t.Cleanup(func() { viper.Set("monitor.interval", prev) })

	for _, v := range []any{nil, 0, -5} {
		viper.Set("monitor.interval", v)
		if got := monitorInterval(); got <= 0 {
			t.Errorf("monitor.interval=%v gave %v; time.NewTicker would panic", v, got)
		}
	}
}

// jsonStr quotes s as a JSON string.
func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
