<!-- file: changelog.d/fix-base-url-path-parsing.md -->
<!-- version: 1.0.0 -->
<!-- guid: 7c4a2f61-38d9-4b05-9e72-6f0b1a83d5e4 -->
<!-- last-edited: 2026-07-30 -->

### Fixed

#### Seven API endpoints were dead when running under a `base_url`

Routes are mounted at `prefix + "/api/..."`, where `prefix` comes from
`base_url`. Seven handlers that pull an ID out of the request path were written
as if that prefix were always empty:

```go
strings.TrimPrefix(r.URL.Path, "/api/media/profile/")
```

Behind a base URL the incoming path is `/sm/api/media/profile/42`, which that
literal does not match. `TrimPrefix` returns the path unchanged, the following
`Split` yields `""` as the first segment, and the handler carries on with an
empty ID — straight into a database lookup rather than a rejection. These
endpoints were not degraded behind a subpath, they were **dead**:

- media profile assignment (get / assign / remove)
- profile lookup by ID
- user management by ID
- tag lookup, tag-to-user assignment, tag-to-media assignment

Path parsing now goes through `apiPath` / `pathSegment` / `pathSegments`, which
strip the configured prefix first. The prefix is only stripped on a whole path
segment, so a `base_url` of `/sm` does not mangle a request to `/small/...`.

The helpers also return nothing rather than an empty first element when the
path does not match the expected route, so "this is not the path I expected" is
a case the handler must handle instead of an empty ID silently flowing into a
lookup.

#### Why nothing caught it

Every test constructs requests with an empty `base_url`, where the literal
prefix happens to match — so a handler with this defect passes its tests and
fails only on a real deployment behind a subpath. That is why this change also
adds a source-level guard (`TestNoRawAPIPathParsing`) that fails the build for
any handler parsing `r.URL.Path` against an `/api` literal. Seven had already
drifted into the pattern; catching the eighth by eye is not a plan.
