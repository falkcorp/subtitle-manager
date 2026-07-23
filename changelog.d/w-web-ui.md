### Added

#### Verify Sync page + frontend build repair

Added a "Verify Sync" tool page (`/tools/verify`) that runs subtitle drift
verification from the web UI via `/api/verify`. Also repaired the previously
broken frontend build: `vite.config.js` `manualChunks` is now a function (Vite 8
/ Rolldown requires it), and `ConfigEditor.jsx` uses `import * as yaml` since
`js-yaml` has no default export. See `docs/WHISPER_PIPELINE_DECISIONS.md`
(W-web). Note: a number of pre-existing frontend unit tests still fail and are
tracked separately.
