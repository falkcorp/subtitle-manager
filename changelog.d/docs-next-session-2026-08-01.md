<!-- file: changelog.d/docs-next-session-2026-08-01.md -->
<!-- version: 1.0.0 -->
<!-- guid: 2d6f83b1-40c5-4a97-b2e8-7130fa5c9d64 -->
<!-- last-edited: 2026-08-01 -->

### Changed

#### Next-session prompt refreshed after the language-profile work

Records what #2230/#2231/#2232 changed, moves mass-edit back to the top now
that profile assignments actually affect downloads, and lists the bugs left
open (default-profile assignment indistinguishable from unassigned, the
undeletable last profile, `cmd/profiles.go` hardcoding pebble, and the Whisper
server speaking `/transcribe` rather than `/asr`). Adds a working note to
confirm which process answered a live verification, after an orphaned binary
holding a port made a working fix look broken.
