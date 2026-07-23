<!-- file: docs/BAZARR_PARITY_STATUS.md -->
<!-- version: 1.0.0 -->
<!-- guid: 2b9f4a1e-8c3d-4f76-9a05-1d7e6b2c4f88 -->
<!-- last-edited: 2026-07-23 -->

# Bazarr Backend Parity — Status & Gap Inventory

Grounded audit of subtitle-manager's backend against
[`BAZARR_FEATURE_LIST_COMPLETE.md`](BAZARR_FEATURE_LIST_COMPLETE.md), verified by
reading the code (2026-07-23). Status legend: **✅ Built**, **🟡 Partial**,
**🔴 Missing**. "Partial" means the plumbing exists but the behaviour is not
wired into the live path.

## The load-bearing caveat: the provider layer

**~52 of ~55 registered subtitle providers are non-functional stubs** — each
GETs a fictional `https://api.<name>.com/subtitles/<file>/<lang>` endpoint
(`pkg/providers/*/*.go`). Only **opensubtitles** is a real integration;
`generic` and `whisper` are configurable HTTP pass-throughs. Bazarr's provider
breadth comes from Python's `subliminal`; porting dozens of real scrapers to Go
is a large, separate effort. **Until that is done, automatic search/download
works only through opensubtitles**, regardless of how good the surrounding
engine is. This is the single biggest parity gap and is tracked separately.

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
| Forced / HI honored at download & naming | 🟡 | flags stored, not honored |
| Default profile auto-assigned to new *arr items | 🟡 | single global default; no auto-assign |
| Mass-edit (bulk profile assign) | 🔴 | single-item only |
| Single-language filename option | 🔴 | always `video.<lang>.srt` |
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

1. **Post-processing pipeline** — UTF-8, chmod, auto-sync, custom script. **done (this PR).**
2. Score-gated auto-download + score-based upgrade (wire `pkg/scoring` into `ProcessFile`).
3. Sonarr/Radarr: decode+apply `monitored`, `tags`, `seriesType`; apply path mappings.
4. Real `embedded` source (ffmpeg extract) preferred before providers.
5. Profile mass-edit; honor Forced/HI + cutoff-score; single-language filename option.
6. Infra: outbound proxy; Plex webhook; external-Whisper model/timeout; Apprise notifications.
7. Blacklist persistence; history retention.
8. **(Large, separate track)** replace the ~52 stub providers with real integrations.
