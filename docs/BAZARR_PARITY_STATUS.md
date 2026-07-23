<!-- file: docs/BAZARR_PARITY_STATUS.md -->
<!-- version: 1.4.0 -->
<!-- guid: 2b9f4a1e-8c3d-4f76-9a05-1d7e6b2c4f88 -->
<!-- last-edited: 2026-07-23 -->

# Bazarr Backend Parity — Status & Gap Inventory

Grounded audit of subtitle-manager's backend against
[`BAZARR_FEATURE_LIST_COMPLETE.md`](BAZARR_FEATURE_LIST_COMPLETE.md), verified by
reading the code (2026-07-23). Status legend: **✅ Built**, **🟡 Partial**,
**🔴 Missing**. "Partial" means the plumbing exists but the behaviour is not
wired into the live path.

## The load-bearing caveat: the provider layer

**~50 of ~55 registered subtitle providers are non-functional stubs** — each
GETs a fictional `https://api.<name>.com/subtitles/<file>/<lang>` endpoint
(`pkg/providers/*/*.go`). Real integrations: **opensubtitles** (hash + REST),
**napiprojekt** (keyless hash protocol), plus **embedded** (ffmpeg track
extraction) and the configurable **generic** / **whisper** HTTP pass-throughs.
Bazarr's provider breadth comes from Python's `subliminal`; porting dozens of
real scrapers to Go is a large, separate effort.

### The provider boundary (what can be done autonomously vs. what needs you)

Porting the remaining stubs splits into three buckets:

- **Keyless / anonymous (implementable & unit-testable now):** hash- or
  anonymous-search services that need no account. `napiprojekt` is done as the
  reference implementation. A handful of others are theoretically keyless but
  rely on **fragile HTML scraping of live sites** (e.g. `podnapisi`,
  `yifysubtitles`, `subscene`), which cannot be implemented correctly or tested
  offline without replicating each site's exact markup — high effort, brittle.
- **Credential-gated (BLOCKED — needs the operator):** the majority require an
  account, API key, or paid tier the agent cannot obtain — e.g. `addic7ed`,
  `titlovi`, `legendasnet`, `ktuvit`, `avistaz`/`hdbits`/`karagarga`
  (private trackers), `betaseries`, `assrt`. These stay stubs until you supply
  credentials and confirm terms-of-service allow automated access.
- **Large scraping track:** everything else is site-specific HTML scraping with
  no stable API; porting them is the bulk of the `subliminal` effort and should
  be scoped deliberately, provider by provider.

**Net:** automatic search/download works through **opensubtitles** and
**napiprojekt** today; `embedded` covers muxed tracks. Going materially further
requires either operator-supplied credentials or a deliberate, funded scraping
effort — it is not fully achievable autonomously.

## Matrix

### Sonarr / Radarr
| Capability | Status | Notes |
| --- | --- | --- |
| REST library sync (episodes/movies, host/port/ssl/base/key) | ✅ | `pkg/sonarr/client.go`, `pkg/radarr/client.go` |
| Scheduled sync wired into webserver | 🟡 | `StartContinuousSync` unused; monitor loop doesn't re-pull |
| Minimum-score threshold applied to *arr downloads | 🟡 | global scorer exists, not applied at download |
| Monitored-only mode | 🔴 | `monitored` field not even decoded |
| Excluded tags / series types | 🔴 | not decoded/filtered |
| Path mappings (arr↔local) applied | 🔴 | import-only stub; never applied |
| Webhook on import → search | 🟡 | only `Download` event; hardcoded `en`; no profile/score |

### Search / download / upgrade engine
| Capability | Status | Notes |
| --- | --- | --- |
| Provider registry breadth | 🔴 | ~52 stubs; 1 real (opensubtitles) |
| Automatic "wanted" search loop (scheduled) | 🟡 | monitor skeleton exists; no score gate |
| Manual search (ranked candidates) | 🟡 | `/api/search`; only opensubtitles implements `Searcher` |
| Score-gated accept/reject (min-score) | 🟡 | `pkg/scoring` not wired into auto-download |
| Score-based upgrade | 🔴 | upgrade decided by file size, not score |
| Adaptive searching | 🟡 | fixed retry→blacklist; in-memory provider backoff |
| Parallel provider fetch (core path) | 🟡 | `multi.go` is serial with backoff sleeps |
| Use embedded tracks as a source | 🟡 | ffmpeg extract exists but not a download source; `embedded` provider is a stub |
| Ignore PGS/image subtitles | 🔴 | not filtered |
| Desired-languages via profiles | ✅ | `FetchWithProfile` iterates by priority |

### Post-processing / languages / profiles
| Capability | Status | Notes |
| --- | --- | --- |
| Language profiles (multi-lang, priority, cutoff) | ✅ | `pkg/profiles` + REST/CLI |
| Forced / HI honored at download & naming | ✅ | per-language Forced/HI mapped to scoring prefs in `FetchWithProfile` |
| Default profile auto-assigned to new *arr items | 🟡 | single global default; no auto-assign |
| Mass-edit (bulk profile assign) | 🔴 | single-item only |
| Single-language filename option | ✅ | `subtitles.single_language` → `video.srt` |
| **UTF-8 re-encoding** | ✅ **(this PR)** | `pkg/postprocess.EncodeUTF8` wired into `ProcessFile` |
| **chmod on written subtitles** | ✅ **(this PR)** | `postprocess.chmod` |
| **Auto-sync after download** | ✅ **(this PR)** | `postprocess.auto_sync` |
| **Custom post-download script** | ✅ **(this PR)** | `postprocess.custom_script` |
| Format conversion on download (ASS/VTT) | 🟡 | astisub writer real but gRPC path stubbed; download emits `.srt` only |
| Anti-captcha wired to providers | 🟡 | solver exists, never called |
| History retention/depth | 🟡 | records written; no pruning |
| Blacklist (per-subtitle) | 🟡 | not persisted; item-level only |

### Infrastructure / settings
| Capability | Status | Notes |
| --- | --- | --- |
| Auth (form/API-key/OAuth) | ✅ | (Basic-auth challenge not offered) |
| Bind addr/port + base_url | ✅ | |
| DB backends (sqlite/pebble/postgres) | ✅ | auth uses a side sqlite `auth.db` |
| Metrics/health/logging | ✅ | |
| Config persistence + settings API | ✅ | `/api/config` + viper WriteConfig |
| Scheduler with persisted per-task intervals | 🟡 | CLI-flag driven; not config/UI per-job |
| Outbound proxy (HTTP/Socks) for providers | 🔴 | none |
| Notifications (Apprise / many channels) | 🟡 | 3 fixed channels; no Apprise |
| Plex incoming webhook | 🔴 | only Sonarr/Radarr/custom |
| External-Whisper model/timeout config | 🟡 | selectable only for the local container |

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
