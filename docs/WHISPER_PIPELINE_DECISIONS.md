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

---

## W5 — Double-subs (bilingual) generator

Implemented as `subtitles.GenerateDualSubtitles` (`pkg/subtitles/dualsub.go`) +
the `dualsub [input] [target-lang]` CLI command (`cmd/dualsub.go`).

- **D5.1 Append the translation as a stacked line; keep the original.**
  Why: the whole point of double-subs is showing both languages. This is the
  opposite of `TranslateFileToSRT`, which replaces the cue text.
  Revise: to change ordering (translation on top), swap the append for a prepend.

- **D5.2 Output is stacked-SRT, not styled ASS.**
  Why: SRT works everywhere and needs no styling engine; the multi-format ASS
  writer (`pkg/media/server.go`) is currently stubbed. Original renders above,
  translation below, within one cue.
  Revise: implement an ASS output path (top/bottom `\an8`/`\an2` styles) once the
  format writer is finished, and add a `--format ass` flag.

- **D5.3 Translate the whole cue as one unit (all lines joined), not just the
  first line.**
  Why: the single-language translator only translates `Lines[0].Items[0]`, which
  drops multi-line dialogue; `itemText` joins all lines for an accurate
  translation.
  Revise: if per-line alignment matters, translate line-by-line instead.

- **D5.4 Default target language is `zh` (Mandarin); default sentinel tag is `eo`
  (Esperanto).**
  Why: matches the stated intent — Mandarin double-subs tagged with an
  unused-language code so players don't confuse them with real tracks.
  Revise: `--sentinel` and the `target-lang` arg both override; change the
  defaults in `cmd/dualsub.go`.

- **D5.5 Output is a sidecar file (`<input>.eo.srt`), not a muxed track.**
  Why: the project only writes sidecar subtitles; there is no container-muxing
  step. The "player won't confuse it" benefit comes from the filename suffix.
  Revise: add an ffmpeg mux-back step (`-metadata:s:s language=epo`) if an
  embedded track is wanted.

- **D5.6 Translation backend is whatever `translate_service` selects
  (google/gpt/grpc); no double-subs-specific backend.**
  Why: reuse the existing, configured translator.
  Revise: pin a backend for double-subs if quality/consistency demands it.

---

## W2 — Self-hosted Whisper transcription client

Implemented `ASRTranscribe` (`pkg/transcriber/asr.go`) and rewired the container
transcription path (`pkg/transcriber/whisper_container.go`).

- **D2.1 Speak the ASR webservice's native `/asr` protocol, not the OpenAI API.**
  Why: the self-hosted image (`onerahmet/openai-whisper-asr-webservice`) exposes
  `POST /asr` with a multipart `audio_file`, not `/audio/transcriptions`. The old
  stub pointed the OpenAI SDK at the container with a `"dummy-key-for-container"`
  and discarded the result. `ASRTranscribe` posts the file and returns SRT.
  Revise: to support a different self-hosted server, add another client function
  and select on config.

- **D2.2 The container path now WRITES the subtitle (`<media>.<lang>.srt`).**
  Why: both the container and external-API paths previously transcribed and threw
  the bytes away, so no subtitle was ever produced. `writeTranscript` writes a
  sidecar using the same validated path builder as the scanner.
  Revise: to return bytes instead of writing, or to choose a different output
  location/format, change `writeTranscript` / the task functions.

- **D2.3 `ENABLE_WHISPER` auto-orchestration stays opt-in (commented in compose).**
  Why: auto-starting a Whisper container needs the docker socket and (ideally) a
  GPU; enabling it by default would surprise users. The mechanism works when set.
  Revise: uncomment `ENABLE_WHISPER=1` in `docker-compose.yml`/`docker-stack.yml`
  to auto-start the container.

