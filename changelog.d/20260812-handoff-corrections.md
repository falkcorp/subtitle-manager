<!-- file: changelog.d/20260812-handoff-corrections.md -->
<!-- version: 1.0.0 -->
<!-- guid: 3ea71c05-9d84-4b26-8f13-6c05a927be40 -->
<!-- last-edited: 2026-08-12 -->

### Changed

#### Next-session handoff brought up to date

Documentation only. Records the five changes merged on 2026-08-12 (pure-Go
SQLite, Settings → Providers, the config-hierarchy guard, the dead Python
removal and automatic bilingual subtitles), and three operational notes worth
not rediscovering:

- **New file operations on a media-derived path need `os.Root` confinement.**
  `security.Validate*` is not recognised by CodeQL as a sanitizer, so passing a
  validated path to `os.Stat`/`os.Open`/`os.Create` still raises
  `go/path-injection`. Design helpers to take a `*os.Root` plus base names.
- **The `github.com/docker/docker` Dependabot alerts are load-bearing.**
  `pkg/transcriber` uses the Docker client to manage Whisper containers, so
  they need a version bump rather than removal.
- **A rebase-merged branch looks unmerged by SHA.** Its commits show as "not in
  origin/main" because they were rewritten, so worktree cleanup must compare
  content rather than the commit list.
