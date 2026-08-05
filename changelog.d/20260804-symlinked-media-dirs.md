### Fixed

#### A media directory reached through a symlink no longer rejects everything

`ValidateAndSanitizePath` compared literal absolute paths, and `filepath.Abs`
does not resolve symlinks. So when the configured directory and the file were
two spellings of the same place, `filepath.Rel` returned `../..` and every file
under the library was refused with "path not in allowed directories".

This is an ordinary setup rather than an edge case: on macOS `/tmp` is a symlink
to `/private/tmp`, and on Linux a media root at `/media -> /mnt/storage` is
common. Hit for real while assigning a language profile from the CLI.

Both sides are now resolved before comparison. The traversal check still runs
against the literal path and the literal path is still what is returned, so
resolving cannot be used to get past the boundary — a symlink pointing outside
every configured directory is still rejected, and there is a test for it.
