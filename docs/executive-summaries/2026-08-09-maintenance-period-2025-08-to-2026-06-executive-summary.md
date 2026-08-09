<!-- file: docs/executive-summaries/2026-08-09-maintenance-period-2025-08-to-2026-06-executive-summary.md -->
<!-- version: 1.0.0 -->
<!-- guid: 6e04b217-3f8d-49ca-b165-902a7c4e83df -->
<!-- last-edited: 2026-08-09 -->

# Executive Summary: August 2025 – June 2026 Maintenance Period

**Shipped:** [340 merged pull requests](https://github.com/falkcorp/subtitle-manager/pulls?q=is%3Apr+is%3Amerged+merged%3A2025-08-01..2026-06-30),
covering 2025-08-01 through 2026-06-30 (eleven months).
**Related doc:** [2026-08-09-july-2025-roundup-executive-summary.md](2026-08-09-july-2025-roundup-executive-summary.md)
**Note:** Written 2026-08-09 as part of a backfill; reconstructed from the merge
record.

**Read this line before the rest: of the 340 pull requests merged in these eleven
months, 296 were automated dependency updates. Forty-four were substantive, and
half of those landed on two days in January.**

This document is short because the period was quiet. Inflating eleven months of
automated version bumps into eleven months of prose would misrepresent what the
project cost and what it delivered, so it is not done here.

## Executive Summary

- **Active development stopped at the end of August 2025 and did not resume until
  July 2026.** Between September 2025 and June 2026 there were fewer than twenty
  substantive changes in total.
- **Automated dependency maintenance continued throughout, unattended.** 296
  dependency updates were merged across the period — Go libraries, frontend
  packages, GitHub Actions, and Docker base images. This is why the project could
  be picked back up in July 2026 without first spending weeks on a dependency
  backlog.
- **Two genuine supply-chain security events were handled.** A compromised
  scanning tool was removed from the pipeline (#2137), and every GitHub Action
  was pinned to an exact commit rather than a moving tag (#2141).
- **A concentrated test-coverage push in January 2026.** Twenty-four pull
  requests over two days (#2045–#2073) added unit tests across configuration,
  logging, monitoring, storage, webhooks, the job queue, internationalisation,
  and four subtitle providers.
- **The internal library migration finished.** The move onto shared "gcommon"
  components that dominated the second half of July 2025 was completed in late
  August 2025 (#1848, #1850).
- **Organisational and standards moves.** The project's references were migrated
  from a personal account to the `falkcorp` organisation (#2153), and shared
  coding standards were centralised into a submodule (#2156).

**Highest-risk items this period:**

- **#2137 — a compromised supply chain.** The Trivy security scanner was removed
  from the build pipeline after the tool itself was compromised. A scanner runs
  with access to the codebase and the build environment, so a compromised scanner
  is a direct route in. Removed rather than pinned.
- **#2141 — every GitHub Action pinned to a commit hash.** Previously the build
  referenced third-party automation by moving tags such as `v3`. Anyone able to
  move that tag could have changed what ran in the build. Now each is pinned to
  an exact, immutable commit.

## What changed, in plain terms

### 1. The project went dormant — and that was survivable

From September 2025 through June 2026, ten months, the substantive change list is
short enough to read in full: one editor configuration commit, one repository
structure sync, twenty-four test files, one scanner removal, one action-pinning
change, two workflow additions, an organisation rename, and a standards
submodule.

What kept running was the automated dependency machinery. Nearly 300 updates
merged without human intervention. The practical consequence is visible in July
2026: work resumed on a codebase whose dependencies were current, rather than one
needing a large and risky catch-up before anything else could start.

That is the argument for the cost of maintaining automated dependency updates on
a dormant project — it is insurance against the resumption cost, and in this case
it paid.

### 2. Two supply-chain security events

**What was wrong:** the build pipeline ran a third-party security scanner, and
that scanner's own distribution was compromised. Separately, the build referenced
third-party automation by moving version tags, which anyone controlling those
tags could repoint.

**The fix:** the compromised scanner was removed outright rather than pinned to
an older version (#2137). Every action reference was replaced with an exact
commit hash (#2141).

**What it means:** these are the two classic ways an attacker gets code into a
build without touching the repository. Both are closed. The cost is that
upgrading an action is now a deliberate act rather than automatic — which is the
intended trade.

### 3. The January test-coverage push

Twenty-four pull requests over 18–19 January 2026 added unit tests to areas that
had none: configuration parsing, logging helpers, the job queue, storage
workflows, webhook validation, subtitle renaming, database migration,
internationalisation, and the fetch paths of four subtitle providers.

This was worthwhile but should be read with a caveat that later work made
unavoidable: **test coverage is not the same as test value.** In August 2026 a
set of user-interface tests was found to be passing while the feature they
covered was completely broken, because the test fixtures described data the
server never actually sends. Coverage counts lines executed, not assumptions
verified.

### 4. Finishing the internal library migration

The shared-component migration begun in July 2025 was completed in late August
2025, replacing local implementations with shared types and the database
protobuf definitions (#1848, #1850). As with the July work, this produced no
user-visible change.

## What this means going forward

The period delivered no product capability, and the document should not pretend
otherwise. What it delivered was continuity: a codebase that stayed current,
two closed security exposures, and a test suite that was broader in June 2026
than in August 2025.

Two things to carry forward:

- **The dependency automation earned its keep.** It is the reason the July 2026
  restart was possible at low cost.
- **The January coverage work needs revisiting, not extending.** Adding more
  tests of the same kind is not obviously valuable while fixtures can disagree
  with the real server and still pass. That lesson was learned expensively in
  August 2026.
