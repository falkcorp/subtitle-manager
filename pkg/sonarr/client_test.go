// file: pkg/sonarr/client_test.go
package sonarr

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/testutil"
)

// TestEpisodes verifies that Episodes fetches media items from the Sonarr API.
func TestEpisodes(t *testing.T) {
	var reqPath, apiKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath = r.URL.String()
		apiKey = r.Header.Get("X-Api-Key")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"series":{"title":"Show"},"episodeFile":{"path":"/s.mkv"},"seasonNumber":1,"episodeNumber":2},{"series":{"title":"Empty"},"episodeFile":{"path":""}}]`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	items, err := c.Episodes(context.Background())
	if err != nil {
		t.Fatalf("episodes: %v", err)
	}
	if reqPath != "/api/v3/episode?includeEpisodeFile=true" {
		t.Fatalf("unexpected path %s", reqPath)
	}
	if apiKey != "key" {
		t.Fatalf("expected header key, got %s", apiKey)
	}
	if len(items) != 1 || items[0].Path != "/s.mkv" || items[0].Season != 1 || items[0].Episode != 2 {
		t.Fatalf("unexpected items %+v", items)
	}
}

// TestSync verifies that Sync stores new episodes only once.
func TestSync(t *testing.T) {
	// This test uses SQLite database, skip if not available
	if err := testutil.CheckSQLiteSupport(); err != nil {
		t.Skipf("SQLite support not available: %v", err)
	}

	store, err := database.OpenSQLStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	// Pre-existing item should not be inserted again.
	store.InsertMediaItem(&database.MediaItem{Path: "/old.mkv", Title: "Show", Season: 1, Episode: 1})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"series":{"title":"Show"},"episodeFile":{"path":"/old.mkv"},"seasonNumber":1,"episodeNumber":1},{"series":{"title":"Show"},"episodeFile":{"path":"/new.mkv"},"seasonNumber":1,"episodeNumber":2}]`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if err := Sync(context.Background(), c, store); err != nil {
		t.Fatalf("sync: %v", err)
	}
	items, err := store.ListMediaItems()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	var found bool
	for _, it := range items {
		if it.Path == "/new.mkv" {
			found = true
		}
	}
	if !found {
		t.Fatal("new item not stored")
	}
}

// TestSyncLogsConflicts verifies that Sync logs metadata conflicts.
func TestSyncLogsConflicts(t *testing.T) {
	if err := testutil.CheckSQLiteSupport(); err != nil {
		t.Skipf("SQLite support not available: %v", err)
	}

	store, err := database.OpenSQLStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Existing item with different metadata triggers conflict.
	if err := store.InsertMediaItem(&database.MediaItem{Path: "/e.mkv", Title: "Show", Season: 1, Episode: 1}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"series":{"title":"Show"},"episodeFile":{"path":"/e.mkv"},"seasonNumber":1,"episodeNumber":2}]`)
	}))
	defer srv.Close()

	// A large buffer, and the previous hook restored afterwards.
	//
	// logging.Hook is a package-level global shared by every test in this
	// package, and NewMemoryHook is a ring buffer. At capacity 10 this test
	// passed in isolation and failed in a full package run: log lines from
	// other tests' still-running goroutines evicted the conflict line before
	// it was read. Leaving the hook installed also leaked it into later tests.
	prevHook := logging.Hook
	logging.SetHook(logging.NewMemoryHook(1000))
	t.Cleanup(func() { logging.SetHook(prevHook) })
	c := NewClient(srv.URL, "")
	if err := Sync(context.Background(), c, store); err != nil {
		t.Fatalf("sync: %v", err)
	}

	found := false
	for _, line := range logging.Hook.Logs() {
		if strings.Contains(line, "media conflict for /e.mkv") {
			found = true
		}
	}
	if !found {
		t.Fatal("conflict not logged")
	}
}