- **D2.4 Kept the OpenAI-compatible external path (`WhisperTranscribe`) as the
  user-provided-server option; the two container designs are NOT yet unified.**
  Why: W2 fixed the primary (ASR webservice) path; the second design
  (`pkg/transcriber/docker.go`, `openai/whisper` CLI) is untouched to keep the
  change focused.
  Revise: remove or gate `docker.go` once the ASR path is confirmed in
  production, per the design doc's "reconcile the two container designs".

- **D2.5 Default ASR HTTP timeout is 30 minutes.**
  Why: transcribing a full movie can take a long time on CPU.
  Revise: pass a custom `*http.Client` to `ASRTranscribe`, or make it
  configurable.

---

## W3 + W4 — Windowed extraction & subtitle drift verification

Implemented as the `pkg/subsync` package (`analyze.go`, `verify.go`,
`measurer.go`) + the `verify [media] [subtitle] [lang]` CLI command
(`cmd/verify.go`). W3 (wiring the orphaned `audio.ExtractTrackWithDuration`) is
folded in — the measurer is its first real caller.

- **D4.1 Drift model is a linear fit `offset = slope·t + intercept`.**
  Why: captures both a constant offset (intercept) and speed-up/slow-down
  (slope) — a plain median shift, like `pkg/syncer`, cannot see drift.
  Revise: for non-linear drift (variable framerate, edits) switch to piecewise
  fitting or per-segment offsets.

- **D4.2 The classifier (`Analyze`) is a pure function; audio/Whisper are
  injected (`AnchorMeasurer`).**
  Why: keeps the math fully unit-testable without ffmpeg or Whisper, and lets the
  measurer target a user-provided or self-hosted server.
  Revise: n/a — this is a testability boundary; extend via new measurers.

- **D4.3 Anchor↔subtitle matching is by nearest start-time within
  `maxMatchMs` (3 s), offset = median of matched diffs.**
  Why: simple and robust for small/moderate offsets; median rejects outliers.
  KNOWN LIMITATION: an offset larger than ~half the inter-cue gap can match the
  wrong neighbouring cue; large drift is still caught at earlier anchors where the
  accumulated offset is smaller.
  Revise: match by TEXT similarity (compare transcribed text to subtitle text)
  instead of time — far more robust but needs fuzzy text matching.

- **D4.4 Rate-drift threshold is 5 ms/s; in-sync tolerance is 150 ms;
  min-confidence 0.5; 10 anchors × 45 s windows, skipping 2 min intro/credits.**
  Why: defaults that flag real framerate drift (≈40 ms/s for 25↔23.976) while
  ignoring jitter, and avoid non-dialogue intros/credits.
  Revise: all are `VerifyOptions` fields / CLI flags.

- **D4.5 `verify` reports and (by exit code) fails when out of sync, but does NOT
  auto-correct.**
  Why: verification and correction are separate concerns; correcting via
  rescale-then-shift (`t' = (1+slope)·t + intercept`) is a follow-up that would
  replace `pkg/syncer`'s single-median shift.
  Revise: add a `--fix` flag that applies the fitted transform to the subtitle.

- **D4.6 Transcription for measurement reuses `WhisperTranscribe`
  (OpenAI-compatible); a self-hosted server is used by pointing its base URL, or
  by injecting a `TranscribeFunc` backed by `ASRTranscribe` (W2).**
  Why: one transcription path; the measurer is backend-agnostic via injection.
  Revise: default the measurer to the container ASR client when configured.

---

## Whisper server configuration (deployment note — no secrets in repo)

The address of the internal Whisper server is intentionally **not** committed. It
lives only in the uncommitted local config (`~/.subtitle-manager.yaml`, outside
the repo) under the `whisper:` block, e.g. `whisper.transcribe_url`.

- **D-cfg.1 Server address stays in `~/.subtitle-manager.yaml`, never in the repo.**
  Why: it is environment-specific and internal; committing an internal IP is
  leakage. The app already reads `$HOME/.subtitle-manager.yaml` by default.
  Revise: use `--config` to point elsewhere, or an env var, per deployment.

