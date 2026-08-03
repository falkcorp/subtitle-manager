<!-- file: docs/NEXT_SESSION_PROMPT.md -->
<!-- version: 1.2.0 -->
<!-- guid: 3a7f21c8-64b9-4e05-9d3a-8f1e07b26c54 -->
<!-- last-edited: 2026-08-02 -->

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

## Where things stand

`origin/main` is green: `go build ./...`, `go test -race ./...`,
`go test -tags sqlite ./...` (**CI does not build this tag — run it locally,
it has caught post-merge breakage twice**), and the frontend suite.
A running instance for evaluation lives at `~/subtitle-manager-preview/`
(`sh start.sh` → http://127.0.0.1:8080, `admin` / `evaluate-me-2026`).

Read `docs/BAZARR_PARITY_STATUS.md` for the full matrix and
`docs/PROVIDER_BUILDOUT_PROMPT.md` for the provider plan. **Verify claims in
both against the code before acting on them** — several have been wrong, most
recently three ✅ rows claiming language profiles were wired to downloads when
nothing read them.

## What the last session (2026-08-01) did

Started on mass-edit, found the underlying feature was broken, and fixed that
instead. Three PRs, all merged:

- **#2230** — per-item profile assignment collided on one key. The UI
  percent-encodes the media path into the URL; `net/http` decodes it before the
  handler runs, so the handler's first-segment extraction made every file under
  `/media` share the key `media`. New `pathRest` helper + `filepath.Clean`
  normalisation.
- **#2231** — corrected three parity rows and recorded the findings.
- **#2232** — library scans now honour an assigned language profile, resolved
  through the same store and key the UI writes, downloading every language in
  priority order. Also fixed `pkg/webserver/scan.go` opening its own store with
  `database.OpenStore` — Pebble's exclusive lock made that fail silently,
  leaving `store` nil, which would have made the whole feature a no-op in
  production while every test passed.

## What to pick up, roughly in order

1. **Mass-edit / bulk profile assign** (`BAZARR_PARITY_STATUS.md`, 🔴). This is
   what the last session was originally sent to do and is now unblocked —
   assignments finally mean something. Self-contained and user-visible.
   Suggested shape: `POST /api/media/profiles/bulk` registered as its own exact
   route (not string-matched inside `mediaProfilesHandler`), `basic` auth to
   match the single-item route, validate `profile_id` once then return a
   per-item results array so the UI can say "7 of 9 assigned". Note
   `MediaLibrary.jsx` already posts to `/api/bulk-operation`, which is not
   mounted — decide whether to remove that dead call or implement it, but don't
   hang the new profile action off its error handling, which swallows failures
   into `console.error`.

2. **Profiles are still unwired outside library scans.** Monitor loop, watcher,
   webhooks, scheduler and *arr all call `scanner.ProcessFile` with a single
   language. Wiring them needs a decision first: `MonitoredItem.Languages` is a
   second, independent desired-languages mechanism, and something has to win.
   Ask me rather than picking.

3. **Three known bugs, none fixed.**
   - A file explicitly assigned the *default* profile is indistinguishable from
     an unassigned one, so it gets the scan's language rather than that
     profile's list. Needs a store method reporting whether an assignment row
     exists, separate from which profile governs the file (interface + 3
     implementations + mocks).
   - The last remaining language profile cannot be deleted:
     `GetDefaultLanguageProfile` returns the first profile when none is flagged,
     and `handleDeleteProfile` refuses to delete the default.
   - `cmd/profiles.go` hardcodes `database.OpenStore(..., "pebble")`, ignoring
     `db_backend`.

4. **Whisper: RESOLVED 2026-08-02, no config change needed.** The transcription
   fallback works end-to-end; leave `whisper.transcribe_url` alone.

   The GPU host runs a custom FastAPI Whisper server that previously served only
   `/transcribe` and discarded segment timings, so `WhisperTranscribe` — which
   speaks the native `/asr` protocol — got a 404. That server now also serves
   `/asr`, returning real SRT/VTT built from `segment.start`/`.end`. Verified
   against a real media file: 196 monotonic cues over 10 minutes, driven through
   the `transcribe` CLI rather than curl.

   Two constraints worth knowing before touching it: the GPU has no room for a
   second resident model, so one process serves both this project and another;
   and inference is therefore serialised behind a lock, meaning a long
   transcription blocks the other project's requests for its duration. The
   deployment is **not** in this repo — the operator's notes have the details.
   Do not record host addresses here; this file is public.

5. **Two findings from an earlier session, still open.**
   - `security.ValidateAndSanitizePath` accepts **any** absolute path under
     `os.TempDir()`, with no build tag and no config gate, voiding
     `allowed_base_dirs` in production. Several tests rely on it, so it needs
     its own PR plus test fallout work.
   - Two open CodeQL "uncontrolled data in path expression" alerts on
     `pkg/webserver/subtitlepath.go`. Believed safe; both constraints pinned
     with tests in `pkg/security/formatpath_test.go`. Dismiss them or don't.

6. **Providers are blocked, not merely unfinished.** Phase 1 (keyless) is
   genuinely exhausted — eight candidates probed live on 2026-07-31, results
   tabulated in `PROVIDER_BUILDOUT_PROMPT.md`. Phase 2 needs credentials in my
   local `~/.subtitle-manager.yaml`; **only wire config key names and protocol,
   never handle the secret values, and use httptest mocks.** Ask me which
   providers I've configured rather than guessing.

## How I want you to work

- One focused PR per item, off a worktree from `origin/main`. Each needs a
  `changelog.d/` fragment and **mandatory** file version headers
  (`file:` / `version:` / `guid:` / `last-edited:`, in that order — bump on
  every change, including files you only touched lightly).
- `rm -rf .standards` in a new worktree and use `git -c submodule.recurse=false`
  — the submodule otherwise breaks git operations there.
- **Prove every test is non-vacuous**: break the fix, confirm the matching test
  fails, restore. This has caught roughly a dozen tests that passed for the
  wrong reason.
- **Don't trust a passing test as evidence the feature works.** Nearly every
  real bug here was invisible until something was actually run. The 2026-08-01
  session is the case in point twice over: a green suite would have shipped a
  feature that did nothing in production, and a "live verification" that
  appeared to disprove a working fix turned out to have been answered by an
  orphaned binary holding the port. **When you verify live, confirm which
  process answered** — `lsof -nP -iTCP:<port> -sTCP:LISTEN -t` against your own
  PID, and check for a stale listener before starting anything.
- Document decisions in the PR body and the parity doc as you go, so I can
  review and disagree later.

---
