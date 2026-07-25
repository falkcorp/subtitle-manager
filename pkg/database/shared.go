// file: pkg/database/shared.go
// version: 1.0.0
// guid: 3f8a1c05-7d94-4e26-b0af-52c9138e7d64
// last-edited: 2026-07-25

package database

import "sync"

// Shared stores, keyed by "backend|path".
var (
	sharedMu     sync.Mutex
	sharedStores = map[string]SubtitleStore{}
)

// GetSharedStore returns a process-wide SubtitleStore for the configured
// backend, opening it on first use and reusing it thereafter.
//
// Callers MUST NOT call Close on the returned store — it is shared with every
// other caller in the process, and closing it breaks them. Use
// CloseSharedStores at shutdown, or from a test cleanup.
//
// This exists because Pebble takes an *exclusive file lock*. The web server
// opens a store at startup for the Sonarr/Radarr sync tasks and holds it for
// the process lifetime, so any handler that opened its own store per request
// failed with "resource temporarily unavailable" and returned 500 — but only
// on Pebble. On SQLite, which permits multiple handles, the same code worked
// fine, so the bug was invisible in tests and depended on the operator's
// choice of backend.
//
// Sharing one handle is safe: SQLStore, PostgresStore and PebbleStore are thin
// wrappers over *sql.DB / *pebble.DB, both of which are safe for concurrent
// use, and none of them holds mutable state of its own.
//
// The key includes the path and backend so that changing configuration (which
// tests do routinely, via a temp directory per test) yields a distinct store
// rather than silently handing back one pointed at the previous database.
func GetSharedStore() (SubtitleStore, error) {
	path := GetDatabasePath()
	backend := GetDatabaseBackend()
	key := backend + "|" + path

	// The lock is deliberately held across OpenStore rather than released for
	// the open and re-taken: two goroutines racing here would otherwise both
	// open a store, and on Pebble the loser's open fails on the lock — exactly
	// the bug this function exists to prevent.
	sharedMu.Lock()
	defer sharedMu.Unlock()

	if s, ok := sharedStores[key]; ok {
		return s, nil
	}
	s, err := OpenStore(path, backend)
	if err != nil {
		// Not cached: a failed open must not poison later attempts, which may
		// succeed once the path exists or the config is corrected.
		return nil, err
	}
	sharedStores[key] = s
	return s, nil
}

// CloseSharedStores closes every store handed out by GetSharedStore and clears
// the cache. It returns the first error encountered, after attempting to close
// all of them.
//
// Intended for process shutdown and for tests that need to release a database
// file (notably on Windows, and for Pebble's lock) before removing it.
func CloseSharedStores() error {
	sharedMu.Lock()
	defer sharedMu.Unlock()

	var firstErr error
	for key, s := range sharedStores {
		if err := s.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(sharedStores, key)
	}
	return firstErr
}
