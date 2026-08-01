<!-- file: docs/NEXT_SESSION_PROMPT.md -->
<!-- version: 1.0.0 -->
<!-- guid: 3a7f21c8-64b9-4e05-9d3a-8f1e07b26c54 -->
<!-- last-edited: 2026-07-31 -->

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
it has caught post-merge breakage twice**), and the frontend suite
(25 files / 59 tests). A running instance for evaluation lives at
`~/subtitle-manager-preview/` (`sh start.sh` → http://127.0.0.1:8080,
`admin` / `evaluate-me-2026`).

Read `docs/BAZARR_PARITY_STATUS.md` for the full matrix and
`docs/PROVIDER_BUILDOUT_PROMPT.md` for the provider plan. **Verify claims in
both against the code before acting on them** — several have been wrong.

## What to pick up, roughly in order

1. **Mass-edit / bulk profile assign** (`BAZARR_PARITY_STATUS.md` line ~106,
   🔴). Currently single-item only. Self-contained, user-visible in the UI, and
   directly evaluable — probably the best value per unit of risk.

2. **Surface `subtitles.format` and `subtitles.single_language` in Settings.**
   Both work but are config-file only, so neither is discoverable in the UI.
   ⚠️ Read the `project_subtitle_manager_notifications_config` memory first:
   `POST /api/config` `viper.Set()`s keys verbatim, and a Settings tab once
   saved flat keys that the runtime never read, so saving silently did nothing.
   **Verify both halves — that the UI writes the key the runtime reads.**

3. **Two findings I raised but deliberately did not fix.** Both are on merged
   PR #2228; decide what you want done:
   - `security.ValidateAndSanitizePath` (`pkg/security/security.go`, the
     `os.TempDir()` branch) accepts **any** absolute path under the system temp
     dir, regardless of `media_directory` / `allowed_base_dirs`. The comment
     says "testing/development", but there is no build tag and no config gate —
     it ships in production, where on Linux/Docker that means all of `/tmp`.
     Demonstrated escape: with a temp-rooted media dir,
     `<mediadir>/../../etc/passwd.mkv` is accepted. Not fixed because several
     tests appear to rely on it, so it needs its own PR plus test fallout work.
   - Two open CodeQL "uncontrolled data in path expression" alerts on
     `pkg/webserver/subtitlepath.go`. I believe they are safe and pinned both
     constraints with tests in `pkg/security/formatpath_test.go`. Dismiss them
     or don't — your call.

4. **Remaining 🟡 matrix items**, if the above run out: scheduled search &
   upgrade loop, per-provider throttling depth, manual-search ranking (only
   `opensubtitles` implements `Searcher`), subsync original-language reference,
   per-job scheduler intervals.

5. **Providers are blocked, not merely unfinished.** Phase 1 (keyless) is
   genuinely exhausted — eight candidates were probed live on 2026-07-31 and
   the results are tabulated in `PROVIDER_BUILDOUT_PROMPT.md`. Phase 2 needs
   credentials in my local `~/.subtitle-manager.yaml`; **only wire config key
   names and protocol, never handle the secret values, and use httptest mocks.**
   Ask me which providers I've configured rather than guessing.

## How I want you to work

- One focused PR per item, off a worktree from `origin/main`. Each needs a
  `changelog.d/` fragment and **mandatory** file version headers
  (`file:` / `version:` / `guid:` / `last-edited:`, in that order — bump on
  every change, including files you only touched lightly).
- `rm -rf .standards` in a new worktree and use `git -c submodule.recurse=false`
  — the submodule otherwise breaks git operations there.
- **Prove every test is non-vacuous**: break the fix, confirm the matching test
  fails, restore. This has caught roughly a dozen tests that passed for the
  wrong reason, including several I had already called verified.
- **Don't trust a passing test as evidence the feature works.** Nearly every
  real bug in this project was invisible until something was actually run —
  a live provider fetch, a real HTTP request, the app itself. Prefer verifying
  against reality over verifying against a mock.
- Document decisions in the PR body and the parity doc as you go, so I can
  review and disagree later.

---
