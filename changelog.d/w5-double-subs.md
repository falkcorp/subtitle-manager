### Added

#### Double-subs (bilingual) generation (`dualsub`)

Added `subtitles.GenerateDualSubtitles` and the `dualsub [input] [target-lang]`
command, which produce a bilingual subtitle: each cue keeps its original line(s)
and gains the translation stacked beneath (default target Mandarin `zh`). The
output is written with a sentinel language suffix (default Esperanto `eo`, e.g.
`video.eo.srt`) so media players treat it as a distinct, unused-language track
rather than overwriting a real subtitle. Output is stacked-SRT; styled ASS
positioning is a documented future option. See
`docs/WHISPER_PIPELINE_DECISIONS.md` (W5).
