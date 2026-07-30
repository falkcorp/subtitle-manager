<!-- file: changelog.d/fix-frontend-tests.md -->
<!-- version: 1.0.0 -->
<!-- guid: 2d6f80b1-9e34-4c57-a812-70f4c3d9e5a6 -->
<!-- last-edited: 2026-07-30 -->

### Fixed

#### Most of the frontend test suite did not run

On `origin/main`, `npx vitest run` reported **11 of 24 test files failing, 20 of
52 tests**. The UI effectively had no trustworthy automated coverage, which is
why nothing caught the settings-page and routing defects fixed elsewhere in this
series. Six files are repaired here, taking it to **4 files / 9 tests**.

Four distinct causes, none of them a broken component:

- **Tests written for Jest, running under Vitest.** `BackButton` and
  `ProviderConfigDialog` used `jest.fn` / `jest.mock` / `jest.requireActual` and
  died with `ReferenceError: jest is not defined`. Note `vi.importActual` is
  **async**, unlike Jest's `requireActual`: the mock factory must `await` it, or
  every real export of the mocked module goes missing.
- **`localStorage` was undefined** in this jsdom setup, so `Navigation` crashed
  before rendering — surfacing as a confusing element-not-found failure rather
  than anything mentioning storage. A guarded polyfill now lives in
  `test-setup.js`, so a real implementation or a per-test stub still wins.
- **Tests mocked `global.fetch` against components that use `apiService`.**
  `UserManagement`, `ConfigEditor` and `DatabaseSettings` were migrated to
  `apiService.*` and no longer touch `fetch`, so the mock observed nothing and
  the assertions timed out waiting for a call that could never happen.
- **MUI puts `data-testid` on the `TextField` wrapper**, not the inner control,
  so `getByTestId('config-editor').value` was `undefined`. The test now queries
  the textarea itself.

#### The web UI was broken under a configured `base_url`

**Ten components called `fetch('/api/...')` directly with hardcoded paths**,
bypassing `apiService`, which prefixes `getBasePath()`. Under a server
`base_url` of `/sm` the browser requested `/api/tags` rather than
`/sm/api/tags`. That path has no route, so the single-page-app catch-all
answered with `index.html` and a `200`, and the component then failed parsing
HTML as JSON or rendered nothing.

This is the client-side twin of the server-side path handling fixed elsewhere
in this series, and **that fix does not help**: the request never reaches the
right route to begin with. Verifying the server under a base URL therefore
proved less than it appeared to.

All 31 call sites across those ten components now go through a new `apiFetch`
helper. `apiFetch` is deliberately **not** `apiClient`: `apiClient` injects
`Content-Type: application/json`, which would corrupt the callers that post
FormData for file uploads or expect a non-JSON response. `apiFetch` only fixes
the URL, so it is a safe one-for-one substitution — which also means the
existing tests, which mock `global.fetch`, keep working unchanged.

It preserves fetch's call arity too. Always passing a second argument turns
`fetch(url)` into `fetch(url, undefined)` — behaviourally identical, but it
changes what a spy records, and it broke two previously-passing test files
before that was corrected.

Components migrated: `App`, `Settings`, `TagManagement`, `History`, `Wanted`,
`MediaLibrary`, `NotificationSettings`, `LanguagesSettings`, `AuthSettings`,
`TagSelector`.

### The remaining four files

All now pass. The suite is **24 files / 56 tests green**, from 11 files and 20
tests failing on `main`. Their causes:

- **`TagManagement`** asserted a two-argument `fetch` call for what is a bare
  GET. `apiFetch` preserves fetch's arity, so it is a one-argument call.
- **`ProviderCard`** looked for `role="checkbox"`; MUI's `Switch` renders
  `role="switch"`.
- **`Settings`** rendered the component without a Router, so every test in the
  file died on `useNavigate() may be used only in the context of a <Router>` —
  the component grew routing after the tests were written.
- **`App`** asserted the login heading reads "Subtitle Manager", but it renders
  `t('app.title')`, so the assertion depended on whether i18n was initialised
  rather than on the login form. It now matches the literal subtitle beneath it.

### A missing route found on the way

`/api/providers/available` **has no handler registered**, so the request falls
through to the single-page-app catch-all and returns `index.html` with a `200`.
`ProviderConfigDialog` assigned that straight to state and crashed on
`availableProviders.map is not a function` — which reads as a React bug rather
than a missing route.

The component now checks the shape before using it, so a wrong response
degrades to an empty list instead of a crash. **That is a guard, not a fix:**
the route still needs mounting server-side, and the "add provider" dialog will
list nothing until it is. It is also not in `route_coverage_test.go`'s path
list, which is hand-curated — so the guard built to catch exactly this class of
bug missed it.

`ProviderCard` fails on `Unable to find an accessible element with the role
"checkbox"` — a genuinely stale UI assertion, not investigated.
