// file: pkg/database/puregosqlite_test.go
// version: 1.0.0
// guid: 2f8c41ab-9d63-4e07-85b1-6c0fa3d29e74
// last-edited: 2026-08-12

package database

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPureGoSQLiteOpensWithoutCGO reproduces the production failure that no unit
// test in this repo could previously see: a binary built the way releases are
// built (CGO_ENABLED=0, no build tags) cannot open the SQLite auth database, so
// `subtitle-manager web` refuses to start with
//
//	web server requires SQLite for authentication
//
// Deliberately NOT gated behind HasSQLite(). Every other SQLite test in this
// package skips itself when SQLite is unavailable, which is exactly why the
// suite stayed green while no published binary could serve the web UI. This
// test must run, and pass, in the default build.
func TestPureGoSQLiteOpensWithoutCGO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%q) failed; a release binary cannot start the web server: %v", path, err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// The file must actually exist on disk. A driver that silently opens an
	// in-memory database would pass a connection check and lose every user.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected a database file at %q: %v", path, err)
	}
}

// TestPureGoSQLiteAppliesSchema guards the trap that a driver swap alone would
// leave behind: initSchema currently lives in the `//go:build sqlite` file, so
// an untagged Open() can return a perfectly healthy handle to a database with
// zero tables. Connection success is not schema success.
//
// The users table is what authentication needs, and the seeded rows prove
// initSchema ran to completion rather than erroring partway through.
func TestPureGoSQLiteAppliesSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	for _, table := range []string{"users", "sessions", "api_keys", "permissions", "language_profiles"} {
		var name string
		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing from schema: %v", table, err)
		}
	}

	// initSchema seeds these; a zero count means it bailed out silently.
	var perms int
	if err := db.QueryRow(`SELECT COUNT(1) FROM permissions`).Scan(&perms); err != nil {
		t.Fatalf("counting permissions: %v", err)
	}
	if perms == 0 {
		t.Error("permissions table is empty; initSchema did not seed default roles")
	}

	var profiles int
	if err := db.QueryRow(`SELECT COUNT(1) FROM language_profiles`).Scan(&profiles); err != nil {
		t.Fatalf("counting language_profiles: %v", err)
	}
	if profiles == 0 {
		t.Error("language_profiles is empty; the default profile was not seeded")
	}
}

// TestPureGoSQLiteReopenIsIdempotent covers migrations under the new driver.
// initSchema runs on every Open, and addColumnIfNotExists detects an existing
// column by matching the string "duplicate column name" in the driver's error.
// That message is driver-specific, so a driver swap can turn a no-op migration
// into a hard failure on the second start — a bug that only appears on restart,
// never on a fresh install.
func TestPureGoSQLiteReopenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.db")

	first, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if _, err := first.Exec(
		`INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, datetime('now'))`,
		"someone", "hash", "admin",
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	first.Close()

	second, err := Open(path)
	if err != nil {
		t.Fatalf("reopening an existing database failed; upgrades would break on restart: %v", err)
	}
	defer second.Close()

	var username string
	if err := second.QueryRow(`SELECT username FROM users WHERE username = ?`, "someone").Scan(&username); err != nil {
		t.Fatalf("data did not survive reopen: %v", err)
	}
	if username != "someone" {
		t.Errorf("username = %q, want %q", username, "someone")
	}
}
