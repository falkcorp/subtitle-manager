<!-- file: changelog.d/fix-shared-store.md -->
<!-- version: 1.0.0 -->
<!-- guid: b5d38f21-6c04-4a97-9e15-3f7a0c82d146 -->
<!-- last-edited: 2026-07-25 -->

### Fixed

#### Web server features no longer fail on the Pebble backend

Several parts of the web server opened their own database handle per request.
On SQLite that is harmless. On Pebble, which takes an **exclusive file lock**,
it could not work: the server opens a store at startup for the Sonarr/Radarr
sync tasks and holds it for the process lifetime, so every subsequent open
failed with `resource temporarily unavailable`.

The result was that entire features worked or did not depending on which
storage backend the operator had configured, with no message pointing at the
cause:

- Language profile endpoints returned `500`.
- Adding a library path returned `200` while the scan behind it never ran —
  the failure appeared only as one line in the log.
- The library search sweep and the library scan had the same defect.

`database.GetSharedStore` now returns a process-wide store, opened once and
reused, keyed by backend and path so that reconfiguring yields a store for the
new database rather than silently reusing the old one. The web server's startup
open, the profile handlers, the library scan, the library search sweep and the
library-path scan all use it. Callers must not `Close` it;
`database.CloseSharedStores` exists for shutdown and tests.

`OpenStoreWithConfig` still returns a fresh, caller-owned store and remains
correct for one-shot CLI commands, which own the database for the life of the
process.

Verified end-to-end against a running server on the Pebble backend: language
profile listing and creation went from `500` to working, and adding a library
path went from one logged store-open failure to none.