- **D-cfg.2 The discovered internal Whisper server is a custom FastAPI service
  (`POST /transcribe` multipart `file` → `{"text","error"}`, plus
  `/transcribe-batch`, `/health`) running `large-v2` — it returns PLAIN TEXT with
  no timestamps.**
  Why (implication): plain text is fine for full-text extraction and for the
  translation source of double-subs (W5), but the TIMED pipeline — drift
  verification (W4) and offset sync — needs cue timings, which this endpoint does
  not provide.
  Revise / how to make the timed features work: point transcription at an
  SRT-returning Whisper instead — either the OpenAI-compatible path
  (`openai_api_url` + `WhisperTranscribe`) or a self-hosted
  `onerahmet/openai-whisper-asr-webservice` via `ASRTranscribe` (W2). The config's
  `whisper.image`/`whisper.port` already reference that ASR webservice for the
  "start our own" path.

- **D-cfg.3 Of the two candidate hosts checked at discovery, one was unreachable
  (powered off) and the other was live; the live address is recorded only in the
  local config.**
  Revise: if the server moves, update `whisper.transcribe_url` in the local config.

---

## W-web (backend) — REST endpoints for the pipeline

Added `pkg/webserver/pipeline.go`: `/api/library/search` (+ `/status`),
`/api/dualsub`, `/api/verify`.

- **D-web.1 `verify` takes server-side paths (JSON), not uploads.**
  Why: uploading a full movie over HTTP is impractical; the library already holds
  local paths. Paths are validated with `ValidateAndSanitizePath`.
  Revise: add an upload variant for ad-hoc files if needed.

- **D-web.2 `verify` runs synchronously; `library-search` runs async with a
  status endpoint.**
  Why: verify is a single on-demand action (the UI shows a spinner); a library
  sweep is long-running and multi-item, so it mirrors `library/scan`'s async
  status pattern.
  Revise: move verify to the task framework if long verifies block the UI.

- **D-web.3 `dualsub` uploads a subtitle file and streams back the `.eo.srt`.**
  Why: subtitles are small; mirrors the existing `/api/translate` handler.

- **D-web.4 All three require the `basic` role (same as scan/translate).**
  Revise: tighten to a dedicated role if desired.

---

## W-web (frontend) — Verify page + frontend build repair

Added `webui/src/Verify.jsx` (a `/tools/verify` page) and repaired the
frontend build.

- **D-web.5 Added a "Verify Sync" tool page calling `/api/verify`.**
  Why: the drift check is the most visible new capability; the page takes media
  + subtitle paths and renders the in-sync / offset / drift result.
  Revise: add a file/library picker instead of raw path text fields; add
  `dualsub` and `library-search` UIs the same way.

- **D-web.6 Repaired the pre-existing frontend build (was broken on main).**
  Two Vite-8/Rolldown incompatibilities blocked `vite build`:
  (1) `manualChunks` must be a function, not the rollup object form — converted
  in `webui/vite.config.js`; (2) `js-yaml@5` has no default export, so
  `import yaml from 'js-yaml'` failed — changed `ConfigEditor.jsx` to
  `import * as yaml`.
  Revise: n/a (bug fixes), but see D-web.7.

- **D-web.7 KNOWN DEBT: ~19 pre-existing frontend unit tests fail (vitest) in
  files unrelated to this change (UserManagement, Wanted, …).**
  Why not fixed here: they are pre-existing (API-mock assertion drift from the
  Vite-8/dependency upgrade), unrelated to the pipeline UI, and fixing all of
  them is a separate effort. The frontend BUILD is now green; the test suite is
  not. There are no required status checks, so this does not block merges.
  Revise: a dedicated "repair frontend tests" pass.

- **D-web.8 Set `working_directories.frontend: webui` in `repository-config.yml`.**
  Why: subtitle-manager's frontend is `webui/`, not the CI default `web/`; without
  this the Frontend CI install fails with "Frontend working directory not found:
  web". This was a gap in the workflow conversion's `repository-config.yml`.
