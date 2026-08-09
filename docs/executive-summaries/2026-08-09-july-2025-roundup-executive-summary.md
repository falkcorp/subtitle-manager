<!-- file: docs/executive-summaries/2026-08-09-july-2025-roundup-executive-summary.md -->
<!-- version: 1.0.0 -->
<!-- guid: 1a6d38f4-92c5-4b70-8e13-6c0f4a9d275b -->
<!-- last-edited: 2026-08-09 -->

# Executive Summary: July 2025 Roundup

**Shipped:** PRs [#1275–#1768](https://github.com/falkcorp/subtitle-manager/pulls?q=is%3Apr+is%3Amerged+merged%3A2025-07-01..2025-07-31),
covering 2025-07-01 through 2025-07-31 (211 merged pull requests, of which 157
were substantive and 54 were automated dependency updates).
**Related doc:** [2026-08-09-june-2025-roundup-executive-summary.md](2026-08-09-june-2025-roundup-executive-summary.md)
**Note:** Written 2026-08-09 as part of a backfill; reconstructed from the merge
record rather than contemporaneous notes.

If June built the product, July tried to make it a credible replacement for
Bazarr — the established tool in this space — and then spent its second half on a
large internal re-plumbing exercise.

## Executive Summary

- **A declared Bazarr feature-parity push.** The month opened with an explicit
  framework for organising the work needed to match the incumbent tool (#1336,
  #1357), and most of the first half executed against it.
- **Automatic best-match selection.** A subtitle quality scoring system was
  built (#1339) so the tool can rank candidates and choose rather than taking the
  first result. This is the difference between "finds subtitles" and "finds the
  right subtitle".
- **Event-driven operation.** A complete webhook system for Sonarr and Radarr
  plus outgoing notifications (#1341), and automatic episode monitoring (#1342),
  so the product reacts to new media instead of waiting to be asked.
- **Speech-to-text productionised.** Whisper (which transcribes spoken audio into
  subtitles when none exist to download) moved from a configuration flag to a
  managed container with health checks, start/stop controls, and status reporting
  (#1358, #1463, #1483, #1485, #1621).
- **Caching and rate limiting throughout.** Search results, manual searches,
  translations, and CLI queries all gained caching, and outgoing searches gained
  rate limiting (#1462, #1475–#1481, #1605). This reduces load on third-party
  subtitle sites, which is both a courtesy and a way to avoid being blocked.
- **Metadata editing became a real workflow.** Fetch, show, pick, apply, and
  interactive selection commands, plus field locks so a manual correction is not
  overwritten by the next automatic refresh (#1429–#1472).
- **Optional cloud storage.** Google Cloud Storage and Azure Blob backends for
  storing subtitles (#1430, #1441).
- **A large migration onto shared internal libraries ("gcommon").** The second
  half of the month moved configuration, authentication, metrics, health checks,
  queueing, caching, and logging onto a shared library used across the owner's
  projects (#1599–#1768, roughly 25 pull requests). No user-visible feature came
  out of this; it is infrastructure consolidation.
- **Operability.** Prometheus metrics, structured logging, a self-test service, a
  systemd service example, reverse-proxy support via a configurable base URL, and
  gRPC timeouts (#1352, #1449, #1468, #1484, #1742).
- **Process noise is visible in the record.** A number of pull requests are
  titled `[WIP]`, several are exact duplicates merged twice (#1407/#1408,
  #1464/#1466, #1473/#1474), and one is titled `[WIP] GARBAGE Optimise database
  queries` (#1362). This is recorded rather than tidied away — see the note at
  the end.

**Highest-risk items this month:**

- **#1275** — the Bazarr import handlers could be pointed at an arbitrary
  address, allowing the server to be used to reach systems it should not
  ("server-side request forgery"). URL validation added.
- **#1276, #1278, #1413, #1433, #1435** — five further path-handling holes, in
  file operations, subtitle search, and the filesystem watcher. All closed, and
  the shared security package introduced in June was adopted rather than each
  site being patched separately.

## What changed, in plain terms

### 1. The parity push

The month began by writing down what "as good as Bazarr" actually required, then
working the list. The concrete outcomes were quality scoring, webhooks, episode
monitoring, backup and restore, manual search with multi-provider support, and
progress reporting for long-running operations.

Two items on that list — language profiles (#1337, #1356) and per-title language
preferences (#1379) — were marked delivered here but, as later months prove, were
not actually connected to the download path. This is the earliest point at which
the record overstates completeness.

### 2. Choosing the right subtitle, not just any subtitle

**What was wrong:** the product could find candidates but had no opinion about
which was best.

**The fix:** a scoring system that ranks candidates on how well they match the
media file, so the best is selected automatically (#1339).

**What it means:** this is the feature that makes unattended operation
worthwhile. Without it, an automatic download is a coin flip.

### 3. Reacting instead of polling

Webhooks let Sonarr and Radarr tell the product that new media has arrived, and
episode monitoring watches for missing subtitles on an ongoing basis. Conflict
logging was added so that a sync which disagrees with local state leaves a
record (#1464). Later in the month this became cron-based automatic
synchronisation (#1633).

### 4. Whisper as managed infrastructure

Speech-to-text moved from "call this API if configured" to a managed container
with lifecycle controls and health reporting. This matters because transcription
is the fallback when no subtitle exists to download anywhere — it is the
difference between "no subtitles available" and "we made some".

### 5. Caching and politeness

Every outbound search path gained caching, and searches gained rate limiting.
Beyond the performance benefit, this protects the relationship with free
third-party subtitle sites, which will block a client that hammers them. Several
follow-up fixes were needed to get the cache key right (#1476, #1477, #1651) —
the key initially varied with provider ordering, so identical searches missed the
cache.

### 6. The gcommon migration — cost with no user-visible output

Roughly a quarter of the month's substantive pull requests moved internal
plumbing onto a shared library: configuration loading, authentication middleware,
metrics, health endpoints, the queue, caching policy, protobuf definitions, and
logging.

This produced no feature a user can see. The justification is that the same
components are then maintained once across several projects rather than
separately in each. It is flagged here because a reader comparing months by pull
request count would otherwise assume July delivered twice what it did.

### 7. Process noise

Duplicate merges, `[WIP]` titles reaching the main branch, and at least one pull
request whose title contains the word `GARBAGE` indicate that automated agents
were opening and merging work faster than it was being reviewed. The engineering
outcome was fine — the code works — but the audit trail from this month is
noticeably lower quality than later months, and reconstructing it for this
document was correspondingly harder.

## What this means going forward

July delivered the features that make unattended operation plausible: scoring,
monitoring, webhooks, transcription, and caching.

Two cautions carry forward:

- **Delivered is not the same as connected.** Language profiles were reported
  complete in both June and July and were still not consulted at download time a
  year later. Feature-completion claims from this period should be treated as
  claims about code existing, not about behaviour being wired end to end.
- **A quarter of the month bought no user-visible capability.** The gcommon
  consolidation may well have been correct, but it should be counted as
  infrastructure investment when judging what the month returned.
