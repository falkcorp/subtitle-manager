<!-- file: docs/executive-summaries/2026-08-09-june-2025-roundup-executive-summary.md -->
<!-- version: 1.0.0 -->
<!-- guid: 8f2c40d1-6b93-4a57-9e08-3d17c5ab2e64 -->
<!-- last-edited: 2026-08-09 -->

# Executive Summary: June 2025 Roundup

**Shipped:** [270 merged pull requests](https://github.com/falkcorp/subtitle-manager/pulls?q=is%3Apr+is%3Amerged+merged%3A2025-06-01..2025-06-30),
covering 2025-06-01 through 2025-06-30 (774 commits).
**Note:** This document was written on 2026-08-09 as part of a backfill of the
project's history. It reconstructs the month from the merge record rather than
from contemporaneous notes.

This is the month the product was built. It is by a wide margin the largest month
in the project's history — more merged work than the following eleven months
combined. Everything a subtitle manager needs in order to exist at all went in
here: the thing that finds subtitles, the thing that stores them, the website you
operate it from, and the locks on the door.

## Executive Summary

- **The product came into existence.** A command-line tool and a full web
  interface were built from an empty repository, backed by an embedded database
  (storage that ships inside the application, with no separate database server to
  install or maintain).
- **Subtitle sources — the core of the product.** The system that searches
  multiple subtitle websites and picks the best result was built, including
  support for running several accounts against the same source, priority ordering
  between sources, and a searchable tag system for organising them.
- **Language profiles.** The feature that lets an operator say "for this show I
  want English and Spanish, and I care about forced and hearing-impaired
  variants" was designed, given a database, an API, a management screen, and a
  migration tool to carry existing settings forward.
- **Integration with the wider media stack.** Periodic synchronisation with
  Sonarr and Radarr (the tools that manage TV and film libraries), plus an
  importer that reads an existing Bazarr configuration so a user switching over
  does not start from nothing.
- **Access control and login.** Role-based permissions, GitHub single-sign-on,
  one-time login tokens, session management with automatic cleanup, and a
  user-administration screen (#37, #154, #219, #838).
- **A recurring class of security defect was found and closed repeatedly.** Five
  separate "path injection" holes — where a crafted request could reach files
  outside the intended folders — were fixed, and the fix was eventually
  centralised rather than patched case by case (#1071, #1113, #1143, #1144,
  #1279, #1280).
- **Speech-to-text and translation.** Configurable Whisper support (transcribing
  spoken audio into subtitles when none exist to download) and automatic
  translation during synchronisation.
- **Programmatic access.** A published API specification plus three
  ready-to-use client libraries (Go, JavaScript, Python) and a configurable gRPC
  server, so other systems can drive the product without going through the web
  interface.
- **Delivery and operations.** Docker images, automatic self-update, scheduled
  maintenance tasks, configuration backup and restore, and logging to both file
  and console.
- **Heavy investment in development automation.** A substantial share of the
  month went into workflow tooling — issue management, pull-request automation,
  code formatting, and continuous-integration pipelines — rather than into
  product features. This is real cost and is called out honestly in the detail
  below.

**Highest-risk items this month** — the ones a stakeholder most needs to know
about, because each one was a way for an outsider to reach data they should not:

- **#1279, #1280** — the folder-browsing feature of the web interface could be
  manipulated to read directories outside the media library. Both closed.
- **#1071, #1142, #1143, #1144** — four further path-handling holes flagged by
  automated security scanning, in the scanner, the batch operations, and the
  Bazarr import URL. All closed.
- **#892** — the web interface was serving pages without standard browser
  security headers and without sanitising user-supplied HTML, leaving it open to
  script-injection. Headers and sanitisation added.
- **#37** — until this landed, there was no authentication at all.

## What changed, in plain terms

### 1. The product came into existence

At the start of the month the repository contained an initial commit and little
else. By the end of it there was a working command-line tool, a web interface
with roughly twenty screens, an embedded database, a Docker image, and a
published API.

Two storage engines were supported deliberately: **Pebble**, an embedded
key-value store that requires no installation, and **SQLite**, a single-file
database that is easier to inspect. Pebble became the default. This dual-backend
decision is worth remembering — it has since been the source of several defects
where behaviour differed between the two, and it remains a live source of risk.

### 2. Subtitle sources

The core value of the product is that it searches many subtitle websites at once
and chooses the best match. That machinery was built here: a provider registry,
the ability to configure multiple accounts against the same website with an
explicit priority order, backoff so a failing source does not stall the others,
and a universal tagging system so sources can be grouped and filtered.

The web interface gained provider configuration screens, a card-based layout, and
click-through configuration from the dashboard.

### 3. Language profiles

A language profile answers "which subtitles do I actually want for this item".
The month delivered the underlying data model, an API, a management screen, and —
importantly — a migration tool so that operators who had already configured
per-language settings were carried forward rather than reset.

This feature was *not* finished in the sense that mattered: as later months show,
profiles were stored and displayed but were not consistently consulted at
download time. That gap survived until August 2026.

### 4. Media-stack integration

Periodic synchronisation with Sonarr and Radarr was added, along with audio and
subtitle track selection and translation during sync. A Bazarr importer was
built so that a user migrating from the incumbent tool could bring their existing
configuration across — including provider credentials.

### 5. Access control and login

The project began with no authentication whatsoever. Over the month it gained
role-based access control, GitHub single-sign-on, one-time login tokens,
credential generation and reset endpoints, periodic session cleanup, and a user
administration UI.

### 6. The path-injection pattern

The single most repeated defect class of the month deserves its own note, because
it is the clearest example of cost that could have been avoided.

**What was wrong:** the product's job is to read and write files in folders the
operator nominates. Several features accepted a file or folder name from the
network and used it directly. A crafted request could therefore walk out of the
media library and read elsewhere on the machine.

**The fix:** it was patched five times in five places (#1071, #1142, #1143,
#1144, #1279, #1280) before being consolidated into shared security utilities
(#1113) plus an explicit allow-list of permitted base directories. Consolidating
first would have been cheaper than fixing it six times.

**What it means:** the class is closed and there is now one place to audit, but
this is the origin of a validation layer that has needed correction as recently
as August 2026.

### 7. Speech-to-text, translation, and metadata

Configurable Whisper support was added for generating subtitles from audio when
none can be downloaded, together with translation during synchronisation and a
multi-method synchroniser. Metadata enrichment arrived via TMDB and OMDb, with a
manual metadata editor for cases the automatic lookup gets wrong.

### 8. Programmatic access

A full OpenAPI specification was published alongside generated client libraries
for Go, JavaScript, and Python, plus a configurable gRPC server. This is how
other systems drive the product without a browser.

### 9. Development automation — a large, honest cost

A significant fraction of the month's 270 pull requests went into tooling rather
than product: an issue-management system, pull-request automation, automatic code
formatting, continuous-integration pipelines, and repeated iterations on all of
the above.

This was genuine expenditure and it is visible in the record. Some of it paid
for itself in later months; some of it — the bespoke issue-update system in
particular — was later retired. It is recorded here rather than quietly folded
into the feature themes.

## What this means going forward

By the end of June 2025 the product existed end to end and could be run. What it
did not have was proof that each part worked against real services — a gap that
took until 2026 to start closing seriously.

Three things from this month have had long tails:

- **The dual-storage decision** (Pebble and SQLite) has repeatedly produced
  defects that appear in one backend and not the other, because the automated
  tests default to SQLite while the product defaults to Pebble.
- **Language profiles** were built but not fully wired into the download path.
  That was not discovered and fixed until August 2026.
- **Path validation** was consolidated but has needed correction since, most
  recently in August 2026.

The month's headline number — 270 merged pull requests — should be read with the
knowledge that a substantial share was development tooling rather than
user-visible capability.
