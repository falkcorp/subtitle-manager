### Fixed

#### Media Library rendered completely empty, and Mass Edit assigned profiles to subtitles

Four defects in `webui/src/MediaLibrary.jsx`, all confirmed in a real browser
and each now covered by a regression test that was watched failing first.

**The page was blank.** `GET /api/library/browse` marshals `isDirectory` (see
`MediaItem` in `pkg/webserver/server.go`), but the component read `item.is_dir`
in nine places. That is always `undefined`, so every directory was filtered out
of the listing. The root of a media tree is all directories, so the page
rendered with no items and no empty state, and Mass Edit was unreachable behind
it. The flag is now normalised once at the fetch boundary.

**Breadcrumbs threw.** Every breadcrumb click called `loadDirectory()`, which is
defined nowhere in the file, raising a `ReferenceError`. The correct
`navigateToPath()` helper already existed and was going unused — ESLint had been
reporting it as an unused variable. The breadcrumbs now call it.

**Opening a directory refetched the previous one.** The item click ran
`setCurrentPath(path)` and then `loadCurrentDirectory()` back to back; the
second call closed over the `currentPath` from the render it was created in, so
it requested the directory being left. Two responses were then in flight and
whichever landed last won. The click now only sets the path and lets the effect
reload.

**"Select all files" swept in sidecar subtitles.** It filtered on
`!item.is_dir && item.path` while the listing filtered on video extensions, so
the selection included the `.srt` files the listing had just hidden — two
visible episodes reported "4 assigned" and attached a language profile to two
subtitle files. Both now share one `isMediaFile()` predicate.

#### Test fixtures now come from observed server payloads

`MediaLibraryBulk.test.jsx` fed the component `is_dir`, a key the server has
never sent. The test and the component therefore agreed with each other while
both disagreed with production, which is why a fully passing suite missed all
four defects above. The fixture now uses the real shape.

The new `MediaLibraryBrowseShape.test.jsx` asserts observable effects — what is
on screen, and which requests were issued — rather than watching
`console.error`. React reports a throw inside an event handler as an unhandled
error that Vitest does not count as a failure, so an error-log assertion passes
with the bug reinstated.
