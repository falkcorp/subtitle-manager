### Fixed

#### `profiles list` printed IDs that `profiles assign` rejects

The list truncated each profile ID to its first 8 characters, but `assign`,
`remove` and `delete` all take an exact ID. So the obvious workflow — list a
profile, copy its ID, assign it — failed with `profile <prefix> not found`, and
there was no command that printed a usable ID at all. The full ID is now shown.

#### A relative `media_directory` rejected every file under it

`ValidateAndSanitizePath` compares absolute paths, so a configured
`media_directory: ./media` could never match and every file beneath it was
refused with "path not in allowed directories".

This was masked until now by the unconditional temp-directory escape hatch,
which accepted such paths anyway whenever the library happened to sit under
`/tmp`. Closing that hatch in production exposed the underlying bug, which had
been there all along. Configured base directories are now resolved to absolute
paths before comparison — which also means the boundary is enforced rather than
accidentally bypassed.

Both were found by running the CLI end to end rather than by the test suite.

#### `fetch --profile` crashed on the default backend

It opened its own SQLite store with `database.OpenSQLStore` regardless of the
configured backend, so on Pebble — the default — it died with
`unable to open database file: is a directory`. It also read the second,
integer-keyed `media_profiles` implementation, which is not where the web UI or
the CLI write assignments, so even on SQLite it consulted the wrong table.

It now routes through `scanner.ProcessWithProfileIfAssigned`, the single place
that resolves an assigned profile — so it works on any backend, downloads every
language the profile asks for in priority order, and applies the profile's
cutoff score and Forced/HI flags. None of that happened before.

Verified end to end: a two-language profile assigned through the CLI now
produces correctly named per-language subtitle files on a Pebble store.
