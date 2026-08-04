### Added

#### Assign a language profile to many files at once

The media library gained a **Mass Edit** mode: toggle it, tick the files you
want, pick a language profile, and apply. This closes the last 🔴 row in the
Bazarr parity matrix for bulk operations.

The new `POST /api/media/profiles/bulk` endpoint validates the profile **once**
before touching anything, then reports a per-item result array so a partial
success can say "7 of 9 assigned" instead of a single pass/fail. Items that
failed stay selected in the UI, so a retry does not re-send the ones that
already landed. An empty `profile_id` clears the assignment, matching the
single-item route.

Two things worth recording for future work in this area. The route is registered
as an **exact** `ServeMux` pattern rather than being string-matched inside the
existing handler: `/api/media/` is registered as a subtree for media tags, so a
non-exact pattern would have been answered by the tag handler with a `200` and
the wrong body rather than a `404`. And `MediaLibrary.jsx` previously carried a
`handleBulkOperation` that posted to `/api/bulk-operation`, an endpoint that was
never mounted; nothing called it, and its errors went only to `console.error`,
so it has been replaced rather than kept alive.
