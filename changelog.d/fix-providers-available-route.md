<!-- file: changelog.d/fix-providers-available-route.md -->
<!-- version: 1.0.0 -->
<!-- guid: b58c07f2-4a1d-4e93-8067-c1f9d3a25e40 -->
<!-- last-edited: 2026-07-30 -->

### Fixed

#### `/api/providers/available` was answered by the tag handler

`ProviderConfigDialog.jsx` has always called this endpoint and no handler was
registered for it. The interesting part is what that meant in practice: the
request did **not** fall through to the single-page-app catch-all, because
`/api/providers/` is registered as a subtree for the universal tag handler — so
the *tag* handler answered it. The dialog assigned that response to state and
crashed on `availableProviders.map is not a function`, which reads as a React
bug rather than a routing one.

It now has a handler returning the provider list that `getAvailableProviders`
already builds. An exact pattern beats a subtree in `http.ServeMux`, so
registering it is sufficient — no reordering required.

### Added

#### A route guard that derives its list from the web UI

`route_coverage_test.go` checks a **hand-maintained** list of frontend paths.
That only ever catches paths somebody remembered to add, and it is why this gap
survived: nobody listed `/api/providers/available`.

`TestDiscoveredFrontendPathsAreMounted` scans `webui/src` for `/api/...`
literals and asserts each resolves to a route, so a new call to an endpoint that
was never mounted fails without anyone updating a list. It self-checks, failing
if the scan returns implausibly few paths rather than passing vacuously, and it
skips `__tests__` and `mockApi.js` — a development double whose
`urlPath.includes('/api/library')` is a prefix match, not an endpoint.

**What it cannot catch, stated plainly:** it asserts a path matches *some*
pattern, not the *right* one. This very bug is the example — the path resolved
happily to a neighbouring subtree, so both this guard and the curated one would
have called it mounted. A path swallowed by an adjacent subtree still needs a
test of the endpoint's actual behaviour, which is why
`TestAvailableProvidersReturnsAList` asserts a JSON array comes back rather than
trusting the routing.
