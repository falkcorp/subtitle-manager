### Added

#### Single-language subtitle filename option

Set `subtitles.single_language: true` to write subtitles without the language
code in the filename (`movie.srt` instead of `movie.en.srt`), matching Bazarr's
single-language naming option. This applies to the scanner download paths
(`ProcessFile` and the profile-based `ProcessFileWithProfile`). The language is
still validated and recorded in the download history; a single-language layout
holds one subtitle language per media file. Default is off — existing
`<base>.<lang>.srt` naming is unchanged.
