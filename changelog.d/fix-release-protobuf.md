<!-- file: changelog.d/fix-release-protobuf.md -->
<!-- version: 1.0.0 -->
<!-- guid: 39b3f7ac-f9c8-4b56-b211-f83b7f06bf78 -->
<!-- last-edited: 2026-07-25 -->

### Fixed

#### Release workflow no longer regenerates protobuf, unbreaking releases

The `Release` workflow had been failing on every push to `main`, so no release
artifacts were published. Two failures shared one stale assumption — that this
repository generates its protobuf code at release time:

- `Generate Protocol Buffers` failed with
  `protoc-gen-go-grpc@v1.6.2 requires go >= 1.25.0 (running go 1.23.12;
  GOTOOLCHAIN=local)`.
- `Build Go Components` failed with `git is in a dirty state` because the
  generator left an untracked `proto-generated/` directory behind, which
  goreleaser refuses to build over.

Neither step was doing anything useful: the 25 generated `.pb.go` files are
committed under `pkg/`, and `go build ./...` succeeds from a clean checkout with
no generation step. The reusable release coordinator gained a
`protobuf-disabled` input (language detection enables protobuf builds for any
repository containing `.proto` files, which is wrong when the generated code is
committed), and `release.yml` now sets it. `proto-generated/` is also gitignored
so stray generator output can never dirty the tree again.
