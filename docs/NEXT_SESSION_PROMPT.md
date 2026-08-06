<!-- file: docs/NEXT_SESSION_PROMPT.md -->
<!-- version: 3.0.0 -->
<!-- guid: 3a7f21c8-64b9-4e05-9d3a-8f1e07b26c54 -->
<!-- last-edited: 2026-08-05 -->

# Next-session prompt

Paste the block between the rules into a fresh Claude Code session in the
`falkcorp/subtitle-manager` repo. It is written to be self-contained — it does
not assume any memory of the session that produced it.

Keep this file updated at the end of each session rather than writing a new one.

---

Continue the **subtitle-manager Bazarr feature-parity** work. The standing goal
is unchanged: *get a fully working subtitle manager with feature parity with
Bazarr that I can evaluate; document any decisions so I can review them later.*

**You have my standing permission to merge anything directly related to this
project into my repos, as long as it has been tested somehow.** Don't stop to
ask. Rules that still apply: rebase-merge only (`gh pr merge <N> --rebase
--admin` — linear-history ruleset), force-push still needs my explicit
confirmation each time, and no bulk/mass label operations. After merging, check
the merged commits actually cover every commit that was in the PR — a PR
silently lost two fix commits that way once.

## Where things stand (verified 2026-08-05)

`origin/main` is green and **releases publish again**. Verified directly, not
inferred: `go build ./...`, `go test ./...`, `go test -tags sqlite ./...`
(**CI does not build this tag — run it locally**), `go test -race` on the
packages whose tests mutate globals, the 64-test frontend suite, and a
`GOOS=windows` cross-build. CI now cross-compiles all four goreleaser targets.

**Downloading works today with no credentials.** Verified end to end on a Pebble
store: a two-language profile produced `...en.srt` (476 cues) and `...es.srt`
(523 cues) from `gestdown` via `fetch --profile`, correctly named and recorded
in download history.

Read `docs/BAZARR_PARITY_STATUS.md` and `docs/PROVIDER_BUILDOUT_PROMPT.md`.
**Verify claims in both against the code before acting** — several have been
wrong before, though both were corrected on 2026-08-04.

## What to pick up, in priority order

### 1. Verify the web UI in an actual browser — nothing here is proven

This is the biggest gap in confidence, not necessarily in code. The last session
exercised **API endpoints with curl only**. Every one returned sensible JSON, and
API routes 401 rather than falling through to the SPA — but **no page was ever
rendered or clicked**.

There are 20 pages (`webui/src/*.jsx`): Dashboard, MediaLibrary, LibraryScan,
MediaDetails, Wanted, History, Settings, Setup, System, UserManagement,
TagManagement, Scheduling, Convert, Extract, Translate, Verify, ConfigEditor…

Confirm by using it:

- **Does library scanning work from the UI?** `/api/scan`, `/api/scan/status`,
  `/api/library/scan`, `/api/library/scan/status` and `/api/library/rescan` are
  all mounted. Does the UI drive them, show progress, and produce real results?
- **Is there actually a file on disk afterwards?** The recurring bug class in
  this repo is a write nothing reads, or a UI reporting success while nothing
  happened. Check the filesystem, not the toast.
- **Does Mass Edit work end to end in the browser?** The bulk profile endpoint
  and its UI shipped in #2238; the UI has unit tests but has never been driven
  by a human or by browser automation.

Use the Chrome tools if available. A running instance for evaluation lives at
`~/subtitle-manager-preview/` (`sh start.sh`).

### 2. Decide what "multiple subtitles, multiplexed" means

The operator asked: *"it should grab multiple but then it has to multiplex them
together — are we doing that?"* **Ask which of these is wanted** rather than
guessing; they are three different features:

- **Separate files per language** — works today, verified above
  (`video.en.srt`, `video.es.srt`). Note `singleLanguageNaming()` collapses these
  to one filename, which would make a multi-language profile overwrite itself;
  `assignedProfileLanguages` currently truncates to the top language when that
  setting is on.
- **One subtitle file containing two languages** — `cmd/dualsub.go` already does
  this (bilingual "double-subs" tagged as a sentinel language). It exists as a
  CLI command only; it is **not** wired into the profile/scan path and has no UI.
- **Muxing tracks into the MKV container** — nothing does this. Needs
  mkvmerge/ffmpeg and is a much larger piece of work.

My read: the profile path already produces multiple files correctly, so the real
ask is probably dualsub-on-download or container muxing. Get that decided first.

### 3. OpenSubtitles needs an API key — and it CANNOT be harvested

**Verified 2026-08-05, so do not spend time re-checking:** an OpenSubtitles API
key is mandatory. A live login against the v1 API with real credentials returned
**403 with no `Api-Key` header** and 429 with an empty one.

There is **no key to pull from Bazarr**. Bazarr's config carries only
`opensubtitlescom.username` / `.password`; it embeds its own API key in its
source rather than storing one per install. The *arr configs do hold 32-char
keys, but those are Sonarr's and Radarr's own API keys, not OpenSubtitles'. All
were checked.

So this is genuinely operator-blocked: a key must be registered at
opensubtitles.com. Everything else on that provider is already fixed — the v1
download handshake, the `file_id` source, the `Api-Key` header, and reading both
Bazarr credential spellings.

