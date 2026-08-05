### Fixed

#### Assigning a nonexistent language profile no longer silently succeeds

`GetLanguageProfile` signals "no such profile" differently per backend, and
nothing in the interface said so: the SQL backends return `sql.ErrNoRows` while
`PebbleStore` returns `(nil, nil)`. Every caller was written against the SQL
behaviour — "an error means not found" — so on Pebble, which is the **default**
backend, an unknown profile ID passed validation.

The visible effect: `POST /api/media/profiles/bulk` with a profile ID that does
not exist returned `200` and `{"succeeded":N,"failed":0}`, writing assignments
pointing at nothing. Found by driving a running server rather than by a test —
the existing tests configure SQLite, the one backend where the naive check
happens to work. The single-item assign route, the profile GET route and both
`subtitle-manager profiles` subcommands had the identical check.

`database.ProfileMissing` now encodes the divergence in one place and every
caller asks it. Normalising there rather than changing `PebbleStore` keeps the
blast radius to the callers that ask this question: `GetMediaProfile` returns
`GetLanguageProfile`'s result directly, so making Pebble return an error would
change what a dangling assignment looks like to unrelated callers.

The new regression tests run against Pebble specifically.
