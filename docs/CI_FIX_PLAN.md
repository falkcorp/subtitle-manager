<!-- file: docs/CI_FIX_PLAN.md -->
<!-- version: 1.0.0 -->
<!-- guid: 38d9f2af-a14a-4e72-8dea-6f0d885f8680 -->
<!-- last-edited: 2026-07-25 -->

# CI / release pipeline — known issues

Successor to the untracked `SUBTITLE_MANAGER_FIX_PLAN.md` scratch file, trimmed
to the items that are **still live as of 2026-07-25**. Each claim below was
re-verified against the repository at that date rather than carried forward on
faith — the original file had drifted, listing several failures that were
already fixed.

## Working rules for this repo

- Work in a git worktree; **rebase-merge only** (branch protection).
- Every PR needs a `changelog.d/` fragment (a `changelog-check` CI job enforces
  it), or `[skip changelog]` in the PR title / the `skip-changelog` label.
- A `guid-index` pre-commit hook regenerates `.github/guid-index.json` and
  **aborts the first commit**; `git add -A` and re-commit. **Do not blind
  `--amend`** after a failed hook — it rewrites the wrong commit.
- Module path is `github.com/jdfalk/subtitle-manager`; the remote is
  `falkcorp/subtitle-manager`.
- The real CI gate is **`Go CI (1.25)`** + **`Analyze (go)`**. One-second
  `Code Quality Check` / `Intelligent AI Labeling` failures are stale
  concurrency-cancelled `pr-automation` jobs — the latest run's conclusion is
  authoritative.

## Still open

### 1. Protobuf generator references a dead gcommon path

`scripts/generate_protobuf_services.py:549` still imports the legacy
`github.com/jdfalk/gcommon/sdks/go/v1/common`. That module path no longer
exists: gcommon became protos-only, published to the Buf Schema Registry, and
`go.mod` now consumes `buf.build/gen/go/falkcorp/gcommon/...` (commit
`233be382`).

This does not currently break `Continuous Integration` or the `Release`
workflow — release-time protobuf generation was disabled for this repo (PR
#2204) precisely because the generated `.pb.go` files are committed. So the
script is dead weight rather than an active failure. Decide between:

- updating it to the BSR/`buf generate` layout, if regeneration is still wanted
  as a developer tool; or
- deleting it, if committed `.pb.go` files are the intended workflow.

### 2. Dependabot alerts (9 open on the default branch)

As of 2026-07-25: **5 high, 3 medium, 1 low**. Not yet triaged. Only open PR
addressing any of them is #2166 (dompurify bump). Re-read the current list
before acting — this count goes stale quickly:

```bash
gh api repos/falkcorp/subtitle-manager/dependabot/alerts \
  --jq '[.[] | select(.state=="open")] | group_by(.security_advisory.severity)
        | map({sev: .[0].security_advisory.severity, n: length})'
```

## Resolved since the original plan

Recorded so nobody re-opens them:

- **CI Go version.** `ci.yml` now pins `go-version: '1.25'`, matching
  `go.mod`'s `go 1.25.0`. The old `1.24` pin is gone.
- **Python gcommon pin.** `requirements.txt` no longer carries the dead
  `git+https://github.com/jdfalk/gcommon.git@df100011…` editable install, nor
  the missing-`selenium` / unterminated-string-literal failures.
- **Release workflow red on every push to `main`** (protobuf generator needing
  Go ≥ 1.25 on a Go 1.23 runner; goreleaser aborting on an untracked
  `proto-generated/`). Fixed in PR #2204 by disabling release-time protobuf
  generation for this repo and gitignoring the output directory.
- **"14-way broken / ambiguous imports" diagnosis.** Stale. `go build ./...`
  and `go vet ./...` pass clean.
