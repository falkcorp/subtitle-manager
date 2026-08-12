<!-- file: changelog.d/20260812-pure-go-sqlite.md -->
<!-- version: 1.2.0 -->
<!-- guid: c4e81f70-2a95-4d63-9b18-5e7d0a3f6c21 -->
<!-- last-edited: 2026-08-12 -->

### Fixed

#### Release binaries can now run the web server at all

No published release binary, on any platform, could start the web UI. Running
the `darwin_arm64` artifact from `v1.0.1-rc.53` produced:

    web server requires SQLite for authentication. Please build with: go build -tags sqlite

The web server opens a SQLite database for authentication unconditionally —
even when `db_backend: pebble`, where it keeps a separate `auth.db` alongside
the Pebble store. SQLite sat behind a `sqlite` build tag that required CGO, and
releases are built with `CGO_ENABLED=0`. So the tag was never set, `Open()` was
a stub that returned an error, and the server refused to start.

The Docker image was **not** a complete escape hatch either, contrary to what
was previously believed. Its build branched on `TARGETARCH`: amd64 got
`CGO_ENABLED=1 -tags=sqlite`, while **arm64 got neither**. The arm64 container
had the same failure as the release binaries.

SQLite now comes from the pure-Go `modernc.org/sqlite` driver and is compiled
into every build. `CGO_ENABLED=0 go build .` produces a binary that serves the
web UI on all five release targets (linux/amd64, linux/arm64, darwin/amd64,
darwin/arm64, windows/amd64), all verified to cross-compile.

Verified end to end, not from tests: a binary built the release way
(`CGO_ENABLED=0`, no build tags) starts `web`, creates a real 217 KB `auth.db`
inside a Pebble deployment's database directory, accepts a form-encoded
`POST /api/login`, serves an authenticated `GET /api/config` with the returned
session cookie, and rejects a wrong password with 401.

#### Data race when starting a background task

`POST /api/tasks/start` marshalled the live `*tasks.Task` straight to the
response. `tasks.Start` has already launched the worker goroutine by that point,
and that goroutine mutates `Status`, `Progress`, `Error` and `CompletedAt` under
the task's mutex — while `json.Encoder` read the same fields by reflection with
no lock held.

`Task` already carried the fix: `GetSnapshot()`, whose doc comment says it
exists "for safe copying and serialization". This one call site reached past it.
`/api/tasks` was already correct, because `tasks.List()` returns snapshots.

This race is not new — it reproduces on the previous release under
`-tags sqlite`. It was invisible because `pkg/webserver`'s tests skipped
themselves when SQLite was unavailable, which in the untagged build CI actually
runs was always.

#### `SQLITE_BUSY` on the SQLite backend under load

`web --db-backend sqlite` could fail a query the moment it started:

    selftest database ping failed: database is locked (5) (SQLITE_BUSY)

`modernc.org/sqlite` defaults `busy_timeout` to 0, meaning a connection that
finds the write lock held gives up immediately rather than waiting. `database/sql`
hands out a connection *pool*, so even single-process operation has several
connections contending for SQLite's single write lock — here the selftest ping
landed while schema seeding still held it.

The connection string now sets `busy_timeout` to 5s. `TestPureGoSQLiteConcurrentWriters`
reproduces the failure deterministically without it (8 goroutines, 20 inserts
each), and `TestPureGoSQLiteBusyTimeoutIsSet` asserts the *effective* pragma
rather than the DSN text — a connection string the driver did not understand
would be silently ignored and the DSN-level check would pass anyway.

An empty path is left untouched. It means "private temporary database" to
SQLite, and the driver only splits query parameters off when something precedes
the `?`, so a bare `?_pragma=...` is read as a literal *filename*. Adding the
suffix unconditionally scattered files named `?_pragma=busy_timeout(5000)`
through three package directories during a test run, with the pragma unapplied.
`TestPureGoSQLiteEmptyPathCreatesNoStrayFile` pins that down.

#### The `user` CLI opened the wrong database on Pebble deployments

Every `user` subcommand died on the default backend:

    subtitle-manager user list --db-backend pebble --db-path <dbdir>
    level=fatal msg="unable to open database file (14)"

Authentication always lives in SQLite regardless of `db_backend`, but the join
onto `auth.db` was written out only in `pkg/webserver/server.go`. The CLI opened
viper's raw `db_path`, which on Pebble is a *directory*. So `user add` was
unusable on the backend the product defaults to, while the web server worked
against the very same configuration.

Both callers now go through `database.GetAuthDatabasePath()`, which returns the
main database on the sqlite backend and `<db_path>/auth.db` on pebble and
postgres. Verified as a round trip on a real Pebble deployment: `user add`
creates an account the web UI can log in with, and `user list` shows accounts
created through the web UI — with no `--db-path` workaround.

### Changed

#### The `sqlite` and `nosqlite` build tags are gone

There is one build configuration. Both the SQLite and PebbleDB backends are
compiled into every binary and selected at runtime with `db_backend`. Passing
`-tags sqlite` is now a harmless no-op.

Removed: `pkg/database/{drivers_sqlite,drivers_nosqlite,sqlite_support,sqlite_no_support,sqlite_disabled}.go`,
`pkg/testutil/{drivers_sqlite,drivers_nosqlite}.go`, and the `build-sqlite`,
`run-web-sqlite`, `test-sqlite`, `test-race-sqlite`, `test-coverage-sqlite`,
`test-all-sqlite` and `test-e2e-all-sqlite` Make targets. `sqlite_enabled.go`
became the untagged `pkg/database/sqlite.go`; `initSchema` had to move with it,
or an untagged `Open()` would have returned a healthy handle to a database with
no tables. `HasSQLite()` is retained and now always returns `true`.

`make run-web` now depends on `build` rather than `binary`, so it embeds a
freshly built frontend instead of silently serving a stale one. The Makefile's
default `CGO_ENABLED` is now `0`.

**Five tests started running for the first time** in the default build once
`HasSQLite()` stopped returning false: `TestCrossBackendCompatibility`,
`TestLanguageProfileSQLiteIntegration`, `TestMigrateToPebble`,
`TestMigration_LanguageProfileAssignments_SQLite` and
`TestSubtitleScoringSQLiteIntegration`. `pkg/database` went from 66 passing / 8
skipped to 74 passing / 3 skipped; the 3 remaining skips are PostgreSQL tests
that need a live server.

`pkg/database/puregosqlite_test.go` is a regression guard, deliberately not
gated behind `HasSQLite()`. Every other SQLite test in the package skips itself
when SQLite is unavailable — which is precisely how a green suite coexisted with
a product that could not start. Do not add a skip guard to it.
