### Added

#### Library → subtitle-search bridge (`library-search`)

Added `scanner.ProcessLibrary` and the `library-search [lang]` command, which
download subtitles for every item in the persisted media library — the rows
populated by `scanlib` and by the Sonarr/Radarr sync. Previously nothing read
the library to fetch subtitles (search only walked filesystem directories), so a
Sonarr/Radarr pull never actually triggered a subtitle search. The sweep reuses
`ProcessFile` per item (same validation, upgrade, history, and events as the
directory scanner), is best-effort (one item's failure doesn't abort the run),
and skips items whose file is missing on disk. See
`docs/WHISPER_PIPELINE_DECISIONS.md` (W1) for the decisions behind this.
