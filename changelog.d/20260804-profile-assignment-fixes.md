### Fixed

#### Three language-profile bugs, all of which silently ignored what you asked for

**A file assigned the *default* profile was treated as unassigned.** Assignment
used to be inferred from "the profile this file resolves to is not the default",
because `GetMediaProfile` falls back to the default on a miss and so can never
report one. Assigning the default profile therefore did nothing — the file was
scanned with the scan's own language instead of that profile's language list.
A new store method, `GetAssignedProfileID`, answers whether an assignment row
exists separately from which profile governs the file.

**The last language profile could not be deleted.** The delete guard asked
`GetDefaultLanguageProfile` whether the target was the default, but that helper
answers "which profile governs by default" and falls back to the *first* profile
when none is flagged. So the last remaining profile always came back as the
default and was always refused — there was no sequence of requests that emptied
the list — and with several profiles and no default set, an arbitrary one was
permanently undeletable. The guard now reads the `IsDefault` flag, and deleting
the only profile is allowed since there is nothing left to fall back to anyway.

**Deleting the last profile did not stick.** `PebbleStore`'s
`GetDefaultLanguageProfile` *created* a profile when the store was empty, and
`GetMediaProfile` called it on every miss — so a delete lasted only until the
next lookup, which wrote the profile back, flagged default, and therefore
undeletable again. Reads no longer create rows, matching `SQLStore`, which never
did.

Also: `subtitle-manager profiles` opened its store with the backend hardcoded to
`pebble`, ignoring `db_backend`, so on a SQLite or Postgres deployment the CLI
and the web UI were reading and writing two different databases.
