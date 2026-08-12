// file: pkg/database/shared_test.go
// version: 1.1.0
// guid: 5e71b4c8-9a02-4d3f-81b6-c47f0e29a1d3
// last-edited: 2026-08-12

package database

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

// usePebbleAt points the database configuration at a throwaway Pebble
// directory and releases every shared store afterwards.
//
// Pebble is used rather than SQLite because it is the backend that exhibits the
// bug: it takes an exclusive file lock, so a second concurrent open fails. It
// also needs no cgo and no build tag, so these tests run in CI — unlike the
// webserver suite.
func usePebbleAt(t *testing.T, dir string) {
	t.Helper()
	prevBackend := viper.GetString("db_backend")
	prevPath := viper.GetString("db_path")
	viper.Set("db_backend", "pebble")
	viper.Set("db_path", dir)
	t.Cleanup(func() {
		if err := CloseSharedStores(); err != nil {
			t.Errorf("CloseSharedStores: %v", err)
		}
		viper.Set("db_backend", prevBackend)
		viper.Set("db_path", prevPath)
	})
}

// TestGetSharedStoreSurvivesSecondOpen is the regression test for the bug this
// package-level cache exists to fix.
//
// The web server opened a store at startup and held it for the process
// lifetime, then each request handler opened another. On Pebble the second open
// failed with "resource temporarily unavailable" and the handler returned 500 —
// while on SQLite, which permits multiple handles, the identical code worked.
// Whole pages were broken or not depending on the operator's backend choice.
func TestGetSharedStoreSurvivesSecondOpen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pebble")
	usePebbleAt(t, dir)

	// Stands in for the web server's startup open, held for the process
	// lifetime.
	first, err := GetSharedStore()
	if err != nil {
		t.Fatalf("first GetSharedStore: %v", err)
	}

	// Stands in for a request handler needing a store while the above is held.
	second, err := GetSharedStore()
	if err != nil {
		t.Fatalf("second GetSharedStore while the first is held: %v "+
			"(this is the 500-on-Pebble bug)", err)
	}
	if first != second {
		t.Error("GetSharedStore returned a distinct store for the same config; " +
			"a second Pebble handle cannot be opened while the first is held")
	}

	// Demonstrates that the old approach genuinely fails here, so the test
	// above is not passing vacuously.
	if extra, err := OpenStoreWithConfig(); err == nil {
		extra.Close()
		t.Error("OpenStoreWithConfig unexpectedly succeeded while a shared " +
			"Pebble store was open; if Pebble no longer takes an exclusive " +
			"lock, GetSharedStore may no longer be necessary")
	}
}

// TestGetSharedStoreSeparatesConfigs verifies the cache is keyed by backend and
// path, so reconfiguring yields a store for the new database rather than
// silently reusing the previous one.
func TestGetSharedStoreSeparatesConfigs(t *testing.T) {
	root := t.TempDir()

	usePebbleAt(t, filepath.Join(root, "one"))
	first, err := GetSharedStore()
	if err != nil {
		t.Fatalf("open first: %v", err)
	}

	viper.Set("db_path", filepath.Join(root, "two"))
	second, err := GetSharedStore()
	if err != nil {
		t.Fatalf("open second: %v", err)
	}

	if first == second {
		t.Error("changing db_path returned the same store; callers would " +
			"silently read and write the previous database")
	}
}

// TestGetSharedStoreConcurrent verifies concurrent callers all receive the same
// store and none observes a lock failure. Run under -race.
//
// The lock in GetSharedStore is held across OpenStore precisely for this case:
// releasing it around the open would let two goroutines both attempt one, and
// on Pebble the loser fails on the file lock.
func TestGetSharedStoreConcurrent(t *testing.T) {
	usePebbleAt(t, filepath.Join(t.TempDir(), "pebble"))

	const n = 20
	var wg sync.WaitGroup
	stores := make([]SubtitleStore, n)
	errs := make([]error, n)

	wg.Add(n)
	for i := range n {
		go func() {
			defer wg.Done()
			stores[i], errs[i] = GetSharedStore()
		}()
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Fatalf("goroutine %d: %v", i, errs[i])
		}
		if stores[i] != stores[0] {
			t.Fatalf("goroutine %d got a different store instance", i)
		}
	}
}

// TestCloseSharedStoresAllowsReopen verifies the cache is cleared, so a
// subsequent GetSharedStore opens a fresh handle rather than returning a closed
// one.
func TestCloseSharedStoresAllowsReopen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pebble")
	usePebbleAt(t, dir)

	first, err := GetSharedStore()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := CloseSharedStores(); err != nil {
		t.Fatalf("CloseSharedStores: %v", err)
	}

	second, err := GetSharedStore()
	if err != nil {
		t.Fatalf("reopen after close: %v", err)
	}
	if first == second {
		t.Error("GetSharedStore returned the closed store after " +
			"CloseSharedStores; every subsequent query would fail")
	}
}
