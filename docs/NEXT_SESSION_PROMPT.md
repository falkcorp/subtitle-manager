<!-- file: docs/NEXT_SESSION_PROMPT.md -->
<!-- version: 5.0.0 -->
<!-- guid: 3a7f21c8-64b9-4e05-9d3a-8f1e07b26c54 -->
<!-- last-edited: 2026-08-11 -->

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

## State as of 2026-08-11: everything is merged

`main` is at `2ed2636d`. **Zero open PRs.** No uncommitted work of mine exists.

Landed this session:

- **The Media Library works.** Four defects were fixed (#2258) — the page had
  been rendering *completely empty* because the component read `is_dir` while
  the server sends `isDirectory`; breadcrumbs called an undefined function;
  opening a directory refetched the previous one; and "Select all" swept in
  sidecar `.srt` files. All four are covered by tests written test-first.
- **Bilingual "double subs" work, manually** (#2259, #2260). Tick two subtitles
  on a file in the Media Library, press **Combine**. Verified in a real browser
  against a real library.
- **Executive summaries** backfilled across the whole project history (#2257) in
  `docs/executive-summaries/`, with a template. These exist to justify cost and
  work to non-technical decision-makers. Update them in place; there is a house
  format documented in `TEMPLATE-executive-summary.md`.
- All dependabot PRs merged.

## Verified working, end to end, in a real browser

A local build served a real library and did the following for real — not from
tests, and not from a toast message:

- Scanned and **detected** a nested library (`TV Shows/Breaking Bad/Season 01/`),
  associating both sidecar subtitles with the episode.
- Navigated four levels deep, used the breadcrumb, and ran Mass Edit.
- **Combined** an English and a Spanish sidecar into one bilingual file, three
  cues, English over Spanish, confirmed on disk.

**To reproduce the environment:** you must build locally with
`CGO_ENABLED=1 go build -tags sqlite`, *and* run `npm run build` in `webui/`
first — the frontend is embedded via `webui/embed.go`, so building the Go binary
alone silently serves a stale/empty shell. `make build-sqlite` does both but
writes to `bin/subtitle-manager-sqlite`, **not** `bin/subtitle-manager`.

## The four things worth doing next, in order

### 1. Release binaries cannot run `web` at all — CONFIRMED, needs your decision

I downloaded the published `darwin_arm64` artifact from `v1.0.1-rc.53` and ran
it. It fails:

```
Error: web server requires SQLite for authentication. Please build with: go build -tags sqlite
```

**No published release binary, on any platform, can start the web UI.** Only the
Docker image works, because the Dockerfile passes `-tags=sqlite`. The web server
opens a SQLite auth DB unconditionally, even when `db_backend: pebble`.

Three ways out, and this is a decision I want from you rather than a guess:

| Option | Scope |
| --- | --- |
| **Pure-Go SQLite driver** (`modernc.org/sqlite`) | ~40 lines + one dependency. Everything funnels through one `Open()` in `pkg/database/sqlite_disabled.go`; swap the `!sqlite` build-tag stub for a real implementation. Fixes every platform at `CGO_ENABLED=0`. Large new dependency; migrations need exercising under it. |
| **CGO releases with `-tags sqlite`** | No code change, but needs a cross-compiler toolchain per target. Fragile releases. |
| **Pebble-native auth** | 16 auth functions take `*sql.DB` and 87 call sites pass it. Cleanest architecturally, by far the most work. |

I lean toward the first. **This also fixes the `user` CLI**, which currently dies
with `unable to open database file: is a directory` on a Pebble deployment
(workaround: `--db-path <dbdir>/auth.db`).

### 2. Settings → Providers is dead, and fails silently

You cannot enable or configure **any** provider from the web UI.
`Settings.jsx:233` PATCHes `/api/providers/{name}` and `:266` POSTs
`/api/providers/{name}/config`; neither is mounted, and both are swallowed by
the `/api/providers/` **subtree** at `server.go:316`. Both call sites are
`if (response.ok)` with no `else`, and `ProviderConfigDialog.jsx:189-190` closes
the dialog unconditionally — so it reads as success.

`POST /api/config` with `{"providers.gestdown.enabled": true}` **works today**,
so repointing the UI is probably better than adding two routes. Only `embedded`
ships enabled, and it needs ffmpeg plus a real container — so out of the box,
via the UI alone, there is no route to a working provider.

### 3. Bilingual subtitles automatically, not just manually

The manual path is done. The automatic one is not: when a language profile asks
for two languages, it should produce the bilingual file during scan/download.
Building blocks all exist (`subtitles.StackTracks`,
`POST /api/subtitles/stack`, `merge --stack`, correct sentinel-language
detection).

Two real decisions in it: does the bilingual file **replace** the separate
per-language sidecars or come **in addition**? And `singleLanguageNaming()`
collapses multi-language output to one filename, so a multi-language profile
would currently overwrite itself.

### 4. Python CI is broken and hidden

`Python CI (3.13)` fails on every PR that touches Python dependencies, and
reports **skipped** on main because `Detect Changes` finds no Python diff. So
the Python build has been broken while main looks green. Likely causes: a dead
`gcommon` git pin in `requirements.txt`, a missing `selenium` test dependency,
and a `SyntaxError` in an inline `python -c` in a workflow. **Ask first whether
Python is needed here at all** — this is a Go service with a React UI, and
deleting a vestigial surface beats fixing it.

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
- **22 of 51 provider hostnames do not resolve** — fabricated `api.<name>.com`
  shells taking a slot in every fetch wave. Audit the registry, not the hostnames.
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

- `docs/NEXT_SESSION_PROMPT.md` had an uncommitted modification predating my
  session; I left it alone until this rewrite.
- Untracked debris in the repo root: `SUBTITLE_MANAGER_FIX_PLAN.md` (dated
  2026-07-22, self-describes as delete-when-done), `dashboard-loggedin.png`,
  `.playwright-mcp/`.
- A stale worktree registration: `git worktree prune` will clear a `prunable`
  entry whose directory is gone, and the empty branch
  `fix/media-library-isdirectory` can be deleted.
- `~/.worktrees/subtitle-manager-release-fix` (`fix/release-protobuf`) and five
  git stashes on abandoned copilot/codex branches have never been checked for
  unpushed work.
- A local server may still be running on `127.0.0.1:18099`.

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
