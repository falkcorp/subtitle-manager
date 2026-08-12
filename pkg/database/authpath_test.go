// file: pkg/database/authpath_test.go
// version: 1.0.0
// guid: 8d41c3e9-57b2-4f60-a8d3-1c96e2705bf4
// last-edited: 2026-08-12

package database

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// TestGetAuthDatabasePath pins down where authentication lives for each
// backend. The `user` CLI opened viper's raw db_path and so handed SQLite the
// Pebble *directory* on a Pebble deployment, dying with
//
//	unable to open database file (14)
//
// while the web server worked, because the join onto "auth.db" existed only in
// pkg/webserver. Both now go through this helper, so they cannot drift apart.
func TestGetAuthDatabasePath(t *testing.T) {
	for _, tc := range []struct {
		name     string
		backend  string
		dbPath   string
		filename string
		want     string
	}{
		{
			// Pebble keeps a directory; the auth database is a file inside it.
			name:    "pebble puts auth.db inside the data directory",
			backend: "pebble",
			dbPath:  "/var/lib/sm/db",
			want:    filepath.Join("/var/lib/sm/db", "auth.db"),
		},
		{
			// SQLite authenticates against the main database itself.
			name:     "sqlite uses the configured database file",
			backend:  "sqlite",
			dbPath:   "/var/lib/sm",
			filename: "subtitle-manager.db",
			want:     filepath.Join("/var/lib/sm", "subtitle-manager.db"),
		},
		{
			// Postgres has no SQLite main store, so auth still needs its own file.
			name:    "postgres keeps a separate auth.db",
			backend: "postgres",
			dbPath:  "/var/lib/sm/db",
			want:    filepath.Join("/var/lib/sm/db", "auth.db"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("db_backend", tc.backend)
			viper.Set("db_path", tc.dbPath)
			if tc.filename != "" {
				viper.Set("sqlite3_filename", tc.filename)
			}

			if got := GetAuthDatabasePath(); got != tc.want {
				t.Errorf("GetAuthDatabasePath() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestGetAuthDatabasePathMatchesSQLiteBackend guards the one case where the
// auth database and the main database are the same file. If these diverge, a
// SQLite deployment grows a second, empty user table and logins start failing
// against a database that looks fine.
func TestGetAuthDatabasePathMatchesSQLiteBackend(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("db_backend", "sqlite")
	viper.Set("db_path", "/var/lib/sm")
	viper.Set("sqlite3_filename", "subtitle-manager.db")

	if got, want := GetAuthDatabasePath(), GetDatabasePath(); got != want {
		t.Errorf("on the sqlite backend auth must live in the main database: got %q, want %q", got, want)
	}
}
