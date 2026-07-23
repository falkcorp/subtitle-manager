// file: pkg/monitoring/blacklist_persist_test.go
// version: 1.0.0
// guid: 2d7b9e04-6a13-4c58-8f21-0e5c3a9b7146

package monitoring

import (
	"testing"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
)

func newPersistMonitor(t *testing.T) (*EpisodeMonitor, database.SubtitleStore) {
	t.Helper()
	store, err := database.OpenPebble(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return &EpisodeMonitor{store: store, logger: logging.GetLogger("test")}, store
}

// TestBlacklistPersistenceRoundTrip verifies entries persist, are honored by
// IsBlacklisted with language scoping, and are removed on request.
func TestBlacklistPersistenceRoundTrip(t *testing.T) {
	m, store := newPersistMonitor(t)
	defer store.Close()

	dur := time.Hour
	if err := m.AddToBlacklist("item1", "/media/a.mkv", "en", ReasonManualBlacklist, "manual", &dur); err != nil {
		t.Fatalf("add: %v", err)
	}

	if !m.IsBlacklisted("item1", "en") {
		t.Fatal("expected item1/en blacklisted")
	}
	if m.IsBlacklisted("item2", "en") {
		t.Fatal("item2 should not be blacklisted")
	}
	if m.IsBlacklisted("item1", "fr") {
		t.Fatal("language-scoped entry should not match a different language")
	}

	if err := m.RemoveFromBlacklist("item1"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if m.IsBlacklisted("item1", "en") {
		t.Fatal("expected item1 removed from blacklist")
	}
	entries, _ := store.(database.BlacklistStore).ListBlacklist()
	if len(entries) != 0 {
		t.Fatalf("expected no persisted entries after removal, got %d", len(entries))
	}
}

// TestBlacklistExpiry verifies expired entries are ignored and cleaned up.
func TestBlacklistExpiry(t *testing.T) {
	m, store := newPersistMonitor(t)
	defer store.Close()

	past := -time.Hour // expiry already in the past
	if err := m.AddToBlacklist("item1", "/media/a.mkv", "", ReasonMaxRetriesExceeded, "retries", &past); err != nil {
		t.Fatalf("add: %v", err)
	}

	if m.IsBlacklisted("item1", "en") {
		t.Fatal("expired entry should not count as blacklisted")
	}
	if err := m.CleanupExpiredBlacklist(); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	entries, _ := store.(database.BlacklistStore).ListBlacklist()
	if len(entries) != 0 {
		t.Fatalf("expected expired entries cleaned, got %d", len(entries))
	}
}
