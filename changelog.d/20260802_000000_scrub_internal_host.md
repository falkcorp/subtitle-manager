<!-- file: changelog.d/20260802_000000_scrub_internal_host.md -->
<!-- version: 1.0.0 -->
<!-- guid: 7c1e4a90-2b83-4f56-9d17-8e0a35b6f2c4 -->
<!-- last-edited: 2026-08-02 -->

### Fixed

- Removed a private network address and internal hostname from
  `docs/NEXT_SESSION_PROMPT.md`. This repository is public and the file had no
  reason to record where a self-hosted service lives. The Whisper section is
  also rewritten: the transcription fallback now works, so the instructions to
  start a container and repoint the URL were both stale and impossible on that
  host.
