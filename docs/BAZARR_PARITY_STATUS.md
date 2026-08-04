<!-- file: docs/BAZARR_PARITY_STATUS.md -->
<!-- version: 1.15.0 -->
<!-- guid: 2b9f4a1e-8c3d-4f76-9a05-1d7e6b2c4f88 -->
<!-- last-edited: 2026-08-04 -->

# Bazarr Backend Parity — Status & Gap Inventory

Grounded audit of subtitle-manager's backend against
[`BAZARR_FEATURE_LIST_COMPLETE.md`](BAZARR_FEATURE_LIST_COMPLETE.md), verified by
reading the code and **cross-referenced against Bazarr's full upstream release
history** (2026-07-23; see the Changelog cross-reference section at the end).
Status legend: **✅ Built**, **🟡 Partial**, **🔴 Missing**. "Partial" means the
plumbing exists but the behaviour is not wired into the live path.

## The load-bearing caveat: the provider layer

**~49 of ~55 registered subtitle providers are non-functional stubs** — each
GETs a fictional `https://api.<name>.com/subtitles/<file>/<lang>` endpoint
(`pkg/providers/*/*.go`). Real integrations: **opensubtitles** (hash + REST),
**napiprojekt** (keyless hash protocol), **gestdown** (keyless Addic7ed REST
proxy, TV-only), **podnapisi** (keyless advanced-search JSON API, movies + TV),
**wizdom** (keyless JSON API, Hebrew only, movies + TV), plus **embedded** (ffmpeg track extraction) and the
configurable **generic** / **whisper** HTTP pass-throughs. Bazarr's provider
breadth comes from Python's `subliminal`; porting dozens of real scrapers to Go
is a large, separate effort.

> **Note for the Phase 2 credential work:** the `opensubtitlescom` package
> (the `.com` REST v1 provider) is still a *stub* hitting a fictional
> `api.opensubtitlescom.com` host — despite the build-out prompt calling it
> "already implemented". The real, working OpenSubtitles integration is the
> classic `opensubtitles` package. Trust the code, not the prompt's
> "already implemented" claims, for each Phase 2 provider.

### The provider boundary (what can be done autonomously vs. what needs you)

Porting the remaining stubs splits into three buckets:

- **Keyless / anonymous (implementable & unit-testable now):** hash- or
  anonymous-search services that need no account. `napiprojekt` (hash protocol),
  `gestdown` (Addic7ed REST proxy, filename→episode via `pkg/metadata`), and
  `podnapisi` (advanced-search JSON API, movies + TV) and `wizdom` (JSON API
  keyed on IMDb ID, Hebrew only) are done as the reference implementations.
  Others really are scraping-only and were skipped on evidence, not on
  reputation: `yifysubtitles` serves HTML with no API, and `subscene` answers
  403 behind a Cloudflare interstitial. The per-candidate probe results are
  tabulated in `PROVIDER_BUILDOUT_PROMPT.md`.

  Two lessons from this pass are worth keeping. Do not trust a "fragile HTML
  scraping" label without probing — it was wrong for `podnapisi`, which has a
  clean JSON API, and it caused `wizdom` to be written off without ever being
  looked at. And a provider is not finished when a mocked test passes: wizdom's
  subtitles are Windows-1255, which the response validator rejected outright as
  "not a subtitle" until that was fixed, and only a live fetch surfaced it.
- **Credential-gated (BLOCKED — needs the operator):** the majority require an
  account, API key, or paid tier the agent cannot obtain — e.g. `addic7ed`,
  `titlovi`, `legendasnet`, `ktuvit`, `avistaz`/`hdbits`/`karagarga`
  (private trackers), `betaseries`, `assrt`. These stay stubs until you supply
  credentials and confirm terms-of-service allow automated access.
- **Large scraping track:** everything else is site-specific HTML scraping with
  no stable API; porting them is the bulk of the `subliminal` effort and should
  be scoped deliberately, provider by provider.

**Net:** automatic search/download works through **opensubtitles**,
**napiprojekt**, **gestdown** (TV episodes), and **podnapisi** (movies + TV)
today; `embedded` covers muxed
tracks. Going materially further
requires either operator-supplied credentials or a deliberate, funded scraping
effort — it is not fully achievable autonomously.

## Matrix

