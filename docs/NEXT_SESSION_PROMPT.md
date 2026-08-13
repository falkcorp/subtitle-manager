<!-- file: docs/NEXT_SESSION_PROMPT.md -->
<!-- version: 7.1.0 -->
<!-- guid: 3a7f21c8-64b9-4e05-9d3a-8f1e07b26c54 -->
<!-- last-edited: 2026-08-12 -->

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

## State as of 2026-08-12

Items 1 and 2 of the previous list are **done and merged**. `main` is at the
Settings→Providers work. Landed 2026-08-12:

- **Release binaries can run `web`** (#2264). SQLite now comes from the pure-Go
  `modernc.org/sqlite` driver and the `sqlite`/`nosqlite` build tags are gone
  entirely — one CGO-free build, both backends, every platform. Verified by
  running a release-style binary, not by tests.
- **Settings → Providers works** (#2265, #2266). Browser-verified: Add Provider
  → Gestdown → Save writes the config and the provider registers **without a
  restart**. A settings save also used to bake `providers.embedded.enabled` into
  the config file (viper.WriteConfig serialises defaults), which on the next
  start collapsed subtitle search to embedded-only; the writer now persists only
  operator-submitted keys, and a request that would replace a settings *group*
  with a single value is refused with 400 before anything is mutated.
- **The dead Python surface is gone** (#2267). `requirements.txt` was Bazarr's
  110-package list that nothing imported, and it was the `requirements*.txt`
  path trigger keeping the broken-and-hidden Python CI alive. `tests/e2e`
  (Selenium, never run, never had its dependency declared) went with it.
  `Python CI (3.13)` now **runs and passes** instead of skipping. Kept:
  `sdks/python` (a published SDK), `scripts/assemble_todo.py` (mandated by
  CLAUDE.md), and the rest of `scripts/*.py`.
- **Bilingual subtitles are automatic** (#2268, merged). A profile
  with 2+ languages stacks the top two during scan/download, additively. Two
  files: `Episode.en-es.srt` (self-describing) and a reflinked
  `Episode.eo.srt` (the sentinel media servers actually surface). Reflink is
  APFS clonefileat / FICLONE on btrfs, XFS and OpenZFS ≥2.2, degrading to a
  copy; deliberately not a hardlink, which would share an inode.

  CodeQL caught 12 high-severity `go/path-injection` alerts in the first cut of
  this — `CloneFile(src, dst string)` handed user-derived paths straight to
  `os.Open`. Fixed by reshaping it as `CloneFileIn(root *os.Root, srcName,
  dstName string)`. **Any new file operation on a media-derived path needs
  `os.Root` confinement**; `security.Validate*` is not recognised as a
  sanitizer. The branch shows ~38 open alerts but ~26 are pre-existing
  elsewhere, so the check gates on newly-introduced ones only.

Four defects had to be fixed along the way. Two were pre-existing bugs the build
tag had been hiding from CI, and one is far more serious than the page itself:

- **Saving ANY setting could collapse subtitle search to one provider.**
  `viper.WriteConfig` serialises `AllSettings()`, which merges defaults, and
  `cmd/root.go` set `providers.embedded.enabled: true`. Saving one unrelated
  setting wrote that default into the config file; on the next start
  `InConfig` reported true, `LoadFromConfig` registered one instance, and
  `FetchFromAll` dropped out of its all-providers fallback to embedded-only.
  `POST /api/config` now writes only operator-submitted keys, and the dead
  default is gone.
- A data race marshalling a live `*tasks.Task` on `POST /api/tasks/start`.
- A clean shutdown exiting status **1**, which a supervisor reads as a crash.
- The `user` CLI opening the Pebble *directory* instead of `auth.db`.

**Two corrections to the previous handoff, which asserted otherwise:**

1. The **Docker image was not a working escape hatch**. Its build branched on
   `TARGETARCH`; amd64 got `-tags=sqlite`, **arm64 got neither**. The arm64
   container could not run `web` either.
2. `POST /api/config` with `providers.<name>.enabled` did **not** "work today".
   It persisted, but `viper.Set` only populates the override layer while
   `LoadFromConfig` reads `viper.InConfig` (the file layer), so nothing applied
   until a restart. Repointing the UI alone would have moved the silent failure,
   not fixed it.

## Reproducing the environment

`CGO_ENABLED=0 go build .` is now enough — no build tags, no CGO. You still must
run `npm run build` in `webui/` (and `go generate ./webui`) first, or the binary
silently serves a stale/empty shell. `make build` does both and writes to
`bin/subtitle-manager`.

## The things worth doing next, in order

### 1. Browser-verify automatic bilingual output

The feature (#2268) is covered by unit tests — both output files, stacking
order, never-overwrite, the partial case, and the reflink backend — but it has
**not** been driven end to end. Doing so needs a real library with a
two-language profile assigned to a file, then a scan, then checking both
`<base>.en-es.srt` and `<base>.eo.srt` land on disk with the right content and
that the player lists the `eo` track.

The operator's decisions, for the record: bilingual is **additive** (the
per-language sidecars stay), the hyphenated name is the primary artifact, and
the `eo` copy exists so media servers show it.

Note the previous handoff was wrong that `singleLanguageNaming()` "would
overwrite itself" — `pkg/scanner/profile.go:116` already truncates a
multi-language profile to one language, so the real prior behaviour was
*silently dropping* the other languages.

### 2. Audit the provider registry

22 of 51 provider hostnames do not resolve — fabricated `api.<name>.com` shells
taking a slot in every fetch wave. Audit the registry, not the hostnames.

## Also open

- **OpenSubtitles is blocked on you.** Verified live: the key supplied on
  2026-08-06 behaves *identically to garbage and to sending nothing* (403 on
  both `/subtitles` and `/login`, under two different User-Agents). Your paid
  VIP is on the **legacy opensubtitles.ORG** account, which is a different system
  and does not provision a `.com` REST key. Register an API consumer at
  opensubtitles.com → Profile → API consumers and give me the key plus the
  username/password. Bazarr works by shipping its own embedded key; copying it
  would impersonate Bazarr and share its rate limit, so I did not.
- **Cloudflare Access** needs your team domain and audience before it can be
  built or tested.
- **Two CodeQL path-injection alerts** on `pkg/webserver/subtitlepath.go` await
  your call. Worth knowing: those are read-only `os.Stat` checks. A *third*
  alert appeared this session on a genuine **write** and was fixed properly with
  `os.OpenRoot`, not dismissed — so the two remaining ones are the weaker case.
- `pkg/webserver` profile tests default to SQLite while the product defaults to
  Pebble. That gap has hidden two real bugs.
- **ZFS-aware tuning** — measure before changing anything. Every item is a
  hypothesis.
- **Bazarr API compatibility layer: scope only, do not build.** Lowest priority.

## Housekeeping I did not do without asking

- Untracked debris in the repo root, all predating me:
  `SUBTITLE_MANAGER_FIX_PLAN.md` (dated 2026-07-22, self-describes as
  delete-when-done), `dashboard-loggedin.png`, `.playwright-mcp/`.
- `~/.worktrees/subtitle-manager-release-fix` (`fix/release-protobuf`),
  `~/.worktrees/sm-libdir` (`docs/next-session-prompt-v5`) and five git stashes
  on abandoned copilot/codex branches have never been checked for unpushed
  work. Check before removing: a rebase-merged branch shows its commits as
  "not in origin/main" because the SHAs were rewritten, so compare **content**
  (`git diff <merged-commit> HEAD`) rather than trusting the commit list.
- Several merged branches are prunable (`feat/combine-subs-ui`,
  `feat/stack-bilingual-subs`, `fix/media-library-browse-shape`,
  `fix/media-library-isdirectory`).
- **Dependabot: 5 high/moderate Go alerts, all `github.com/docker/docker`.**
  Not removable — `pkg/transcriber` genuinely uses the Docker client for
  Whisper containers, so they need a version bump. `modernc.org/sqlite`, added
  2026-08-12, has zero alerts.
- A local server may still be running on `127.0.0.1:18099` — it is an **old**
  binary from a previous session, so anything verified against that port is
  not testing your build. Check the listener before trusting a live check.

*(Resolved 2026-08-12: the stale scratchpad worktree registration was pruned,
`fix/media-library-isdirectory` deleted, and the uncommitted
`NEXT_SESSION_PROMPT.md` modification turned out to be byte-identical to
`origin/main` — local `main` was simply 13 commits behind.)*

## How I want you to work

- One focused PR per item, off a worktree from **`origin/main`**, created under
  **`~/.worktrees/`** — never a session scratchpad. A scratchpad worktree was
  cleared between sessions and destroyed four verified fixes plus their tests,
  which had to be redone from prose.
- `rm -rf .standards` in a new worktree and use `git -c submodule.recurse=false`.
- Each PR needs a `changelog.d/` fragment and **mandatory** file version headers
  (`file:` / `version:` / `guid:` / `last-edited:`, bumped on every change).
- **Write the test first and watch it fail** for the right reason before fixing.
  This repeatedly caught tests that would have passed for the wrong reason.
- **Never chain verification and publication in one command.** Run the suites,
  read the result, *then* push.
- **Fix the product, not the test.**
- **When you verify live, confirm which process answered** — check for a stale
  listener before starting anything.

### Traps that have cost real time

- **A green suite proves very little here.** All four Media Library defects
  survived a passing suite because the *fixtures* were wrong, not the
  assertions: the test fed `is_dir`, a key the server has never sent, so test
  and component agreed with each other while both disagreed with production.
  Build fixtures from observed server payloads.
- **React reports a throw inside an event handler as an unhandled error that
  Vitest does NOT count as a failure.** Assert the observable effect — a
  follow-up request, a changed list — never `console.error`.
- **A skipped check is not a passing check, and the reverse matters too.**
  Dropping the `sqlite` build tag made five `pkg/database` tests run for the
  first time (66 passing/8 skipped → 74/3) and exposed two real bugs in
  packages CI had never built. When de-gating anything, report the skip delta —
  a green suite before and after looks identical without it.
- **Reading CI: sort by duration, not status.** Concurrency-cancelled jobs are
  reported as failures; `Code Quality Check` and `Intelligent AI Labeling`
  routinely appear **twice**, once failing at 0–1s and once passing. But do not
  assume a failure is stale — a `CodeQL` failure at 3s this session was a
  genuine new high-severity finding. The tell is that it appeared once, not
  twice. Fetch the detail before dismissing anything.
- **A skipped check is not a passing check** (see Python CI above).
- **Wait-loops must count total pending checks**, not pending checks filtered by
  name — a name filter returns zero both when the job finished and when GitHub
  has not created it yet, so it exits instantly and looks green. `Go CI` takes
  ~10–11 minutes.
- **Login posts form data, not JSON.** `POST /api/login` with a JSON body
  returns 401 and looks exactly like bad credentials.
- An unmounted `/api` path under a **subtree** pattern is answered by the
  subtree's handler; an unmounted path with no subtree returns 200 +
  `index.html` from the SPA catch-all. Never trust a 200.
- **`/api/v1/infos/formats` on OpenSubtitles returns 200 for any key including
  garbage** — it is unauthenticated and useless as a credential test. Use
  `/subtitles` or `/login`.
- A PR branched off another *unmerged* branch goes `CONFLICTING` once that
  branch rebase-merges, and **GitHub cannot build a merge ref for a conflicting
  PR, so CI never queues at all**. Check `gh pr view N --json mergeable` first.
- `workflow_dispatch` on `ci.yml` **skips Go CI**, so it is not a substitute
  signal. Every action is SHA-pinned; a tag ref fails at "Set up job".
