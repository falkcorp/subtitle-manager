### Fixed

#### Replace the stale `pr-automation.yml` with the ghcommon standard

subtitle-manager still had the old minimal standalone labeler workflow, which
lacked `permissions: pull-requests: write` and so failed on every PR with
`Resource not accessible by integration` ("Auto Label PR"). Synced the standard
`pr-automation.yml` from ghcommon (which sets the correct permissions and carries
the full PR-automation), completing the workflow standardization.
