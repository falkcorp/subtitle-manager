### Added

#### Subtitle post-processing pipeline (Bazarr parity)

Added `pkg/postprocess`, wired into the download path (`scanner.ProcessFile`), so
downloaded subtitles can be post-processed like Bazarr does — all opt-in via
config: `postprocess.utf8_encoding` (detect charset and convert to UTF-8),
`postprocess.chmod` (set file permissions), `postprocess.auto_sync` (sync the
subtitle to the media after download), and `postprocess.custom_script` (run a
shell command with `SM_SUBTITLE_PATH`/`SM_MEDIA_PATH`/`SM_LANG`). With no config
the pipeline is a no-op. Also added `docs/BAZARR_PARITY_STATUS.md`, a grounded
backend feature-parity inventory and plan.
