<!-- file: docs/WHISPER_PIPELINE_DECISIONS.md -->
<!-- version: 1.0.0 -->
<!-- guid: 3e1b072f-e045-4b9d-8920-e957f3f06cd2 -->
<!-- last-edited: 2026-07-23 -->

# Whisper Pipeline — Decisions Log

Running record of decisions made while implementing the Whisper-centric pipeline
described in [`WHISPER_PIPELINE_DESIGN.md`](WHISPER_PIPELINE_DESIGN.md). Each
entry is a decision the reviewer may want to **revise later**. Format:
**Decision → Why → How to revise.**

Work items are W1–W6 from the design doc. This log is appended to by each
work-item PR.

---

## W1 — Library → subtitle-search bridge

Implemented as `scanner.ProcessLibrary` (`pkg/scanner/library.go`) + the
`library-search [lang]` CLI command (`cmd/librarysearch.go`).

- **D1.1 Reuse `ProcessFile` per item rather than a new download path.**
  Why: keeps language validation, output-path construction, existing-subtitle
  skipping, upgrade logic, download-history recording, and failure events
  identical to the directory scanner — one code path to maintain.
  Revise: if library items should be searched differently from filesystem files
  (e.g. richer provider queries from stored Title/Season/Episode), change the
  body of `ProcessLibrary` to call a metadata-aware fetch instead of `ProcessFile`.

- **D1.2 Best-effort sweep: a single item's fetch failure is logged, not fatal.**
  Why: a missing subtitle for one title should not abort the whole library run.
  Revise: to make failures fatal, return the error from the pool goroutine in
  `ProcessLibrary` instead of logging and returning nil.

- **D1.3 Skip items whose file is missing on disk (not an error).**
  Why: the persisted library can lag the filesystem (deleted/moved media).
  Revise: to surface stale entries, add a "prune missing" pass or log at a higher
  level / emit an event.

- **D1.4 Provider selection: use all configured providers (`FetchFromAll` via
  `nil` provider), keyed on file path + language only.**
  Why: matches current `scan` behaviour; no new configuration needed.
  Revise: switch to `FetchWithProfile`/`FetchWithProfileTagged` (needs the
  `*sql.DB` handle) to honour per-item language profiles and tags; or thread the
  stored Title/Season/Episode into provider search queries for better matches.

- **D1.5 Auto-trigger after Sonarr/Radarr sync is DEFERRED; the bridge is exposed
  as a CLI command for now.**
  Why: wiring a post-sync search into `pkg/sonarr/scheduler.go` /
  `pkg/radarr/scheduler.go` requires threading language + provider config through
  the schedulers; kept out of the first PR to stay focused. The CLI command is
  cron-able in the meantime.
  Revise: add an opt-in `sonarr.search_after_sync` / `radarr.search_after_sync`
  config and call `ProcessLibrary` at the end of the scheduler's job function.

- **D1.6 Default worker count in the CLI is 4 when `scan_workers` is unset.**
  Why: a sensible parallelism default; `ProcessLibrary` itself clamps `<1` to 1.
  Revise: change the default in `cmd/librarysearch.go` or set `scan_workers`.
