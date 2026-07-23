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
