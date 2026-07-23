<!-- file: docs/WHISPER_PIPELINE_DESIGN.md -->
<!-- version: 1.0.0 -->
<!-- guid: f718211e-21cd-4556-889d-ba004fd431f5 -->
<!-- last-edited: 2026-07-22 -->

# Whisper-Centric Subtitle Pipeline — Design & Build Assessment

## Purpose

This document evaluates what it takes to make Subtitle Manager deliver a
Whisper-centric workflow:

- scan and build the media library;
- pull the library from **Sonarr** and **Radarr** over their REST APIs;
- search for subtitles across providers;
- transcribe with a **Whisper** server the user provides, **or start our own**;
- **verify** an existing subtitle against the audio at several timestamps to
  detect that it has not sped up or slowed down (constant offset **and** linear
  rate drift, e.g. 23.976 ↔ 25 fps);
- use Whisper to **extract** a full subtitle, **translate** it to Mandarin, and
  emit a **double-subs** (bilingual) track tagged with an otherwise-unused
  language code (**Esperanto, `eo`**) so media players/servers do not confuse it
  with a real track.

The assessment is grounded in the current codebase (file/line references
throughout). **Bottom line: the project is ~70–80% of the way there.** Library,
`*arr`, providers, translation, and `eo` tagging are built. The missing work is
concentrated in the two Whisper-differentiated features — self-hosted
transcription and drift verification — plus the double-subs generator.

## Current-state capability matrix

| Capability | State | Evidence |
| --- | --- | --- |
| Filesystem library scan + persistence | **Built** | `pkg/metadata/metadata.go:481` `ScanLibrary` → `database.MediaItem` (`pkg/database/database.go:138`), multi-backend store (`pkg/database/store.go:36`) |
| Sonarr REST pull | **Built** | `pkg/sonarr/client.go:43` `GET /api/v3/episode?includeEpisodeFile=true`, `X-Api-Key`, maps `episodeFile.path` |
| Radarr REST pull | **Built** | `pkg/radarr/client.go:39` `GET /api/v3/movie?includeMovieFile=true` |
| Subtitle providers | **Built** | ~60 providers (`pkg/providers/registry.go`), `pkg/providers/multi.go:14` `FetchFromAll`, profile-aware fetch |
| Search driven **by** the `*arr` library | **Gap (decoupled)** | Search keys on filesystem paths (`pkg/scanner/scanner.go:114`); no code reads `store.ListMediaItems()` to drive a provider fetch |
| User-provided Whisper (OpenAI-compatible) | **Built** | `pkg/transcriber/transcriber.go:62` `WhisperTranscribe`, overridable base URL (`SetBaseURL`, `:32`) |
| Self-hosted Whisper — lifecycle | **Built** | `pkg/transcriber/whisper_container.go` start/stop/status; `cmd/whisper.go`; `docker-init.sh:36` `ENABLE_WHISPER` orchestration |
| Self-hosted Whisper — transcription | **Stub** | `pkg/transcriber/whisper_container.go:279` `transcribeWithContainer` is a placeholder pointing the OpenAI SDK at the container with `"dummy-key-for-container"`; base-URL restore bug at `:285` |
| Timestamp/segment audio extraction | **Built, orphaned** | `pkg/audio/audio.go:86` `ExtractTrackWithDuration` (ffmpeg `-ss`/`-t`) has **no non-test caller** |
| Constant-offset sync | **Built (crude)** | `pkg/syncer/syncer.go:241` `computeOffset` (median of first ≤10 cues) + `:228` `Shift` (uniform) |
| Linear/rate drift correction | **Missing** | No slope fit, no fps-ratio detection anywhere |
| Drift **verification** (sample N timestamps, report) | **Missing** | `pkg/selftest` is a DB ping; `pkg/scoring` scores download candidates, not timing |
| Whisper-extract → subtitle | **Half** | Works via API (`cmd/transcribe.go`); REST/container path rides the stub |
| Translate to Mandarin | **Built** | `pkg/translator/translator.go:254` Google/GPT/gRPC; `zh` passes through free-form |
| Double subs (orig + Mandarin in one cue) | **Missing** | `pkg/subtitles/merge.go:12` interleaves by time; `pkg/subtitles/translatefile.go:72` **overwrites** the original line |
| Tag as Esperanto (`eo`) | **Built (sidecar)** | `pkg/security/security.go:232` allows any ≤10-char alnum code; `.eo.srt` naming via `security.go:293`. No embedded-track muxing exists |

## Work items

Effort is in T-shirt sizes (S/M/L) relative to the existing code, not calendar
time.

