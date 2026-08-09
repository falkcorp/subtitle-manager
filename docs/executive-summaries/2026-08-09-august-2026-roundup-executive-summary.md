<!-- file: docs/executive-summaries/2026-08-09-august-2026-roundup-executive-summary.md -->
<!-- version: 1.0.0 -->
<!-- guid: 2d95c1e6-47ab-4f30-b8e2-19c630fa5487 -->
<!-- last-edited: 2026-08-09 -->

# Executive Summary: August 2026 Roundup (month to date)

**Shipped:** [26 merged pull requests](https://github.com/falkcorp/subtitle-manager/pulls?q=is%3Apr+is%3Amerged+merged%3A2026-08-01..2026-08-09),
covering 2026-08-01 through 2026-08-09.
**Related doc:** [2026-08-09-july-2026-roundup-executive-summary.md](2026-08-09-july-2026-roundup-executive-summary.md)
**Status:** This covers a partial month and will be updated in place as August
continues.

Nine days, and the shape of the month is different from July's: fewer new
features, and a great deal of finding out that things reported as working did
not. The most valuable output of these nine days is arguably not the code — it is
the first evidence-based inventory of what is actually broken.

## Executive Summary

- **Language profiles finally reach every download path.** The feature reported
  complete in June 2025, again in July 2025, and partially wired in July 2026 is
  now honoured by library scans, automated downloads, and the command line, with
  quality cutoffs and forced/hearing-impaired preferences reaching the scoring
  system (#2232, #2242, #2247).
- **Bulk assignment of language profiles.** An operator can now select many files
  and assign a profile in one action rather than one at a time (#2238).
- **OpenSubtitles was substantially rebuilt** — the download handshake, the file
  identifier, the authentication header, and the two different configuration
  spellings a Bazarr import can produce (#2248, #2250, #2252). **It still cannot
  be used**; see below.
- **Two security fixes.** A temporary-directory escape hatch that voided the
  configured folder restrictions in production was closed (#2241), and
  media directories reached through a symbolic link are now handled correctly
  rather than rejected (#2255).
- **Every Windows release build had been broken since early August 2025** by a
  platform-specific system call. Fixed, and the build pipeline now compiles all
  four target platforms so it cannot silently regress again (#2243).
- **The web interface was driven in a real browser for the first time.** Library
  scanning and bulk editing were confirmed working end to end — verified by
  checking files on disk and reading assignments back from the server, not by
  trusting on-screen confirmations.
- **That browser session found four defects that a fully passing test suite had
  missed**, one of which meant the Media Library page rendered completely empty.
  **This work was subsequently lost before being committed and has not yet been
  redone.**
- **Sonarr/Radarr configuration was unified.** Two competing configuration
  schemes were consolidated into one (#2246).
- **Failures now say why.** A failed search reported "no subtitle found"
  regardless of cause — including authentication failures. It now reports the
  actual reason (#2251).

**Highest-risk items this period:**

- **#2241** — the temporary-directory bypass. The setting that restricts which
  folders the product may touch could be circumvented in production. Now gated to
  test runs only.
- **#2243** — Windows releases had been failing to build for a full year without
  being noticed.
- **#2255, and its follow-up correction** — symbolic-link handling in the
  path-security check. The first attempt resolved links in the wrong place; the
  correction resolves them only for the comparison.

## What changed, in plain terms

### 1. Language profiles, finally connected

**What was wrong:** an operator could define "for this show I want English and
Spanish, with forced subtitles preferred", see it saved, see it displayed — and
have it ignored at the moment a subtitle was actually chosen.

**The fix:** the profile is now consulted on every automated path, and its
quality cutoff and forced/hearing-impaired preferences reach the scoring system
that ranks candidates.

**What it means:** this is the third time this feature has been reported
delivered. It is the first time it has been demonstrated working through the
product's own download paths.

### 2. The browser session — and the lost work

The web interface had never been driven by a human or a browser; every claim
about it rested on automated tests. Nine days in, it was.

Library scanning worked: real subtitle files appeared on disk, byte-identical to
a known-good reference. Bulk profile assignment worked, confirmed by reading the
assignment back from the server.

Getting that far required fixing four defects in the Media Library page, the
worst of which caused it to render **completely empty** — no items, no message —
because the page looked for a field name the server does not send. The bulk-edit
feature was unreachable behind it.

**All four fixes, and the tests written for them, were lost before being
committed.** They were held in a temporary workspace that was cleared between
sessions. The defects are fully described in the project's working notes and can
be redone in one pass, but as of this document **the Media Library page is still
broken in the shipped code.**

This is a process failure, not a technical one, and it has been recorded as such
so it is not repeated.

### 3. Why a passing test suite proved nothing

All four Media Library defects survived a fully passing test suite. The
assertions were fine; the **fixtures were wrong** — the tests fed the page a data
shape the real server never produces, so the test and the page agreed with each
other while both disagreed with production.

A second trap compounded it: the browser framework reports an error thrown inside
a click handler in a way the test runner does not count as a failure. A test
written to catch one of these defects passed even with the defect deliberately
reinstated.

**What it means:** test fixtures must be built from observed server responses,
and tests must assert an observable effect rather than the appearance of an error
message. This is the single most transferable lesson of the month.

### 4. OpenSubtitles — rebuilt, and still unusable

The largest subtitle source in the world was substantially repaired this month.
It still cannot be used, and the reason is not technical.

Access requires an API key registered with the service. Investigation
established, with live testing, that:

- No usable key exists anywhere in the operator's existing systems.
- Bazarr — the incumbent tool — ships its own registered key compiled into the
  application, which is why it never asks a user for one. (Its source was read
  directly to establish this. Note that on the operator's own installation
  Bazarr's OpenSubtitles source is not switched on, so this is what its code
  does, not a demonstration of it working there.) Copying that key would mean
  impersonating another product and sharing its rate limit; it was deliberately
  not done.
- A key supplied on 2026-08-06 does not work. Tested against the live service, it
  behaves identically to a deliberately invalid key and to sending no key at all.
- The operator's paid "VIP" status is on the **legacy** OpenSubtitles system,
  which is separate from the current one and does not grant access to it.

**What it means:** this is blocked on a business action — registering an API
consumer with OpenSubtitles — not on engineering. No amount of further
development will unblock it.

### 5. A year of broken Windows releases

A platform-specific system call for reading disk statistics was compiled on all
platforms. It does not exist on Windows. Every Windows release build had failed
since early August 2025 and nobody had noticed, because the release process did
not attempt all platforms.

Fixed, and the build now compiles all four targets on every change so the same
class of failure surfaces immediately.

## What this means going forward

Nine days produced real progress on language profiles and real security fixes.
But the honest headline is that this period was mostly about discovering the gap
between what the record claimed and what the product does.

**Known broken or unproven as of 2026-08-09:**

- **The Media Library page is broken in shipped code.** The fixes exist only as a
  written description; redoing them is the highest-priority outstanding item.
- **No subtitle source can be enabled from the web interface.** The settings page
  calls two addresses that are not connected, and it fails silently — the dialog
  closes as though the change was saved. Only one source ships enabled by default,
  and that one requires additional software and a container to function. Out of the
  box, via the interface alone, there is no route to a working subtitle source.
- **OpenSubtitles is blocked on obtaining a registered API key.**
- **It is unverified whether the published release binaries can run the web
  server at all.** The web server requires a build option that the documented
  multi-platform build target does not pass. This is unconfirmed rather than
  known-broken, and it outranks everything else if it turns out to be true.
- **User administration is broken on the default database.** The command-line
  user commands compute the authentication database location differently from the
  web server, so they fail on a default deployment.
- **Roughly 22 of 51 configured subtitle sources appear to be non-functional
  placeholders**, each occupying a slot in every search.

**Where the money went, and what it bought.** These nine days bought one feature
completed properly after three prior claims of completion, two closed security
holes, a year-old release failure fixed, and — most valuably — a verified list of
what does not work, replacing a documentation record that had been wrong in at
least six places. A project that knows what is broken is in a materially better
position than one that believes it is finished.
