<!-- file: changelog.d/docs-buildout-and-ci-plan.md -->
<!-- version: 1.0.0 -->
<!-- guid: 074e7e52-ecb4-4fc2-bd85-7c6fc3c0af8f -->
<!-- last-edited: 2026-07-25 -->

### Changed

#### Provider build-out prompt and CI fix plan are now tracked docs

`docs/PROVIDER_BUILDOUT_PROMPT.md`, the governing document for the provider
build-out, was previously untracked. It is now committed and corrected: Phase 1
(keyless providers) is marked complete and exhausted, and its claim that
opensubtitles.com was "already implemented" is flagged as **wrong** — the
`opensubtitlescom` package is still a fictional stub, and the real integration
is the separate classic `opensubtitles` package.

The untracked `SUBTITLE_MANAGER_FIX_PLAN.md` scratch file is replaced by
`docs/CI_FIX_PLAN.md`, trimmed to the items re-verified as still live: the
protobuf generator's dead gcommon import path, and untriaged Dependabot alerts.
Items that had since been fixed (the CI Go-version pin, the Python gcommon pin)
are recorded as resolved so they are not re-opened.
