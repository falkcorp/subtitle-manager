<!-- file: docs/tasks/GCOMMON-MIGRATION-STATUS-2026-07-18.md -->
<!-- version: 1.0.0 -->
<!-- guid: a216242e-b4c6-47f6-9913-a35cdf186467 -->
<!-- last-edited: 2026-07-18 -->

# gcommon Migration Status (2026-07-18)

Investigation notes ahead of unblocking `go build ./...` on `main`, which has
been broken independent of any change made this session. Written up for
review before/alongside the fix, per [[GCOMMON-CORRECTION.md]] and
[TASK-01-002](01-core-architecture/TASK-01-002-gcommon-adoption.md), both of
which predate the `jdfalk` → `falkcorp` org transfer and are now stale.

## Answer: is anything actually updated to falkcorp?

No, at every level:

- `go.mod`'s own module line is still `module github.com/jdfalk/subtitle-manager`
  (repo's remote is `falkcorp/subtitle-manager`, but the Go module identity
  was never moved).
- `go.mod` requires `github.com/falkcorp/gcommon v1.8.0` for gcommon, but that
  version was **never published under the falkcorp org** — real root tags
  start at `v1.9`. This require line was apparently produced by an
  automated org-rename pass that rewrote the org name without checking the
  version actually existed there.
- Every one of the 28 Go files (16 source + 12 test/mock) that touch gcommon
  types still imports the deprecated, pre-transfer
  `github.com/jdfalk/gcommon/sdks/go/v1/*` SDK layout — old org, old
  directory structure. None import the current `pkg/*pb` monorepo layout.
- `docs/tasks/GCOMMON-CORRECTION.md` (Sept 2025) still tells readers to use
  `github.com/jdfalk/gcommon/proto/gcommon/v1/...` imports — written before
  the transfer and never revisited.
- The legacy `sdks/` directory itself no longer exists in `falkcorp/gcommon`
  at any current ref (`main`, `v1.9`, `v2.1`, etc. all 404 on `/sdks`) — it
  was removed when the repo moved to the current `pkg/*pb` layout, so there
  is no falkcorp tag that still carries the old SDK shape.

## What actually breaks the build

```
pkg/cache/proto.go:8:2: github.com/falkcorp/gcommon@v1.8.0: reading
  github.com/falkcorp/gcommon/go.mod at revision v1.8.0: unknown revision v1.8.0
```

(same for `pkg/database/pb_conversions.go`, `pkg/queue/jobs.go`,
`pkg/media/client.go` — every direct importer of the dead require line.)

Confirmed via `git stash` against unmodified `origin/main`, so this predates
and is unrelated to anything done this session or in gcommon PR #2 (see
[[project-gcommon-pr2-authservice-v2]] memory).

## Legacy SDK usage inventory

Only 5 legacy subpackages are actually used, across 16 source files (plus
12 tests/mocks importing the same packages transitively):

| Legacy import (`github.com/jdfalk/gcommon/sdks/go/v1/...`) | Files | Current falkcorp equivalent |
|---|---|---|
| `common` | `cache/proto.go`, `config/types.go`, `database/{database,pebble,postgres,store}.go`, `authserver/server.go`, `gcommonauth/auth.go`, `gcommon/config/config.go`, `gcommon/subtitle_format.go`, `gcommonlog/logrus_provider.go` | `pkg/commonpb/v2` (package `v2`) |
| `database` | `database/pb_conversions.go` | `pkg/databasepb/v2` (package `v2`) |
| `queue` | `queue/jobs.go` | `pkg/queuepb/v2` (package `v2`) |
| `media` | `media/{client,server}.go` | `pkg/mediapb/v2` (package `v2`) |
| `metrics` | `metrics/metrics_gcommon.go` | `pkg/metricspb/v2` (package `v2`) |

Spot-checked the type/method surface for `CachePolicy` (common),
`Row` (database), `MediaServiceClient` (media), and `QueueMessage` (queue):
names and opaque getter/setter methods (`GetX`/`SetX`) match 1:1 between the
legacy SDK and the current `pkg/*pb/v2` packages. On the surface this looks
like a mechanical import-path rewrite, not a semantic rewrite — **but this
was only spot-checked on a handful of types**, not exhaustively, and
`authserver`/`gcommonauth` are exactly the area gcommon PR #2 rewired (6→18
AuthService methods, most still stubbed) — expect real API drift there, not
just a rename.

## A second, previously-undiscovered problem: the submodules were never re-tagged either

This is new since [[project-gcommon-org-migration-incomplete]] (which only
looked at the root module). None of the `pkg/*pb/v2` submodules subtitle-manager
needs have ever been published under their current org/path:

```
$ go get github.com/falkcorp/gcommon/pkg/common/v2@v2.1.5
go: github.com/falkcorp/gcommon/pkg/common/v2@v2.1.5 requires
  github.com/falkcorp/gcommon/pkg/common/v2@v2.1.5: parsing go.mod:
    module declares its path as: github.com/jdfalk/gcommon/pkg/common/v2
            but was required as: github.com/falkcorp/gcommon/pkg/common/v2
```