### Sonarr / Radarr
| Capability | Status | Notes |
| --- | --- | --- |
| REST library sync (episodes/movies, host/port/ssl/base/key) | ✅ | `pkg/sonarr/client.go`, `pkg/radarr/client.go` |
| Configurable *arr request timeout | ✅ | `integrations.{sonarr,radarr}.timeout` (`pkg/arr.Timeout`) |
| Scheduled sync wired into webserver | 🟡 | `StartContinuousSync` unused; monitor loop doesn't re-pull |
| Minimum-score threshold applied to *arr downloads | ✅ | `scoring.min_score` gate in scored path (when `scoring.enabled`) |
| Monitored-only mode | ✅ | `arr.Filters.MonitoredOnly` |
| Excluded tags / series types | ✅ | `arr.Filters` excluded tags/series-types |
| Path mappings (arr↔local) applied | ✅ | `arr.Filters.PathMappings` |
| Webhook on import → search | 🟡 | Sonarr/Radarr/custom + Plex; language now configurable (`languages.default`), still not per-series profile |
| Notify *arr to rescan after download | ✅ | `pkg/arrnotify`; id resolved from path at event time (see note below) |

### Search / download / upgrade engine
| Capability | Status | Notes |
| --- | --- | --- |
| Provider registry breadth | 🔴 | ~48 stubs; 5 real (opensubtitles, napiprojekt, gestdown, podnapisi, wizdom) + embedded. Keyless candidates are exhausted **on evidence** — see the probe table in `PROVIDER_BUILDOUT_PROMPT.md`; the rest need operator credentials or are scraping-only |
| Whisper fallback (transcribe when no provider match) | ✅ | `whisper.fallback_enabled` (`pkg/scanner.whisperFallback`) |
| Automatic "wanted" search loop (scheduled) | 🟡 | monitor skeleton; score gate now available |
| Manual search (ranked candidates) | 🟡 | `/api/search`; only opensubtitles implements `Searcher` |
| Score-gated accept/reject (min-score) | ✅ | wired into `ProcessFile` + `FetchWithProfile` |
| Score-based upgrade | ✅ | compares persisted `DownloadRecord.MatchScore` |
| Adaptive searching | 🟡 | fixed retry→blacklist; in-memory provider backoff |
| Parallel provider fetch (core path) | ✅ | `multi.go` runs waves of 4, resolved in priority order |
| Use embedded tracks as a source | ✅ | `embedded` provider extracts muxed tracks via ffmpeg |
| Ignore PGS/image subtitles | ✅ | `video.SubtitleStream.ImageBased()` skipped by embedded |
| Desired-languages via profiles | 🟡 | library scans honour an assigned profile's languages in priority order (`scanner.ScanDirectoryProgress`); monitor, watcher, webhooks, scheduler and *arr still fetch one language |

### Post-processing / languages / profiles
| Capability | Status | Notes |
| --- | --- | --- |
| Language profiles (multi-lang, priority, cutoff) | 🟡 | multi-language and priority order consumed by library scans; the profile's **cutoff score is not** — `ProcessFile` scores against global `scoring.*` config, not `profile.CutoffScore` |
| Forced / HI honored at download & naming | 🔴 | only `FetchWithProfile` mapped per-language Forced/HI onto scoring prefs, and no download path calls it; the wired scan path goes through `ProcessFile`, which reads global scoring config |
| Default profile auto-assigned to new *arr items | 🟡 | single global default; no auto-assign |
| Mass-edit (bulk profile assign) | ✅ | `POST /api/media/profiles/bulk`, per-item results; Mass Edit mode in `MediaLibrary.jsx` |
| Single-language filename option | ✅ | `subtitles.single_language` → `video.srt` |
| **UTF-8 re-encoding** | ✅ | `pkg/postprocess.EncodeUTF8` wired into `ProcessFile` |
| **chmod on written subtitles** | ✅ | `postprocess.chmod` |
| **Auto-sync after download** | ✅ | `postprocess.auto_sync` |
| **Custom post-download script** | ✅ | `postprocess.custom_script` |
| Custom post-processing script variables (score/provider) + threshold | ✅ | `SM_PROVIDER`/`SM_SCORE`; `postprocess.score_threshold` |
| Format conversion on download (ASS/VTT) | ✅ | `subtitles.format` (srt/vtt/ass) applied at the single write path — manual download, scan and monitor loop; falls back to SRT if a payload cannot be converted. Config-file only (like `subtitles.single_language`); gRPC path still stubbed |
| Anti-captcha wired to providers | 🟡 | solver exists, never called (moot until providers are real) |
| History retention/depth | ✅ | `history.retention_days` (`maintenance.PruneDownloadHistory`) |
| Blacklist (per-subtitle) | ✅ | persisted w/ reason+expiry via `database.BlacklistStore` |

