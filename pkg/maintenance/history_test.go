// file: pkg/maintenance/history_test.go
// version: 1.0.0
// guid: c8e0a3f5-7b41-4d29-9a06-2f1d6e5b8074

package maintenance

import (
	"context"
	"testing"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/database"
)

// TestPruneDownloadHistory verifies records older than the retention window are
// deleted while newer ones are kept, and that a non-positive retention is a
// no-op.
func TestPruneDownloadHistory(t *testing.T) {
	store, err := database.OpenPebble(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	old := &database.DownloadRecord{File: "old.srt", VideoFile: "v.mkv", Language: "en", CreatedAt: time.Now().Add(-48 * time.Hour)}
	recent := &database.DownloadRecord{File: "new.srt", VideoFile: "v.mkv", Language: "en", CreatedAt: time.Now()}
	if err := store.InsertDownload(old); err != nil {
		t.Fatalf("insert old: %v", err)
	}
	if err := store.InsertDownload(recent); err != nil {
		t.Fatalf("insert recent: %v", err)
	}

	// retention <= 0 is a no-op.
	if n, err := PruneDownloadHistory(context.Background(), store, 0); err != nil || n != 0 {
		t.Fatalf("no-op prune: n=%d err=%v", n, err)
	}

	// Prune everything older than 24h → removes only the old record.
	n, err := PruneDownloadHistory(context.Background(), store, 24*time.Hour)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 pruned, got %d", n)
	}

	remaining, err := store.ListDownloads()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(remaining) != 1 || remaining[0].File != "new.srt" {
		t.Fatalf("expected only new.srt to remain, got %+v", remaining)
	}
}
