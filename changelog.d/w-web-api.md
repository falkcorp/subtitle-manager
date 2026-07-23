### Added

#### REST endpoints for the Whisper pipeline

Added web API endpoints so the new features are usable from the UI:
`/api/library/search` (+ `/api/library/search/status`) runs the library→search
bridge, `/api/dualsub` generates bilingual double-subs from an uploaded subtitle,
and `/api/verify` runs subtitle drift verification for a server-side media +
subtitle path and returns the drift report. See
`docs/WHISPER_PIPELINE_DECISIONS.md` (W-web).
