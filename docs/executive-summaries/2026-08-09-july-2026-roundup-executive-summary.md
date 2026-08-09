<!-- file: docs/executive-summaries/2026-08-09-july-2026-roundup-executive-summary.md -->
<!-- version: 1.0.0 -->
<!-- guid: b73f5a08-1d26-4e94-a3c7-58e0d1f6b24a -->
<!-- last-edited: 2026-08-09 -->

# Executive Summary: July 2026 Roundup

**Shipped:** PRs [#2161–#2227](https://github.com/falkcorp/subtitle-manager/pulls?q=is%3Apr+is%3Amerged+merged%3A2026-07-01..2026-07-31),
covering 2026-07-01 through 2026-07-31 (66 merged pull requests, 60 substantive).
**Related doc:** [2026-08-09-maintenance-period-2025-08-to-2026-06-executive-summary.md](2026-08-09-maintenance-period-2025-08-to-2026-06-executive-summary.md)
**Note:** Written 2026-08-09 as part of a backfill; reconstructed from the merge
record.

After ten dormant months, the project restarted — and the restart immediately
exposed how much of what had been reported as finished did not actually work.
This month delivered more genuine Bazarr-parity capability than any month since
July 2025, and simultaneously produced the first honest inventory of what was
broken.

## Executive Summary

- **The build and release pipeline was modernised first.** Bespoke workflows were
  replaced with shared standard ones, and the resulting failures — a genuine data
  race and a coverage gate — were fixed rather than suppressed (#2167, #2170,
  #2171).
- **A large batch of real Bazarr-parity features landed.** Subtitle
  post-processing, score-gated automatic download and upgrades, a persistent
  blacklist with reasons and expiry, download-history retention, an outbound
  proxy, a Plex webhook receiver, notifications with Apprise support, and
  configurable subtitle naming (#2181–#2196).
- **Four subtitle sources were made real.** Napiprojekt, Gestdown, Podnapisi, and
  Wizdom were implemented as working keyless providers — meaning they need no
  paid account (#2190, #2201, #2202, #2227).
- **The speech-to-text pipeline was made to actually transcribe.** The
  self-hosted Whisper client was fixed, given separate connection and
  transcription timeouts, wired in as a fallback when no provider has a subtitle,
  and paired with drift verification that detects a subtitle running fast or slow
  against the audio (#2175, #2176, #2194, #2219).
- **A significant discovery: the web interface was calling API endpoints that
  were never connected.** Language profiles, rescan-all, the wanted list, and
  provider availability all had working back-end code and no route to reach it.
  Because of how the site serves pages, these failed silently rather than
  visibly (#2207, #2211, #2224, #2179).
- **A database defect was breaking multiple features at once.** The embedded
  database allows only one writer per process; the web server was opening a new
  connection per request, so any feature relying on it failed with a server error
  (#2209).
- **Bilingual subtitles.** Generation of a single subtitle file containing two
  languages at once (#2174).
- **Sonarr/Radarr integration deepened.** Monitored-only filtering, tag and
  series-type exclusion, path mappings between differing directory layouts,
  configurable timeouts, and an automatic rescan after a subtitle is downloaded
  (#2182, #2195, #2205).
- **Test suites were repaired, not deleted.** Four Go packages and six frontend
  test files were fixed (#2222, #2223, #2225).
- **Process changes to support parallel work.** Changelog and TODO entries moved
  to a fragment system so simultaneous changes stop colliding on the same file
  (#2161, #2163).

**Highest-risk items this month:**

- **#2209** — the shared-database defect. Every feature depending on the embedded
  database returned a server error. This is the most user-visible outage-class
  defect of the month.
- **#2179** — client-side navigation in the web interface was broken; the server
  did not serve the application shell for internal routes.
- **#2200, #2199, #2220** — three concurrency defects, including a data race in
  the asynchronous library scan. Races produce intermittent, unreproducible
  corruption and are disproportionately expensive to diagnose later.
- **#2215** — provider status was being *invented* rather than measured: the
  interface reported provider health that had not been checked.

## What changed, in plain terms

### 1. The silent-failure discovery

**What was wrong:** the web interface is a single-page application. When the
browser asks the server for an address the server does not recognise, the server
returns the application's start page rather than an error. That is correct for
page addresses — and disastrous for data addresses, because a request to a
*missing* data endpoint comes back looking successful.

The result: several features had complete, working server-side implementations
that were never connected to an address the interface could call. Nothing
errored. The pages simply did nothing.

**The fix:** the missing routes were connected (#2207, #2211, #2224), and the
page-serving behaviour was corrected (#2179).

**What it means:** this is the single most important lesson of the month, and it
changed how the work is verified. A successful HTTP response is not evidence a
feature works. Every front-end call now has to be checked against the list of
addresses the server actually answers.

### 2. The one-writer database defect

**What was wrong:** the product's default database allows a single writer per
process. The web server opened a fresh connection on each request. The second
request onward failed.

**The fix:** one shared connection per process (#2209).

**What it means:** several features that appeared individually broken shared this
single cause. It is a good example of why the reflex to fix symptoms one at a
time is expensive.

### 3. Real subtitle sources

Four sources were implemented properly rather than left as placeholders, all of
them usable without a paid account. This matters commercially: it is the
difference between a product that requires the user to already hold a paid
subtitle subscription and one that is useful out of the box.

An audit begun this month also raised a concern carried into August: a number of
the configured sources appear to be non-functional placeholders.

### 4. Speech-to-text made real

The self-hosted transcription client did not actually transcribe. It was fixed,
then given the surrounding machinery to be useful: separate connection and
transcription timeouts (a transcription legitimately takes minutes, a connection
should not), automatic fallback when no source has a subtitle, and verification
that detects a subtitle drifting out of sync with the audio.

### 5. Bazarr-parity feature batch

The bulk of the month. Post-processing scripts with access to provider and score
variables; automatic download gated on a quality threshold and automatic upgrade
when a better match appears; a blacklist that remembers why something was
rejected and for how long; history retention; an outbound proxy; a Plex webhook;
notifications via Apprise; and configurable naming for single-language output.

### 6. Fixing tests instead of deleting them

Ten test files across Go and the frontend were repaired. This is noted because
the alternative — deleting or skipping failing tests to get a green build — is
the practice that produced the blind spots this project has been paying for.

## What this means going forward

July 2026 restored momentum and delivered real capability. It also established
that reported completeness from 2025 could not be trusted, and that the failure
mode to watch for is *silence*, not errors.

Three things carried into August:

- **Language profiles were still not honoured on every download path**, despite
  having been reported complete twice in 2025. Addressed in August.
- **The provider registry likely contains non-functional placeholder entries**,
  each consuming a slot in every search.
- **Nothing in the web interface had been verified in a real browser.** Every
  claim about the interface to this point rested on automated tests, which the
  following month proved could pass while the feature was entirely broken.