### Infrastructure / settings
| Capability | Status | Notes |
| --- | --- | --- |
| Auth (form/API-key/OAuth) | ✅ | (Basic-auth challenge not offered) |
| Bind addr/port + base_url | ✅ | |
| DB backends (sqlite/pebble/postgres) | ✅ | auth uses a side sqlite `auth.db` |
| Metrics/health/logging | ✅ | |
| Config persistence + settings API | ✅ | `/api/config` + viper WriteConfig |
| Scheduler with persisted per-task intervals | 🟡 | CLI-flag driven; not config/UI per-job |
| Outbound proxy (HTTP/Socks) for providers | ✅ | `proxy_url` → `http.DefaultTransport` (`pkg/proxy`) |
| Notifications (Apprise / many channels) | ✅ | Apprise + Discord/Telegram/email; **fire on subtitle events** |
| Plex incoming webhook | ✅ | `POST /api/webhooks/plex` (`library.new`) |
| External-Whisper URL/model/timeout config | ✅ | `whisper.transcribe_url` (native /asr), model, timeout |
| Provider status / throttle view | ✅ | `/api/providers/status` computed on read (enabled/throttled/last success); no UI consumer yet |
| Per-provider enable/priority/tags from config | ✅ | `providers.LoadFromConfig` at server start + on settings save; no-op when unconfigured |

## Open findings (2026-08-01): language profiles

Three defects found while starting on mass-edit. All were verified against
running code, not inferred from the call graph. The first is fixed; the rest
are open.

### Per-item assignments collided on one key — **fixed** (PR #2230)

The web UI identifies a media item by its file path, percent-encoded into the
URL. `net/http` decodes the request target before a handler runs, so the `%2F`
separators became indistinguishable from real ones and `mediaProfilesHandler`
took only the first path segment as the identifier. Every file under `/media`
therefore keyed as `media`, and each assignment overwrote the last.

Fixed by taking the whole remainder of the path (`pathRest`) and
`filepath.Clean`ing it so the key matches what the CLI looks up. Existing rows
are keyed on a truncated segment and are not migrated — they recorded a
collision, not a usable assignment.

### Language profiles are not wired to downloads — **partly fixed** (PR #2232)

Library scans now honour an assigned profile. Everything else still does not,
and two parts of a profile are ignored even on the fixed path.

**What is fixed.** `scanner.ScanDirectoryProgress` resolves each file's
assigned profile through the same store and the same `filepath.Clean`-ed path
key the web handler writes, then calls `ProcessFile` once per language in
priority order. `ProcessFileWithProfile` was rewritten to delegate to
`ProcessFile` rather than fetch and write inline, which had bypassed score
gating, the Whisper fallback, post-processing, the *arr rescan event and
download history. `pkg/webserver/scan.go` was also switched to
`database.GetSharedStore()`: it had been calling `database.OpenStore`, which
fails under Pebble's exclusive lock, and the swallowed error left the scan
with a nil store — without that fix the wiring would have been a no-op in
production with a fully green test suite.

**What is still open.**

- ~~Only library scans.~~ **Fixed 2026-08-04.** The monitor loop, watcher, the
  Sonarr/Radarr webhooks and the `sonarr`/`radarr` CLI commands now route
  through `scanner.ProcessWithProfileIfAssigned`, which honours the file's own
  profile and reports whether it did so the caller can fall through.

  **Precedence, decided here:** an assigned profile beats
  `MonitoredItem.Languages`. That field is *accumulated*, not curated —
  `sync.go` unions in every language it is ever asked for and never removes one,
  so it cannot express "stop wanting German" — while an assignment is a
  deliberate per-file choice. Pinned by
  `TestAssignedProfileBeatsMonitoredItemLanguages`.

  Deliberately **not** overridden: requests that name a language directly, i.e.
  the web download endpoint's `?lang=`, the custom webhook's validated `lang`,
  and `fetch <file> <lang>`. Those are one specific request, not a policy.
