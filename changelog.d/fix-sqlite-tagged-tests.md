<!-- file: changelog.d/fix-sqlite-tagged-tests.md -->
<!-- version: 1.0.0 -->
<!-- guid: e17b3d95-04ca-4b26-8f73-9d5c8a1e6403 -->
<!-- last-edited: 2026-07-30 -->

### Fixed

#### Tests under `-tags sqlite` — the configuration the web server actually requires

The web server **refuses to start without the `sqlite` build tag**:

    web server requires SQLite for authentication. Please build with: go build -tags sqlite

CI does not build that tag, so the configuration every user runs is the one
nothing tests. Running it surfaced failures in six packages. Four are fixed
here; all were **test** defects rather than product ones, but they are what has
kept this configuration unverifiable.

- **`pkg/database` / `TestSQLiteAvailability`** passed a *directory* to
  `OpenStore(..., "sqlite")`. Only the pebble backend takes a directory; sqlite
  is routed to `OpenSQLStore`, which opens the path as a database file. It
  failed with "unable to open database file: is a directory".
- **`pkg/database` / `TestLanguageProfileSQLiteIntegration`** inserted
  `DefaultLanguageProfile()` verbatim, whose ID is the one the store creates on
  initialisation — so it hit `UNIQUE constraint failed`. It also asserted the
  store contained *exactly one* profile, an assumption that stopped holding
  when the built-in default was introduced.
- **`cmd` / `TestExtractCmdStoresRecord`** passed a relative media path, which
  `security.ValidateAndSanitizePath` rejects. The test predates that hardening.
- **`pkg/radarr` and `pkg/sonarr` / `TestSyncLogsConflicts`** passed in
  isolation and failed in a full package run. See below — this one is worth
  reading.

#### `logging.Hook` could be set and silently have no effect

`GetLogger` memoises one logger per component and calls `AddHook` when it first
builds that logger. A hook assigned to `logging.Hook` afterwards is therefore
attached to nothing: any component whose logger already exists keeps logging
straight past it, with no error anywhere.

That is why the *arr conflict tests passed alone and failed in a package run —
a sibling test caused the same component's logger to be created first, so the
memory hook installed by the test captured nothing and the assertion could
never succeed.

New `logging.SetHook` sets the hook and discards the logger cache, so existing
components pick it up. Assigning `logging.Hook` directly still works for
process startup, where nothing has logged yet.

### Still failing under `-tags sqlite`

Not fixed here, and called out rather than left to be rediscovered:

- `pkg/metadata` — `TestScanLibraryProgress`
- `pkg/webserver` — `TestSPAIndexFallback`, `TestScanHandlers`, `TestTranslate`,
  `TestTranslateUpload`, `TestSystemHandlers`

Adding `-tags sqlite` to CI should wait until those are green, or it lands
permanently red. The default (untagged) build is unaffected by this change.
