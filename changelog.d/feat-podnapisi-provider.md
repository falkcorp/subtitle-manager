<!-- file: changelog.d/feat-podnapisi-provider.md -->
<!-- version: 1.0.0 -->
<!-- guid: f23258f8-e811-4f3e-b361-b0365bb9b8b0 -->
<!-- last-edited: 2026-07-24 -->

### Added

#### Real keyless Podnapisi provider (movies and TV)

The `podnapisi` provider now talks to podnapisi.net's public advanced-search
JSON API instead of the previous stub that hit a fictional
`api.podnapisi.com/subtitles/{file}/{lang}` endpoint. Podnapisi indexes by
title/season/episode (TV) or title/year (movies) rather than by file hash, so
the provider parses the media file name via `pkg/metadata`, queries
`/subtitles/search/advanced` with the matching `movie_type`, selects the first
result whose media type matches, and downloads `/{id}/download?container=zip` —
extracting the subtitle from the returned ZIP archive. Unlike the TV-only
Gestdown provider, Podnapisi covers both movies and TV episodes. No account or
API key is required. The request shape and ISO 639-1 language code were verified
against subliminal's recorded API cassettes.