### W1 — Library → subtitle-search bridge (S)

**Gap.** The `*arr` sync writes `MediaItem` rows that no subtitle-search codepath
reads; search is driven by directory walks (`pkg/scanner/scanner.go:28`).

**Change.** Add a routine that iterates `store.ListMediaItems()` and calls
`providers.FetchWithProfile` (or `FetchFromAll`, `pkg/providers/multi.go:14`) per
item path, using the stored `Title`/`Season`/`Episode` to build better provider
queries (the provider interface `Fetch(ctx, mediaPath, lang)` currently keys on
path + language only, `pkg/providers/provider.go:7`). Wire it into the existing
schedulers (`pkg/sonarr/scheduler.go`, `pkg/radarr/scheduler.go`) so a pull is
followed by a search pass. Pure plumbing over working parts.

### W2 — Self-hosted Whisper transcription client (M)

**Gap.** Lifecycle management is built, but `transcribeWithContainer`
(`pkg/transcriber/whisper_container.go:279`) is a stub, and there are **two
competing container designs**:

- `whisper_container.go` manages `onerahmet/openai-whisper-asr-webservice` (port
  9000, native `/asr` API);
- `pkg/transcriber/docker.go` runs `openai/whisper:latest` via the `whisper` CLI
  with a bind-mount (`TranscribeFile`, `:113`), wired only through
  `TranscribeWithMethod` (`transcriber.go:38`).

**Change.**

1. Pick one design — recommend the ASR webservice (long-running, GPU, no
   per-call container spin-up) and demote `docker.go` to an optional fallback.
2. Implement a real client for the ASR webservice protocol: multipart
   `POST /asr?task=transcribe&language=..&output=srt` (and `output=json` for word
   timings), returning SRT bytes / timed segments. This replaces the OpenAI-SDK
   shim in `transcribeWithContainer`.
3. Fix the base-URL restore bug at `whisper_container.go:285` (`oldBaseURL`
   captures the already-overwritten value).
4. Flip `ENABLE_WHISPER` on as an opt-in in the compose files (currently
   commented at `docker-compose.yml:34`, `docker-stack.yml:30`); it requires the
   `/var/run/docker.sock` mount already present at `docker-compose.yml:20`.

Result: both "use the user's Whisper" (already works via
`SetBaseURL`) and "run our own" produce real subtitles through one client
abstraction.

### W3 — Wire windowed audio extraction (S)

**Gap.** `pkg/audio/audio.go:86` `ExtractTrackWithDuration(mediaPath, track,
offset, duration)` is implemented but unused; only whole-track `ExtractTrack` is
wired (into the syncer, `pkg/syncer/syncer.go:133`).

**Change.** Expose it as the shared primitive for W2 (segment transcription) and
W4 (anchor sampling). No new ffmpeg work — just callers and a small helper that
returns `(wavPath, cleanup)`.

### W4 — Drift verification + rate correction (M–L)

This is the net-new QA capability and the most algorithmically involved item.
The reusable foundation already exists: windowed extraction (W3),
`transcriber.WhisperTranscribe` (window → text+timing), and framerate read
(`pkg/video/video.go:112`).

**Design.**

1. **Sample.** Choose N anchor windows spread across the runtime (e.g. 8–12
   windows of 30–60 s, skipping the first/last few minutes to avoid
   intros/credits). Extract each with `ExtractTrackWithDuration`.
2. **Align per anchor.** Transcribe each window with Whisper. For each window,
   measure the **local offset** between the window's spoken lines and the
   candidate subtitle's cues in that time range (cross-match by text similarity
   and/or nearest-cue start-time delta). This yields a set of
   `(t_i, offset_i)` samples — the local timing error at each anchor.
3. **Fit.** Linear-regress `offset = a·t + b`.
   - `a ≈ 0, b ≈ 0` → in sync.
   - `|b|` large, `a ≈ 0` → constant offset (what the current shifter handles).
   - `a` significantly non-zero → **rate drift**. Flag when `1 + a` is near a
     known frame-rate ratio (25/23.976 ≈ 1.04270 playback, or the
     PAL-speedup 25/24 ≈ 1.0417) to name the likely cause.
4. **Report.** A verification mode that emits per-anchor residuals, the fitted
   `(a, b)`, an `R²`/confidence, and a pass/fail against thresholds (e.g.
   `|residual| ≤ 150 ms` at every anchor). No such command/package exists today.
5. **Correct (optional).** Replace the single-median `computeOffset` + uniform
   `Shift` with a rescale-then-shift: `t' = (1 + a)·t + b`, applied to every cue
   start/end. Keep the current path as the `a = 0` special case.

