<!-- file: changelog.d/chore-gcommon-bsr-sdk.md -->
<!-- version: 1.0.0 -->
<!-- guid: 40ded265-0b30-4305-94f6-3971174427d8 -->
<!-- last-edited: 2026-07-24 -->

### Changed

#### gcommon protobuf consumption moved to the Buf Schema Registry SDK

subtitle-manager now pulls the generated gcommon protobuf/gRPC code from the
Buf Schema Registry (`buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go` and
`.../grpc/go`) instead of the hand-maintained `github.com/falkcorp/gcommon/v2`
Go module. This is the intended distribution model now that gcommon is a
protos-only repository published to `buf.build/falkcorp/gcommon`. Message and
service types are unchanged; the only difference is that BSR splits messages and
gRPC service stubs into separate packages. No configuration is required — the
BSR modules verify against the public Go checksum database.
