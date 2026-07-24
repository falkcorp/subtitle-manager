<!-- file: docs/BAZARR_PARITY_STATUS.md -->
<!-- version: 1.7.0 -->
<!-- guid: 2b9f4a1e-8c3d-4f76-9a05-1d7e6b2c4f88 -->
<!-- last-edited: 2026-07-24 -->

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
plus **embedded** (ffmpeg track extraction) and the
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
  `podnapisi` (advanced-search JSON API, movies + TV) are done as the reference
  implementations. A handful of others are theoretically keyless but
  rely on **fragile HTML scraping of live sites** (e.g. `yifysubtitles`,
  `subscene`), which cannot be implemented correctly or tested
  offline without replicating each site's exact markup — high effort, brittle.
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
| Webhook on import → search | 🟡 | Sonarr/Radarr/custom + Plex; still hardcoded `en` |
| Notify *arr to rescan after download | 🔴 | needs media→*arr-id tracking we don't keep |

### Search / download / upgrade engine
| Capability | Status | Notes |
| --- | --- | --- |
| Provider registry breadth | 🔴 | ~49 stubs; 3 real (opensubtitles, napiprojekt, gestdown) + embedded |
| Whisper fallback (transcribe when no provider match) | ✅ | `whisper.fallback_enabled` (`pkg/scanner.whisperFallback`) |
| Automatic "wanted" search loop (scheduled) | 🟡 | monitor skeleton; score gate now available |
| Manual search (ranked candidates) | 🟡 | `/api/search`; only opensubtitles implements `Searcher` |
| Score-gated accept/reject (min-score) | ✅ | wired into `ProcessFile` + `FetchWithProfile` |
| Score-based upgrade | ✅ | compares persisted `DownloadRecord.MatchScore` |
| Adaptive searching | 🟡 | fixed retry→blacklist; in-memory provider backoff |
| Parallel provider fetch (core path) | 🟡 | `multi.go` is serial with backoff sleeps |
| Use embedded tracks as a source | ✅ | `embedded` provider extracts muxed tracks via ffmpeg |
| Ignore PGS/image subtitles | ✅ | `video.SubtitleStream.ImageBased()` skipped by embedded |
| Desired-languages via profiles | ✅ | `FetchWithProfile` iterates by priority |

### Post-processing / languages / profiles
| Capability | Status | Notes |
| --- | --- | --- |
| Language profiles (multi-lang, priority, cutoff) | ✅ | `pkg/profiles` + REST/CLI |
| Forced / HI honored at download & naming | ✅ | per-language Forced/HI mapped to scoring prefs in `FetchWithProfile` |
| Default profile auto-assigned to new *arr items | 🟡 | single global default; no auto-assign |
| Mass-edit (bulk profile assign) | 🔴 | single-item only |
| Single-language filename option | ✅ | `subtitles.single_language` → `video.srt` |
| **UTF-8 re-encoding** | ✅ | `pkg/postprocess.EncodeUTF8` wired into `ProcessFile` |
| **chmod on written subtitles** | ✅ | `postprocess.chmod` |
| **Auto-sync after download** | ✅ | `postprocess.auto_sync` |
| **Custom post-download script** | ✅ | `postprocess.custom_script` |
| Custom post-processing script variables (score/provider) + threshold | ✅ | `SM_PROVIDER`/`SM_SCORE`; `postprocess.score_threshold` |
| Format conversion on download (ASS/VTT) | 🟡 | astisub writer real but gRPC path stubbed; download emits `.srt` only |
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

- 🔴 **Notify Sonarr/Radarr after download** so they index the new subtitle —
  needs a media→*arr-item-id mapping we don't currently persist.
- 🔴 **Real-time (SignalR) sync** with Sonarr v3 / Radarr v3 — large; needs a
  SignalR client. We poll/sync via REST only.
- 🟡 **Provider throttling** — Bazarr tracks per-provider throttle state and a
  "throttled providers" view with cooldowns on 429/503/maintenance; we only have
  simple in-memory backoff.
- 🟡 **Whisper**: separate connection vs read timeouts; language mapping; audio
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