**New surface.** A `pkg/subsync`/`pkg/verify` package (or an extension of
`pkg/syncer`), a `verify` CLI command, and a REST endpoint + task, plus a stored
per-subtitle verification result for the UI.

### W5 — Double-subs (bilingual) generator (S–M)

**Gap.** Nothing pairs an original cue with its translation. `MergeTracks`
(`pkg/subtitles/merge.go:12`) interleaves two tracks chronologically;
`TranslateFileToSRT` (`pkg/subtitles/translatefile.go:72,134`) replaces
`item.Lines` with the translated text.

**Design.** Add a per-cue combine step: for each source `astisub.Item`, keep the
original line and **append** the Mandarin translation as an additional
`astisub.Line` on the same item (SRT → two stacked lines per cue). For nicer
rendering, optionally emit **ASS** with the original and translation on separate
styles (top/bottom). Feed the existing `.eo.srt` output path
(`pkg/subtitles/translatefile.go:154`, `security.go:293`). Do **not** reuse
`MergeTracks`.

**Caveat — ASS styling.** Real multi-format writing with styling currently lives
only in the gRPC media server `ConvertSubtitleFormat`
(`pkg/media/server.go:173`) and is **stubbed** (`/tmp` demo paths,
`preserveStyling` ignored, `server.go:153`). Styled ASS double-subs require
finishing that writer; plain stacked-SRT double-subs do not.

### W6 — Esperanto sentinel convention (~0)

No code change. `security.ValidateLanguageCode` (`pkg/security/security.go:232`)
already permits `eo`, and the output path builder yields `video.eo.srt`. Adopt
`eo` as the sentinel for generated double-subs. **Limitation:** the system writes
sidecar files only — it never muxes a track back into the container
(`pkg/subtitles/extract.go` is one-way), so the "player won't confuse it"
benefit comes from the filename suffix, not an embedded track-language tag. If
embedded tagging is desired later, a mux-back step (ffmpeg `-c copy` with
`-metadata:s:s language=epo`) is additional work.

## Recommended build sequence

1. **W2** self-hosted Whisper transcription — unblocks everything Whisper.
2. **W1** library → search bridge — makes `*arr` → subtitles end-to-end.
3. **W3** windowed extraction wiring — shared primitive.
4. **W5** double-subs + `eo` convention — high-value, low-effort, user-visible.
5. **W4** drift verification/correction — the QA capstone.

Only **W2** and **W4** are substantial; the rest is wiring over existing,
working machinery.

## Configuration additions (indicative)

- `whisper.mode`: `external | container` (external = user-provided URL;
  container = start our own). External already supported via `SetBaseURL`.
- `whisper.asr_endpoint` / `whisper.model` / `whisper.device` — extend existing
  `whisper.*` defaults (`cmd/root.go:261`).
- `verify.anchors`, `verify.window_seconds`, `verify.max_residual_ms`,
  `verify.drift_slope_threshold` — W4 tunables.
- `doublesubs.target_lang` (default `zh`), `doublesubs.sentinel_lang` (default
  `eo`), `doublesubs.format` (`srt | ass`) — W5.

## Relationship to existing open-source tools

- **Bazarr** already covers library + `*arr` + provider search (pillars 1–3);
  this repo is partly Bazarr-aware (`pkg/bazarr/`, `docs/BAZARR_*`).
- **ffsubsync / alass** do audio-anchored sync better than the current
  constant-offset shifter (W4 draws on the same idea, but Whisper transcription
  gives text anchors instead of pure VAD/cross-correlation).
- **No open-source tool** does Whisper-extract → translate → double-subs →
  drift-QA as one integrated pipeline. That integrated flow is the
  differentiator, and it maps exactly onto W2/W4/W5.

## Risks and open questions

- **Whisper cost/latency at scale.** Transcribing every file (W1 × W2) is
  expensive; gate Whisper behind "no provider subtitle found" or "verification
  failed" rather than running it unconditionally.
- **Alignment robustness (W4).** Text-similarity matching between Whisper output
  and an existing subtitle can be noisy (paraphrase, music, silence). Mitigate
  with per-anchor confidence and by requiring agreement across ≥K anchors before
  declaring drift.
- **Two container designs (W2).** Must be reconciled to avoid maintaining both.
- **ASS styling (W5).** Styled double-subs depend on finishing the stubbed
  format writer; scope decides whether v1 ships stacked-SRT only.
- **Language target for translation backends.** `zh` vs `zh-CN`/`zh-TW`
  behaviour differs per backend (Google/GPT/gRPC); pick and validate one.
