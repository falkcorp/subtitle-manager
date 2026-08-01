<!-- file: changelog.d/fix-media-profile-path-key.md -->
<!-- version: 1.0.0 -->
<!-- guid: 5c8e1f42-9a37-4b6d-8e21-3f0d7a1c9b45 -->
<!-- last-edited: 2026-08-01 -->

### Fixed

#### Per-item language profile assignments no longer collide

Assigning a language profile to one media file reassigned it for every other
file in the library.

The web UI identifies a media item by its file path, percent-encoded into the
URL: `PUT /api/media/profile/%2Fmedia%2FShow%2FS01E01.mkv`. `net/http` decodes
the request target before a handler runs, so by then those `%2F` separators are
indistinguishable from real ones, and the handler took only the *first* path
segment as the identifier. Every file under `/media` therefore stored its
assignment under the key `media`, and each new assignment overwrote the last.

The handler now takes the whole remainder of the path as the identifier, via a
new `pathRest` helper alongside the existing `pathSegment`. The identifier is
also `filepath.Clean`ed before use: the CLI looks up the cleaned absolute path
that `security.ValidateAndSanitizePath` returns, so without normalising here an
assignment made in the UI would be invisible to
`subtitle-manager profiles show <path>` and vice versa.

Nothing caught this because every test used an identifier like `42` or
`some-id`, which has no slash to split on. Two regression tests now assign
profiles to two slash-bearing paths and assert they resolve independently, and
assert that a key written through the handler is readable at the cleaned path
the CLI uses.

Any existing `media_profiles` row is keyed on a truncated segment and becomes
unreachable after this change. Those rows recorded a collision rather than a
usable assignment, so they are left in place rather than migrated; re-assign
affected items in the UI.
