<!-- file: changelog.d/feat-wanted-list.md -->
<!-- version: 1.0.0 -->
<!-- guid: c1f6b830-9d47-4e52-8a06-3b52e7d914af -->
<!-- last-edited: 2026-07-25 -->

### Added

#### Wanted list

`/api/wanted` now exists. The web UI has always called it, but no handler was
ever registered, so the Wanted page silently rendered empty — the request fell
through to the single-page-app catch-all and came back as `index.html` with a
`200`, which the frontend accepted before failing to parse it.

The endpoint is backed by the existing `monitored_items` table, which already
carried status, retry count and blacklist integration, so no schema change was
needed.

    GET    /api/wanted                     list monitored media
    POST   /api/wanted {path, languages}   start monitoring a file
    DELETE /api/wanted {path}              stop monitoring a file

The Wanted page now lists **media files missing subtitles**, which is what
Bazarr means by "wanted", showing each file with its requested languages,
status and retry count. It previously kept a list of subtitle *download URLs*
bookmarked from search results — a different concept that Bazarr has no
equivalent of, and that no backend ever implemented. "Add to Wanted" on a
search result is replaced by "Monitor this file", which acts on the media path.

#### Automatic subtitle monitoring in the web server, off by default

The monitoring loop that acts on the wanted list previously existed only in the
`monitor` CLI command, so `subtitle-manager web` never searched for missing
subtitles on its own. It can now run inside the web server.

**It is disabled by default and must be turned on explicitly** with
`monitor.enabled: true`. This differs from Bazarr, which monitors by default,
and the reason is deliberate: the loop runs the full download pipeline, which
contacts subtitle providers and **writes subtitle files into the operator's
media directories**, unattended, on a timer. Verified in testing — with
monitoring enabled the server wrote a `.srt` next to a media file without
further input. Bazarr asks about this during setup; there is no equivalent
prompt here yet, so enabling it by default would mean an upgrade silently
starts modifying a media library.

Other keys: `monitor.interval` (minutes, default 60), `monitor.languages`
(default `["en"]`), `monitor.max_retries` (default 3),
`monitor.quality_check`.

When Sonarr or Radarr is configured, the wanted list is populated from them on
startup. It is deliberately **not** derived from the local media library: that
would enrol every scanned file at once and turn a single opt-in into a
library-wide download. Individual files can always be added through
`POST /api/wanted`.
