### Fixed

#### Replace the stale `pr-automation.yml` with the ghcommon standard

subtitle-manager still had the old minimal standalone labeler workflow, which
lacked `permissions: pull-requests: write` and so failed on every PR with
`Resource not accessible by integration` ("Auto Label PR"). Synced the standard
`pr-automation.yml` from ghcommon (correct permissions + full PR automation),
completing the workflow standardization. Also refreshed the stale
`super-linter-pr.env`: the old copy began with a `#` comment line, which the
workflow's `cat super-linter-pr.env >> "$GITHUB_ENV"` step cannot parse; the
standard version encodes its header as `KEY=value` pairs so it loads cleanly.
