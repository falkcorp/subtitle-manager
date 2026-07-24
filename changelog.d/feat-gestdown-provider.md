<!-- file: changelog.d/feat-gestdown-provider.md -->
<!-- version: 1.0.0 -->
<!-- guid: 3f1c6a48-2b90-4d7e-8c15-a4e29d0b7f62 -->
<!-- last-edited: 2026-07-23 -->

### Added

#### Real keyless Gestdown provider (Addic7ed TV subtitles)

The `gestdown` provider now talks to the real gestdown.info REST API — a free,
keyless proxy over Addic7ed's TV-subtitle catalogue — instead of the previous
stub that hit a fictional `api.gestdown.com/subtitles/{file}/{lang}` endpoint.
Because Addic7ed indexes by show/season/episode rather than by file hash, the
provider parses the media file name into `(series, season, episode)` via
`pkg/metadata`, resolves the show through `/shows/search/{name}` (preferring a
candidate whose season list covers the target season), lists completed
subtitles for the requested language via `/subtitles/get/{showId}/{season}/{episode}/{lang}`,
and downloads the first match. It is TV-only: movie files are rejected, matching
Gestdown/Addic7ed's scope. No account or API key is required.
