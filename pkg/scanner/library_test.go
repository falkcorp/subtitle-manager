// file: pkg/scanner/library_test.go
// version: 1.0.0
// guid: 39c3dc5d-d0bf-489d-8c43-0b33d257c01c

package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	providersmocks "github.com/jdfalk/subtitle-manager/pkg/providers/mocks"
)

// TestProcessLibrary verifies the library-search bridge fetches subtitles for
// media items recorded in the store and skips items whose file is missing.
func TestProcessLibrary(t *testing.T) {
	dir := t.TempDir()
	vid := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	viper.Set("media_directory", dir)
	defer viper.Reset()

	store, err := database.OpenPebble(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	// A real library item pointing at an existing file, plus a stale item whose
	// file no longer exists (should be skipped, not error).
	if err := store.InsertMediaItem(&database.MediaItem{Path: vid, Title: "Movie"}); err != nil {
		t.Fatalf("insert item: %v", err)
	}
	if err := store.InsertMediaItem(&database.MediaItem{Path: filepath.Join(dir, "gone.mkv"), Title: "Gone"}); err != nil {
		t.Fatalf("insert stale item: %v", err)
	}

	m := providersmocks.NewMockProvider(t)
	// Fetch must be called exactly once — only for the existing file.
	m.On("Fetch", mock.Anything, vid, "en").Return([]byte("sub"), nil).Once()

	if err := ProcessLibrary(context.Background(), "en", "test", m, false, 2, store); err != nil {
		t.Fatalf("process library: %v", err)
	}
	m.AssertExpectations(t)

	data, err := os.ReadFile(filepath.Join(dir, "movie.en.srt"))
	if err != nil {
		t.Fatalf("read subtitle: %v", err)
	}
	if string(data) != "sub" {
		t.Fatalf("unexpected subtitle %q", data)
	}
}

// TestProcessLibraryNilStore verifies a nil store is a clear error, not a panic.
func TestProcessLibraryNilStore(t *testing.T) {
	if err := ProcessLibrary(context.Background(), "en", "test", nil, false, 1, nil); err == nil {
		t.Fatal("expected error for nil store")
	}
}

// TestProcessLibraryBestEffort verifies one item's fetch failure does not abort
// the sweep: a second, healthy item still gets its subtitle.
func TestProcessLibraryBestEffort(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.mkv")
	bad := filepath.Join(dir, "bad.mkv")
	for _, p := range []string{good, bad} {
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatalf("create %s: %v", p, err)
		}
	}
	viper.Set("media_directory", dir)
	defer viper.Reset()

	store, err := database.OpenPebble(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	for _, p := range []string{bad, good} {
		if err := store.InsertMediaItem(&database.MediaItem{Path: p}); err != nil {
			t.Fatalf("insert %s: %v", p, err)
		}
	}

	m := providersmocks.NewMockProvider(t)
	m.On("Fetch", mock.Anything, bad, "en").Return([]byte(nil), context.DeadlineExceeded)
	m.On("Fetch", mock.Anything, good, "en").Return([]byte("ok"), nil)

	if err := ProcessLibrary(context.Background(), "en", "test", m, false, 1, store); err != nil {
		t.Fatalf("process library returned error despite best-effort: %v", err)
	}

	if data, err := os.ReadFile(filepath.Join(dir, "good.en.srt")); err != nil || string(data) != "ok" {
		t.Fatalf("healthy item not processed after a failure: data=%q err=%v", data, err)
	}
}
