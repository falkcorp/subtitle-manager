### Changed

#### Sync instruction/config files to the ghcommon house standard

Refreshed the synced house-standard files from ghcommon: `AGENTS.md`,
`CLAUDE.md`, `.github/instructions/`, `.github/prompts/`,
`.github/ISSUE_TEMPLATE/`, `.github/security-guidelines.md`, `.prettierrc`, and
`.prettierignore`. The repo-specific `.github/dependabot.yml` is deliberately
left as-is. Also queued a low-priority TODO to standardize the security workflow
(deferred from the workflow conversion because the repo uses CodeQL default
setup).
