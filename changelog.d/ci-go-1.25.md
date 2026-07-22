### Fixed

#### Bump CI Go toolchain to 1.25 to match `go.mod`

`go.mod` requires `go 1.25.0`, but CI still pinned Go 1.24, so every Go and
Docker build failed with `go.mod requires go >= 1.25.0 (running go 1.24.13)`.
Bumped the Go version to 1.25 in `ci.yml` (`GO_VERSION` and the test matrix),
`release-protobuf.yml`, and the release build matrix emitted by
`.github/scripts/detect_languages.py`.