Every real tag (`pkg/common/v1.9.7`, `pkg/common/v2.1.5`, and the database/
media/queue/metrics equivalents) was cut back when the module still declared
itself `github.com/jdfalk/gcommon/pkg/common/v2` — i.e. **before** both the
org rename *and* the `common` → `commonpb` directory rename on `main`. No tag
exists where the published go.mod matches the current import path. This is
the exact same failure mode as the root module bug fixed in commit
`08c26f44`, just not yet fixed for the submodules, and it blocks any
`go get @latest`/`@vX.Y.Z` for these packages from any downstream consumer,
not just subtitle-manager.

**Workaround that does work**: a pseudo-version pinned to `falkcorp/gcommon`'s
current `main` HEAD resolves cleanly, since HEAD's own go.mod for each
submodule is internally consistent (declares itself under the current
`falkcorp`/`*pb` path):

```
$ go get github.com/falkcorp/gcommon/pkg/commonpb/v2@94cd87ab47ad6036c9253224f26638bc484971c8
go: added github.com/falkcorp/gcommon/pkg/commonpb/v2 v2.0.0-20260717214808-94cd87ab47ad
```

This is what this session used to unblock the build. It is a pin, not a
release — **gcommon needs real `pkg/*pb/v2` tags cut under the falkcorp org
before any consumer can depend on a proper version instead of a pseudo-version
pinned to a commit.** Flagging as follow-up work on `falkcorp/gcommon`
itself, out of scope for this repo.

## Plan applied

1. Rewrite all 28 files' imports from `github.com/jdfalk/gcommon/sdks/go/v1/X`
   to `github.com/falkcorp/gcommon/pkg/Xpb/v2`, aliased to the same local
   identifier the code already uses (`common`, `database`, `queue`, `media`,
   `gmetrics`, `authpb`, `commonpb`, `mediapb`, `gcommon`) so call sites don't
   need to change.
2. Drop the dead `github.com/falkcorp/gcommon v1.8.0` require; add pseudo-version
   requires for `pkg/{commonpb,databasepb,mediapb,queuepb,metricspb}/v2` pinned
   to `falkcorp/gcommon@94cd87ab`.
3. Iterate `go build ./...` / `go test ./...`, fixing genuine API drift
   file-by-file (expected mainly in `gcommonauth`/`authserver`).
4. Left `go.mod`'s own module path as `github.com/jdfalk/subtitle-manager` —
   renaming that is a separate, higher-consequence decision (breaks any
   external importer of this repo as a Go module) not needed to fix the
   build, and out of scope here.

## Outcome: blocked, not fixed — gcommon's generated code is corrupted upstream

`go build ./...` and `go vet ./...` pass cleanly after the rewrite above (0
errors), which looked like success. It is not: **the compiled binary panics
at startup**, before even parsing flags:

```
$ go build -o /tmp/sm-spike . && /tmp/sm-spike --help
panic: runtime error: slice bounds out of range [-4:]
  .../google.golang.org/protobuf/internal/filedesc/desc_init.go:262
  .../github.com/falkcorp/gcommon/pkg/commonpb/v2@.../ack_level.pb.go:125
```

`go build`/`go vet` never execute package `init()`, so they can't catch this
— only actually running the binary (or `go test`, which does run `init()`)
surfaces it. Root cause, confirmed independent of subtitle-manager (a bare
Go program importing only `github.com/falkcorp/gcommon/pkg/commonpb/v2`
panics identically, on both protobuf-go v1.36.10 and v1.36.11/latest — not a
runtime-version issue):

**Commit `08c26f44` on `falkcorp/gcommon` (the org-migration fix from
2026-07-16, see [[project-gcommon-org-migration-incomplete]]) corrupted the
embedded raw protobuf `FileDescriptor` bytes in ~3523 generated `.pb.go`
files repo-wide.** Its blind `github.com/jdfalk/gcommon` →
`github.com/falkcorp/gcommon` sed rewrote the `go_package` string *inside*
each file's binary-encoded `rawDesc` byte literal, but didn't recalculate the
protobuf length-prefix byte that precedes it — "falkcorp" is 2 bytes longer
than "jdfalk", so every rewritten descriptor is now malformed. Confirmed via
`git diff 48e80cd3 08c26f44 -- pkg/commonpb/v2/ack_level.pb.go`: the length
byte stayed `)` (0x29=41) while the string it prefixes grew by 2 bytes. The
file still compiles (it's just a Go string literal to the compiler) but
`protoimpl.TypeBuilder.Build()` panics decoding it at `init()` time.

Net: **there is currently no usable version of `falkcorp/gcommon`'s
`pkg/*pb` submodules for any consumer.** Real tags (`pkg/common/v2.1.5` etc.)
have correct bytes but declare the pre-rename module path and fail to
resolve under the `falkcorp` import path at all; `main` HEAD resolves under
the right path but panics at init. The import rewrite in this repo (28
files, go.mod/go.sum) is believed correct and left in the working tree
uncommitted, but **cannot land until `falkcorp/gcommon` is fixed** — almost
certainly a full `buf generate` regeneration from source `.proto` files
followed by fresh tags, not a manual byte patch. That is out of scope for
this repo; tracked as a `falkcorp/gcommon`-side follow-up.
