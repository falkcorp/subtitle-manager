### Changed

#### Convert CI/release/security workflows to the ghcommon standard

Replaced the old self-contained workflow fleet (`release-go.yml`,
`release-python.yml`, `release-protobuf.yml`, `release-docker.yml`,
`release-frontend.yml`, `release-rust.yml`, `build-assets.yml`, and the bespoke
`ci.yml`/`release.yml`) with the house-standard thin callers (`ci.yml`,
`release.yml`, `security.yml`) that delegate to the shared reusable workflows
(`reusable-ci.yml`, `reusable-release.yml`, `reusable-security.yml`) and the
published `falkcorp/gha-*` actions. This is language-detecting, keeps all build
logic in one central place, and pins the Go toolchain to 1.25 to match `go.mod`.

### Fixed

#### Repair the Python build

Dropped the dead `gcommon` git pin from `requirements.txt` (the referenced ref
no longer exists and no Python code imports it) and added a root `pytest.ini`
that excludes the out-of-band Selenium E2E suite (`tests/e2e/`) from standard
CI collection, so `pip install -r requirements.txt` and pytest both succeed.