- **Cutoff score and Forced/HI are dropped.** `processWithAssignedProfile`
  passes a bare language code, so `ProcessFile` scores candidates with
  `scoring.LoadProfileFromConfig()` — the global `scoring.*` settings. The
  per-language `Forced`/`HI` preferences and the profile's `CutoffScore` never
  reach the scorer. Only the retired `scoredProfileFetch` ever applied them.
- ~~A file explicitly assigned the default profile reads as unassigned.~~
  **Fixed 2026-08-04.** `GetAssignedProfileID` was added to `SubtitleStore` and
  all three implementations; it reports whether an assignment row exists,
  separately from which profile governs the file, and never falls back or
  creates rows. `assignedProfileLanguages` now uses it instead of inferring
  assignment from "resolved to something other than the default".

The original diagnosis, for context:

There are two `media_profiles` implementations sharing a table name:

- The web UI and CLI write **path-keyed** rows through
  `database.SubtitleStore` (`AssignProfileToMedia`).
- `pkg/profiles.Service` — used only by `pkg/providers/profiles.go` — reads
  **integer-keyed** rows out of a raw `*sql.DB`, resolving
  `media_items.path → id` first (`GetMediaProfileByPath`). Under the pebble or
  postgres backends that is not even the same database.

Independently, `scanner.ProcessFileWithProfile` has no non-test callers. Every
real download path — web library scan, monitor loop, watcher, webhooks,
scheduler, Sonarr/Radarr — calls `scanner.ProcessFile` with a single language.
`FetchWithProfile` is reachable only from `subtitle-manager fetch --use-profile`,
which opens its own SQLite store regardless of the configured backend.

So a profile's language list, priority order, cutoff score and Forced/HI flags
have no effect on what gets downloaded. This is what demoted the three matrix
rows above.

Note when fixing: `MonitoredItem.Languages` is a second, independent
desired-languages mechanism. Making the monitor loop profile-driven means
deciding which of the two wins — a product decision, not a refactor.

### The last language profile cannot be deleted — **fixed 2026-08-04**

`GetDefaultLanguageProfile` returned the first profile in the list when none was
explicitly marked default, and `handleDeleteProfile` refused to delete the
default. With one profile left, that profile was always "the default", so
`DELETE /api/profiles/{id}` answered 400 forever. Reproduced on the preview
instance. With several profiles and none flagged, an arbitrary one was likewise
permanently undeletable.

The guard now reads the `IsDefault` flag from the list rather than asking
`GetDefaultLanguageProfile`, and deleting the *only* profile is permitted —
there is no other profile for scans to fall back to either way, so refusing
just left the user stuck.

A second defect sat behind it, and fixing only the guard would have left the
bug reappearing one request later: `PebbleStore.GetDefaultLanguageProfile`
**created** a profile when the store was empty, and `GetMediaProfile` called it
on every miss. Deleting the last profile therefore lasted only until the next
lookup, which wrote it back — flagged default, and so refused again. Pebble no
longer creates on read, matching `SQLStore`, which never did.

Also fixed: `cmd/profiles.go` hardcoded `database.OpenStore(..., "pebble")`,
ignoring `db_backend`, so `subtitle-manager profiles show` read a different
store from the web UI on a sqlite or postgres deployment.

## Implementation plan (backend)

Ordered by value × tractability. Items marked **done** are landing now.

1. **Post-processing pipeline** — UTF-8, chmod, auto-sync, custom script. **done.**
2. **Score-gated auto-download + score-based upgrade** — `pkg/scoring` is wired
   into `ProcessFile` for the OpenSubtitles provider via an optional
   `scoredProvider` capability: search → score → select best above
   `scoring.min_score` → download that specific candidate; upgrades compare the
   new candidate's score against the persisted `DownloadRecord.MatchScore`.
   Gated behind `scoring.enabled` (off by default). **done.**
3. Sonarr/Radarr: decode+apply `monitored`, `tags`, `seriesType`; apply path mappings. **done.**
4. Real `embedded` source (ffmpeg extract) preferred before providers. **done.**
5. Profile options — **done**. Single-language filename (`subtitles.single_language`);
   per-language Forced/HI preferences and the profile `CutoffScore` are now
   honored in `FetchWithProfile` via a score-gated OpenSubtitles fetch
   (`scoring.SelectBestResult`), active when `scoring.enabled`. (`pkg/providers`
   importing `pkg/scoring` is cycle-free, so no scanner refactor was needed.)
