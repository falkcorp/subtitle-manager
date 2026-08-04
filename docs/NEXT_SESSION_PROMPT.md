<!-- file: docs/NEXT_SESSION_PROMPT.md -->
<!-- version: 2.0.0 -->
<!-- guid: 3a7f21c8-64b9-4e05-9d3a-8f1e07b26c54 -->
<!-- last-edited: 2026-08-04 -->

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
it has caught post-merge breakage twice**), and the frontend suite. CI now also
cross-compiles the four goreleaser targets, so a Unix-only syscall can no longer
pass every check and break only the release.
A running instance for evaluation lives at `~/subtitle-manager-preview/`
(`sh start.sh` → http://127.0.0.1:8080, `admin` / `evaluate-me-2026`).

Read `docs/BAZARR_PARITY_STATUS.md` for the full matrix and
`docs/PROVIDER_BUILDOUT_PROMPT.md` for the provider plan. **Verify claims in
both against the code before acting on them** — several have been wrong, most
recently three ✅ rows claiming language profiles were wired to downloads when
nothing read them. Both docs were updated on 2026-08-04 and should be accurate
as of then.

## What the last session (2026-08-04) did

Six PRs, all merged. The language-profile work is **finished**; what is left is
mostly providers and polish.

- **#2238** — mass-edit / bulk profile assign: `POST /api/media/profiles/bulk`
  plus a Mass Edit mode in `MediaLibrary.jsx`. The route is an **exact**
  `ServeMux` pattern because `/api/media/` is a subtree for the tags handler,
  which would otherwise answer it with a 200 and the wrong body.
- **#2239** — the three profile bugs. A file assigned the *default* profile now
  counts as assigned (new `GetAssignedProfileID` on `SubtitleStore`); the last
  profile can be deleted; and `PebbleStore.GetDefaultLanguageProfile` no longer
  *creates* a profile on an empty store, which used to resurrect a just-deleted
  one on the next lookup. `cmd/profiles.go` honours `db_backend`.
- **#2241** — `ValidateAndSanitizePath` no longer accepts anything under
  `os.TempDir()` in production. Gated on `testing.Testing()`, so no test changed.
- **#2242** — profiles now govern the monitor loop, watcher, Sonarr/Radarr
  webhooks and the `sonarr`/`radarr` CLI, not just library scans.
- **#2243** — **releases had been failing since ~2026-08-03**: `syscall.Statfs`
  does not exist on Windows, so goreleaser's windows build died and took the
  release with it, invisibly, because CI only compiled the runner's platform.
  Fixed per-platform, and CI gained a `cross-build` job over all four goreleaser
  targets.

### Decisions made, for you to disagree with

- **An assigned profile beats `MonitoredItem.Languages`.** That field is
  *accumulated*, not curated — `pkg/monitoring/sync.go` unions in every language
  it is ever asked for and never removes one, so it cannot express "stop wanting
  German". An assignment is a deliberate per-file choice.
- **Requests that name a language directly are not overridden**: the web
  download endpoint's `?lang=`, the custom webhook's validated `lang`, and
  `fetch <file> <lang>`.
- **Deleting the only language profile is allowed.** There is no other profile
  to fall back to either way, and refusing left the user permanently stuck.

### Process notes worth keeping

- A PR branched off another unmerged branch ends up `CONFLICTING` once that
  branch rebase-merges, and **GitHub cannot build a merge ref for a conflicting
  PR, so `Continuous Integration` never queues at all**. It looks like a broken
  trigger. Check `gh pr view N --json mergeable,mergeStateStatus` first. Branch
  from `origin/main`.
- `workflow_dispatch` on `ci.yml` **skips Go CI** — `Detect Changes` has no diff
  to compare against — so it is not a substitute signal.
- Every action in this repo is **SHA-pinned**; a tag ref fails at "Set up job",
  which reads like an infrastructure flake rather than a config error.
- Plain `go test` misses what CI catches: run **`go test -race`** on any package
  whose tests mutate package globals (`providers.SetInstances`,
  `security.allowTempDirPaths`).

## What to pick up, roughly in order

1. **Providers, Phase 2 (credential-gated).** This is now the biggest remaining
   parity gap. Phase 1 (keyless) is genuinely exhausted — eight candidates
   probed live 2026-07-31, tabulated in `PROVIDER_BUILDOUT_PROMPT.md`. Phase 2
   needs credentials in my `~/.subtitle-manager.yaml`; **only wire config key
   names and protocol, never handle the secret values, and use httptest mocks.**
   Ask me which providers I have configured rather than guessing. Note
   `opensubtitlescom` is still a stub despite the prompt doc claiming otherwise.

2. **Two CodeQL "uncontrolled data in path expression" alerts** on
   `pkg/webserver/subtitlepath.go`. Believed safe; both constraints pinned by
   `pkg/security/formatpath_test.go`. Dismiss them or don't — it is a judgement
   call I did not want to make for you.

3. **Cutoff score and Forced/HI are still dropped** on the profile path.
   `processWithAssignedProfile` passes a bare language code, so `ProcessFile`
   scores with the global `scoring.*` settings and the profile's `CutoffScore`
   and per-language `Forced`/`HI` never reach the scorer. This is the last real
   hole in the profile feature.

4. **`/api/providers/status` is a lazily-filled cache** that stays `{}` until
   `POST /api/providers/refresh`, and that refresh hardcodes only
   `opensubtitles` + `subscene` — so real providers never appear and the stub
   `subscene` reports `available: true`.

5. **Whisper is done; leave it alone.** `whisper.transcribe_url` points at the
   server's GPU container (`large-v3-turbo`, int8). Verified 2026-08-04 end to
   end through the `transcribe` CLI: 10 minutes of audio, 76 s, 333 cues.
   `ASR_QUANTIZATION=int8` is **mandatory** — the image defaults to float32 on
   GPU and OOMs a 4 GB card during model load. That model **cannot translate**,
   which is safe only because both `ASROptions{}` call sites in
   `pkg/transcriber` omit `Task`. Do not record host addresses here; this file
   is public.

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
