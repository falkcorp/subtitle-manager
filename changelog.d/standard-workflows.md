### Changed

#### Convert CI and release workflows to the ghcommon standard

Replaced the old self-contained workflow fleet (`release-go.yml`,
`release-python.yml`, `release-protobuf.yml`, `release-docker.yml`,
`release-frontend.yml`, `release-rust.yml`, `build-assets.yml`, and the bespoke
`ci.yml`/`release.yml`) with the house-standard thin callers (`ci.yml`,
`release.yml`) that delegate to the shared reusable workflows (`reusable-ci.yml`,
`reusable-release.yml`) and the published `falkcorp/gha-*` actions. This is
language-detecting, keeps all build logic in one central place, and pins the Go
toolchain to 1.25 to match `go.mod`. Also refreshed the synced
`sync-receiver.yml` and `commit-override-handler.yml` to their current ghcommon
versions. (Security-workflow standardization is deferred to a follow-up because
the repo currently uses CodeQL default setup, which conflicts with a custom
CodeQL workflow.)

### Fixed

#### Repair the Python build

Fixed `requirements.txt` so it installs cleanly under strict pip resolution
(the old CI ran `pip install` via `os.system`, silently ignoring failures):
dropped the dead `gcommon` git pin (unfetchable ref, not imported by any Python
code), bumped `cffi` to 2.0.0 to satisfy `cryptography==48.0.1` (`cffi>=2.0.0`),
and removed unused security-key/smartcard freeze cruft — `fido2` /
`yubikey-manager` (capped `cryptography<45`) and `pyscard` (fails to build
without PC/SC system headers); none are imported by any Python code. Added a root `pytest.ini` that
excludes the out-of-band Selenium E2E suite (`tests/e2e/`) from standard CI
collection, so both `pip install -r requirements.txt` and pytest succeed.
