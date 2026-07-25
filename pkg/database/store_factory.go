// file: pkg/database/store_factory.go
// version: 1.1.0
// guid: e84ac7c7-7e79-4715-b815-2a759bf96277
// last-edited: 2026-07-25

package database

// OpenStore selects a storage backend and returns a SubtitleStore.
// Backend may be "sqlite", "pebble" or "postgres". Any other value defaults to SQLite.
func OpenStore(path, backend string) (SubtitleStore, error) {
	switch backend {
	case "pebble":
		return OpenPebble(path)
	case "postgres":
		return OpenPostgresStore(path)
	default:
		return OpenSQLStore(path)
	}
}

// OpenStoreWithConfig opens a *new* store using the current configuration. The
// caller owns it and must Close it.
//
// Long-lived processes that open a store more than once — the web server, and
// anything running inside it — must use GetSharedStore instead. Pebble takes an
// exclusive file lock, so a second concurrent open fails outright. This
// function is appropriate for one-shot CLI commands, which own the database for
// the duration of the process.
func OpenStoreWithConfig() (SubtitleStore, error) {
	path := GetDatabasePath()
	backend := GetDatabaseBackend()
	return OpenStore(path, backend)
}
