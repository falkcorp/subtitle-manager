<!-- file: changelog.d/fix-route-drift-tail.md -->
<!-- version: 1.0.0 -->
<!-- guid: 8b4e21d7-3c06-4f95-a1e8-6d720f3ba9c4 -->
<!-- last-edited: 2026-07-26 -->

### Fixed

#### Global "Rescan Library" button now does something

`POST /api/library/rescan-all` is mounted. The button in the app bar has always
called it, but no handler was registered, so the request fell through to the
single-page-app catch-all and came back as `index.html` with a `200` — the
frontend logged "Global library rescan started" and nothing was scanned.

It is an alias for the existing `libraryRescanHandler`, which already rescans
every configured library path; `/api/library/resync` is an alias of the same
handler. No new behaviour was needed, only the missing registration.

#### Removing a wanted item with an unclean path

`DELETE /api/wanted` matched the request path literally, while `POST` stores the
`filepath.Clean`-ed path. An equivalent but unclean spelling (`/media/./a.mkv`)
therefore matched no row and still returned `204`, reporting a successful delete
that removed nothing. The path is now cleaned before matching.

Only cleaned, deliberately not run through the full `ValidateAndSanitizePath`
allowlist: items synced from Sonarr or Radarr carry *that* server's paths, which
need not lie under an allowed base directory, and enforcing the allowlist here
would have made those items permanently undeletable. The allowlist exists to
guard filesystem access, and this branch performs none — it is a key match
against rows the store already holds.

### Notes

Two paths the web UI calls still have no handler, and both are recorded in
`route_coverage_test.go` with the reason rather than silently omitted:

- `/api/bulk-operation` — `MediaLibrary.jsx` defines `handleBulkOperation` and
  `toggleFileSelection`, but **nothing references them**: there is no selection
  checkbox and no bulk toolbar. This is dead frontend code rather than a feature
  missing its backend, so writing a handler would mean inventing a request
  contract no UI sends. The selection UI and the endpoint should be built
  together.
- `/api/notifications/test/{type}` — blocked on a config namespace mismatch, not
  on the handler. The notification settings page saves through
  `POST /api/config`, which sets flat keys (`discord_webhook`), while the runtime
  reads namespaced ones (`notifications.discord_webhook`). A test button on top
  of that would report "not configured" immediately after a successful save, or
  report success for a channel that never fires in production. The key bridge
  and the endpoint have to land together.
