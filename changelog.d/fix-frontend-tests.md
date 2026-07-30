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

### Not fixed: four files blocked on a product bug

`App`, `Settings` and `TagManagement` fail because their components call
`fetch('/api/...')` **directly, with a hardcoded path**, while `apiService`
prefixes `getBasePath()`. The tests assert against those literal paths.

Rewriting the assertions to match would have made the suite green while
cementing a real defect: **10 components bypass `apiService` this way**, so
under a configured `base_url` they request `/api/tags` instead of
`/sm/api/tags`, hit the single-page-app catch-all, and receive `index.html`
where they expect JSON. This is the frontend twin of the server-side path
handling fixed in the base-URL change, and it is not fixed by it — the requests
never reach the right route in the first place.

Affected: `App`, `Settings`, `TagManagement`, `History`, `Wanted`,
`MediaLibrary`, `NotificationSettings`, `LanguagesSettings`, `AuthSettings`,
`TagSelector`. Migrating them to `apiService` is a deliberate change of its own,
and the failing tests are left failing so it stays visible.

`ProviderCard` fails on `Unable to find an accessible element with the role
"checkbox"` — a genuinely stale UI assertion, not investigated.