6. Infra: outbound proxy **done**; Plex webhook **done**; external-Whisper
   model/URL/timeout **done**; Apprise notifications **done** (plus subtitle
   events now actually reach all notification channels).
7. Blacklist persistence + history retention — **done**. History retention
   (`history.retention_days`) prunes old download records. Blacklist entries
   (reason + expiry) now persist via a new `BlacklistStore` optional-capability
   interface implemented across sqlite/postgres/pebble; `IsBlacklisted` honors
   expiry and `CleanupExpiredBlacklist` actually removes expired entries.
8. **(Large, separate track)** replace the stub providers with real integrations.
   `napiprojekt` **done** (keyless hash protocol) as the reference; see the
   provider-boundary section above for what is keyless vs. credential-gated vs.
   scraping-only. The bulk is credential-gated or site-scraping and is not
   fully achievable without operator input.

## Changelog cross-reference (upstream Bazarr release notes)

Cross-referenced against Bazarr's full release history (all releases, ~1,956
lines of notes) to catch backend features the code-audit matrix above missed.
Backend items only — pure UI and bugfixes excluded.

### Implemented from the changelog scan

- **Whisper fallback** — transcribe when no provider has a match
  (`whisper.fallback_enabled`).
- **Configurable Sonarr/Radarr request timeout** (`integrations.*.timeout`).
- **Custom post-processing variables** `SM_PROVIDER`/`SM_SCORE` and a
  **score threshold** (`postprocess.score_threshold`).
- **Configurable scores** (min-score gate) and **score-based upgrade**.
- **Outbound proxy**, **Apprise notifications firing on subtitle events**,
  **Plex webhook**, **provider throttling (basic backoff exists)**.

### Remaining backend gaps (from the changelog)

- ✅ **Notify Sonarr/Radarr after download** so they index the new subtitle —
  implemented in `pkg/arrnotify` as an `events.EventPublisher` subscriber, using
  the same commands Bazarr sends (`RescanSeries`/`seriesId`,
  `RescanMovie`/`movieId` on `POST /api/v3/command`). Enabled by default for any
  enabled *arr; opt out with `integrations.{sonarr,radarr}.rescan_after_download: false`.

  **Design note.** The earlier claim that this "needs a media→*arr-item-id
  mapping we don't persist" overstated the gap. The id is resolved from the
  media path at event time instead: the *arr's own folder listing is fetched
  (cached for 5 minutes, since a library scan can emit hundreds of downloads)
  and matched by longest containing folder. Matching happens in
  subtitle-manager's path space — `Filters.MapPath` is applied to the *arr's
  folders rather than inverted, because `MapPath` is longest-prefix-wins and so
  not reliably invertible. Ambiguous matches are skipped rather than guessed.

  Persisting `SeriesID`/`MovieID` on `database.MediaItem` was considered and
  rejected for now: `media_items` is written by hand-rolled SQL in ~15 places
  across sqlite/postgres/pebble with no migration framework, making a column
  addition disproportionately wide. The accepted cost is that a rescan is lost
  (logged, not retried) when the *arr is unreachable at that moment. If that
  proves to matter, persisting the id becomes a focused follow-up.
- 🔴 **Real-time (SignalR) sync** with Sonarr v3 / Radarr v3 — large; needs a
  SignalR client. We poll/sync via REST only.
- 🟡 **Provider throttling** — Bazarr tracks per-provider throttle state and a
  "throttled providers" view with cooldowns on 429/503/maintenance; we only have
  simple in-memory backoff.
- 🟡 **Whisper**: connect vs total timeout now split (`whisper.connect_timeout`); language mapping; audio
  delay detection in MKV headers via ffprobe. We expose a single timeout.
- 🟡 **Subsync**: option to use the original-language audio track as the sync
  reference; ffsubsync progress reporting.
- 🟡 **Scheduled search & upgrade loop** (incl. Weekly option) driving the
  wanted/upgrade queues — our monitor skeleton doesn't drive it yet.
- 🔵 **Anti-captcha** integration — solver exists but is never called; moot until
  providers are real (blocked with the provider track).
- 🔵 **Webhook `hostname` anti-poisoning setting** — minor hardening.
- 🔵 **Provider-specific settings** (SSL toggle, `subx` key, uploader filters,
  `hi_fallback`) — blocked with their (credential-gated/scraping) providers.

Legend: 🔴 not started · 🟡 partial · 🔵 blocked/low-priority.
