// file: pkg/radarr/client_test.go
package radarr

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

// TestMovies verifies that Movies fetches media items from the Radarr API.
func TestMovies(t *testing.T) {
	var reqPath, apiKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath = r.URL.String()
		apiKey = r.Header.Get("X-Api-Key")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"title":"Movie","movieFile":{"path":"/m.mkv"}},{"title":"Empty","movieFile":{"path":""}}]`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key")
	items, err := c.Movies(context.Background())
	if err != nil {
		t.Fatalf("movies: %v", err)
	}
	if reqPath != "/api/v3/movie?includeMovieFile=true" {
		t.Fatalf("unexpected path %s", reqPath)
	}
	if apiKey != "key" {
		t.Fatalf("expected header key, got %s", apiKey)
	}
	if len(items) != 1 || items[0].Path != "/m.mkv" || items[0].Title != "Movie" {
		t.Fatalf("unexpected items %+v", items)
	}
}

// TestSync verifies that Sync stores new media items only once.
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
	store.InsertMediaItem(&database.MediaItem{Path: "/m1.mkv", Title: "Old"})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"title":"Old","movieFile":{"path":"/m1.mkv"}},{"title":"New","movieFile":{"path":"/m2.mkv"}}]`)
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
		if it.Path == "/m2.mkv" {
			found = true
		}
	}
	if !found {
		t.Fatal("new item not stored")
	}
}

// TestSyncLogsConflicts verifies that Sync logs conflicts when metadata differs.
func TestSyncLogsConflicts(t *testing.T) {
	if err := testutil.CheckSQLiteSupport(); err != nil {
		t.Skipf("SQLite support not available: %v", err)
	}

	store, err := database.OpenSQLStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Existing item with different title should trigger conflict log.
	if err := store.InsertMediaItem(&database.MediaItem{Path: "/c.mkv", Title: "Old"}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"title":"New","movieFile":{"path":"/c.mkv"}}]`)
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
		if strings.Contains(line, "media conflict for /c.mkv") {
			found = true
		}
	}
	if !found {
		t.Fatal("conflict not logged")
	}
}
