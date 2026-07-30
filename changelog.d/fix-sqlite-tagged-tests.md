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

- **`pkg/metadata` / `TestScanLibraryProgress`** used the fixture
  `movie-GRP.mkv`, which `ParseFileName` rejects outright as an unrecognised
  file name — so the scan stored an item with no release group and the
  assertion could never pass. A realistic release name
  (`Movie.2020.1080p.BluRay.x264-GRP.mkv`) parses correctly to
  `title="Movie" group="GRP"`. The parser was right; the fixture was not.

### Some Go tests silently require `npm run build`

`TestSPAIndexFallback` fails with a 404 when `webui/dist` has not been built:
the single-page-app fallback has no `index.html` to serve. It is not a code
defect and it is not fixed here, but it means a green Go suite depends on a
frontend build having happened first — worth knowing before wiring the sqlite
tag into CI.

### Three product bugs the untested build configuration was hiding

Chasing the last `pkg/webserver` failures turned up real defects, not stale
tests:

- **Library scans were cancelled before doing any work.** `scanHandler` started
  its background task with `r.Context()`, which `net/http` cancels as soon as
  the handler returns — and it returns `202 Accepted` immediately. Every file
  logged "context canceled" and the scan reported zero completed. The same
  pattern was in `startTaskHandler`. Both now use `context.WithoutCancel`,
  which keeps request-scoped values while detaching from the response
  lifecycle.

- **`POST /api/translate` panicked on a nil metrics collector.**
  `metrics.Initialize()` was called only from `StartServer`, so the running web
  command was fine, but any other way of building the handler left the
  Prometheus collectors nil, and the translate handler calls `WithLabelValues`
  on one unconditionally. A nil `*CounterVec` panics, killing the connection
  mid-request: the client sees a bare `EOF`, no status, nothing logged.
  Initialisation moved to `newMux`, which every path goes through; it is
  idempotent, so `StartServer`'s call is harmless.

- **A failed translation was reported as a bare 500 with the error discarded.**
  No way to tell a bad API key from an unreachable service from a malformed
  subtitle. It is logged now — which is how the mock mismatch below was found.

Two test defects alongside them: both Google Translate mocks returned a single
fixed translation regardless of input, so any subtitle with more than one cue
failed the translator's count check ("expected 2 translations, got 1"). They
now return one translation per input string, as the real API does.

#### `/api/announcements` returned 404 on every deployment

The handler answered `404` when `announcements.json` was absent — which is
always: none is committed, and the path is resolved relative to the **process
working directory**, so even adding one would only work if the binary happened
to run two levels below it. An absent file now means "no announcements" and
returns an empty list, and the location is configurable via
`announcements_file`. Making that path sane by default is still worth doing;
this stops a working install looking like a broken endpoint in the meantime.

### The `-tags sqlite` suite is now green

Every package passes under the tag, so it is safe to add to CI — which is the
point, since it is the configuration the web server requires and the only one
users actually run. The default (untagged) build is unaffected.
