<!-- file: changelog.d/fix-duplicate-command-registration.md -->
<!-- version: 1.0.0 -->
<!-- guid: c4e91a37-6b02-4d58-9e13-af7025c8b6d4 -->
<!-- last-edited: 2026-07-30 -->

### Fixed

#### `--help` listed five commands twice

`batch`, `fetch`, `radarr`, `sonarr` and `rename` were registered both in
`cmd/root.go`'s setup block and in their own file's `init()`, so cobra held two
entries for each and printed them twice in the command list.

Nothing was functionally broken, which is why it survived — but a help listing
that repeats itself reads as an unfinished tool, and it is among the first
things anyone sees. The duplicates are removed from `root.go`; each of those
commands keeps the self-registration in its own file, which is the pattern the
other ~25 commands already follow. The five commands that only `root.go`
registers (`convert`, `merge`, `translate`, `queue`, `profile`) stay there.

Found by building the binary and running `--help`, not by reading the code.
Added a test that fails if any command is registered more than once.
