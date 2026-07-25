<!-- file: changelog.d/fix-mount-profile-routes.md -->
<!-- version: 1.0.0 -->
<!-- guid: 9c4e1a76-2b8d-4f30-85a1-6e0d7c93b482 -->
<!-- last-edited: 2026-07-25 -->

### Fixed

#### Language profile endpoints are reachable

`profilesHandler` and `mediaProfilesHandler` were implemented and unit-tested
but never registered with the router, so the Settings → Languages page and
every media profile assignment silently did nothing.

The failure was invisible because the router serves the single-page app from a
catch-all at `/`. An unregistered `/api/...` path did not 404 — it returned
`200` with `index.html`, so the frontend's `if (res.ok)` check passed, the
following `res.json()` threw into a `catch` that only logged to the browser
console, and the page rendered empty with nothing in the server log.

Now mounted: `/api/profiles`, `/api/profiles/{id}`, `/api/media/profile/{id}`,
and `/api/language-profiles[/{id}]`. The frontend calls the
`language-profiles` spelling while the handlers parse `/api/profiles`, so the
former is aliased onto the latter rather than duplicated.

The profile handlers also selected their storage backend by calling
`OpenSQLStore` directly, ignoring the configured backend. They now use
`OpenStoreWithConfig`.

**Known limitation:** these endpoints work on the SQLite backend but return
`500` on Pebble. They open a store per request, and Pebble takes an exclusive
file lock already held by the server process for its lifetime. This is a
pre-existing bug shared with `librarypaths.go`, `pipeline.go` and `scan.go`;
fixing it needs a process-wide shared store and is deliberately left to a
follow-up rather than mixed into this wiring fix.

#### Regression guard for unmounted API routes

`TestFrontendAPIPathsAreMounted` asserts that paths the web UI calls do not
match the SPA catch-all. It checks the matched `http.ServeMux` pattern rather
than the response, because what the catch-all returns depends on the build —
`200` with `index.html` when `webui/dist` is populated, `404` when it is empty
in tests — so asserting on the response passes in CI while the route is broken
in production.

It also runs without the `sqlite` build tag, unlike the rest of this package's
tests, which call `skipIfNoSQLite` and therefore skip entirely in CI.