**Ask the operator for the key rather than searching for it again.** When wiring
providers: only handle config key *names* and protocol, never the secret values,
and use httptest mocks.

### 4. Cloudflare Access JWT verification + broader OAuth

**GitHub OAuth already exists** (`/api/oauth/github/login`, `/callback`,
`/generate`, `/regenerate`, `/reset`, plus `pkg/authserver`).

**Cloudflare is entirely absent** — zero matches for `cloudflare`, `CF-Access`
or `jwt` anywhere in `pkg/`. This is greenfield:

- Accept and **verify** the `Cf-Access-Jwt-Assertion` header against the team's
  JWKS endpoint (fetch and cache the keys; verify `aud`, `iss`, `exp`).
- Map a verified identity onto a subtitle-manager user and role.
- **Fail closed.** The obvious mistake is trusting the header when verification
  is unconfigured or the JWKS fetch fails — that lets anyone authenticate by
  setting a header. Make it a hard refusal and test that path explicitly.
- Decide whether it replaces or supplements the session cookie.

Ask whether other providers (Google, generic OIDC) are wanted too.

### 5. ZFS-aware tuning

The deployment host uses ZFS and nothing in the repo acknowledges that. Worth
investigating, but **measure before changing anything** — every item below is a
hypothesis, not a known win:

- Pebble/SQLite on ZFS: record size vs page size, and whether double
  write-caching (ZFS ARC plus the DB's own) costs anything.
- `sync` behaviour: Pebble writes with `pebble.Sync`, which interacts with the
  ZIL/SLOG.
- Whether large media scans benefit from different readahead.
- ZFS snapshots as the backup mechanism instead of the app's own backup code.

### 6. Bazarr API compatibility layer (operator's idea — lower priority)

*"Is it possible to have a Bazarr API compatibility layer so that applications
that don't know how to use our app can just drop it in?"*

Yes, and it is a good idea for adoption. Note that what exists today is **config
import only** — `/api/setup/bazarr`, `/api/setup/bazarr/upload`,
`/api/bazarr/preview`, `/api/bazarr/config` read a Bazarr config. There is no
Bazarr-shaped API surface.

Scope it before building. The useful subset is probably what the *arr ecosystem
actually calls — `/api/system/status`, `/api/episodes`, `/api/movies`,
`/api/providers`, `/api/history`, and the webhook endpoints. A faithful clone of
all of Bazarr's API is far bigger than a shim for the handful of routes other
tools use. Start by finding what actually calls Bazarr's API in the operator's
stack.

### 7. Still open from earlier sessions

- **22 of 51 provider hostnames do not resolve** — fabricated `api.<name>.com`
  shells that take a slot in every fetch wave. Data and suggested approach are in
  `PROVIDER_BUILDOUT_PROMPT.md`. Audit the *registry*, not the hostnames.
- **Two CodeQL "uncontrolled data in path expression" alerts** on
  `pkg/webserver/subtitlepath.go`. Believed safe, pinned by
  `pkg/security/formatpath_test.go`. Dismiss them or don't — operator's call.
- **`pkg/webserver` profile tests default to SQLite while the product defaults
  to Pebble.** That gap has now hidden two real bugs. Running the profile suite
  against both backends is worthwhile structural work.

## How I want you to work

- One focused PR per item, off a worktree from **`origin/main`**. Each needs a
  `changelog.d/` fragment and **mandatory** file version headers
  (`file:` / `version:` / `guid:` / `last-edited:`, bumped on every change).
- `rm -rf .standards` in a new worktree and use `git -c submodule.recurse=false`.
- **Prove every test is non-vacuous**: break the fix, confirm the matching test
  fails, restore. This has caught roughly a dozen tests that passed for the wrong
  reason.
- **Never chain verification and publication in one command.** A
  `go test && git push` chain pushed a commit with a failing security test on
  2026-08-04. Run the suites, read the result, *then* push.
- **Don't trust a passing test as evidence the feature works.** Six real bugs on
  2026-08-04 were invisible to a green suite and appeared only when the binary
  was run — including one in code shipped earlier the same day. Run the thing.
- **Fix the product, not the test.** A test was patched to work around a symlink
  mismatch that turned out to be a real defect in path validation; that hid it
  for hours.
- **When you verify live, confirm which process answered** — check for a stale
  listener before starting anything.

### Traps that cost real time on 2026-08-04

- A PR branched off another *unmerged* branch goes `CONFLICTING` once that branch
  rebase-merges, and **GitHub cannot build a merge ref for a conflicting PR, so
  CI never queues at all** — it looks like a broken trigger. Check
  `gh pr view N --json mergeable,mergeStateStatus` first.
- `workflow_dispatch` on `ci.yml` **skips Go CI** (`Detect Changes` has no diff),
  so it is not a substitute signal. `Go CI` also correctly shows `skipping` on
  docs-only PRs.
- Every action is **SHA-pinned**; a tag ref fails at "Set up job" with no useful
  message.
- The 0-second `Code Quality Check` / `AI Labeling` failures are stale
  concurrency-cancelled jobs. The real gates are **`Go CI (1.25)`** and
  **`Analyze (go)`**.
