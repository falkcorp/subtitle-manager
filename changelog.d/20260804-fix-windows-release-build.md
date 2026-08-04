### Fixed

#### Releases build again on Windows

Every release since at least 2026-08-03 failed. `pkg/webserver/system.go` called
`syscall.Statfs` unconditionally to report disk usage, and that call does not
exist on Windows, so goreleaser's `windows/amd64` build failed and took the
whole release with it — while CI stayed green, because CI only ever compiled the
runner's own platform.

Disk usage now goes through a small per-platform helper: `statfs` on Unix,
`GetDiskFreeSpaceEx` on Windows. On Windows the free figure is the bytes
available *to the caller* rather than total free bytes; the two differ under
disk quotas, and the former is the more useful answer to "how much room is left
for subtitles". `"/"` is also not a volume on Windows, so the queried root now
comes from `SystemDrive`.

CI gained a `cross-build` job that compiles all four targets goreleaser ships
(linux/amd64, linux/arm64, darwin/arm64, windows/amd64). Without it this class
of break stays invisible until a release fails, which is exactly what happened.
