// file: pkg/database/puregosqlite_test.go
// version: 1.1.0
// guid: 2f8c41ab-9d63-4e07-85b1-6c0fa3d29e74
// last-edited: 2026-08-12

package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
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

// TestPureGoSQLiteBusyTimeoutIsSet guards against SQLITE_BUSY under concurrent
// access. modernc.org/sqlite defaults busy_timeout to 0, meaning a connection
// that finds the write lock held fails immediately instead of waiting. That is
// not theoretical: `web --db-backend sqlite` logged
//
//	selftest database ping failed: database is locked (5) (SQLITE_BUSY)
//
// on a loaded machine, because database/sql keeps a connection *pool* and the
// selftest pinged on a second connection while schema seeding still held the
// write lock on the first.
//
// This asserts the effective pragma rather than the DSN string. A DSN the
// driver does not understand is silently ignored, so checking that we passed
// the option would pass while the setting had no effect.
func TestPureGoSQLiteBusyTimeoutIsSet(t *testing.T) {
	for _, path := range []string{filepath.Join(t.TempDir(), "auth.db"), ":memory:"} {
		t.Run(path, func(t *testing.T) {
			db, err := Open(path)
			if err != nil {
				t.Fatalf("Open(%q): %v", path, err)
			}
			defer db.Close()

			var timeout int
			if err := db.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout); err != nil {
				t.Fatalf("reading busy_timeout: %v", err)
			}
			if timeout <= 0 {
				t.Errorf("busy_timeout = %d, want > 0; concurrent writers will fail instantly with SQLITE_BUSY", timeout)
			}
		})
	}
}

// TestPureGoSQLiteConcurrentWriters exercises the pool the way the running
// server does: several goroutines writing through one *sql.DB. Without a busy
// timeout this surfaces "database is locked" instead of serialising.
func TestPureGoSQLiteConcurrentWriters(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	const writers = 8
	errs := make(chan error, writers)
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, err := db.Exec(
					`INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, datetime('now'))`,
					fmt.Sprintf("user-%d-%d", n, j), "hash", "user",
				)
				if err != nil {
					errs <- err
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent write failed: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&count); err != nil {
		t.Fatalf("counting users: %v", err)
	}
	if want := writers * 20; count != want {
		t.Errorf("wrote %d rows, want %d", count, want)
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
