<!-- file: changelog.d/feat-wire-profiles-to-scan.md -->
<!-- version: 1.0.0 -->
<!-- guid: c1e94a05-3b72-4d68-8f31-5a06e2c7b943 -->
<!-- last-edited: 2026-08-01 -->

### Fixed

#### Library scans now honor an assigned language profile

Assigning a language profile to a media item had no effect on what was
downloaded. Nothing read the assignment at download time: the web UI and CLI
write path-keyed rows through `database.SubtitleStore`, while the only code
that consumed a profile (`pkg/profiles.Service`, reached from
`providers.FetchWithProfile`) read integer-keyed rows out of a raw `*sql.DB`
after resolving `media_items.path → id` — under the pebble or postgres backends
not even the same database. Separately, `scanner.ProcessFileWithProfile` had no
callers at all; every download path called `scanner.ProcessFile` with a single
language.

A library scan now checks each video file for an assigned profile and, when it
has one, downloads every language that profile asks for in priority order
instead of the single language the scan was started with. Resolution goes
through the same store and the same `filepath.Clean`-ed path key the web
handler writes.

`ProcessFileWithProfile` was rewritten to delegate to `ProcessFile` once per
language rather than fetching and writing inline. Its old implementation
bypassed score gating, the Whisper fallback, UTF-8 re-encoding, chmod,
auto-sync, the custom post-download script, the *arr rescan event and
download-history persistence — everything the ordinary download path does
around the fetch.

Scope is deliberately limited to user-initiated library scans. The monitor
loop, watcher, webhooks, scheduler and Sonarr/Radarr paths still download a
single language, because `MonitoredItem.Languages` is a second, independent
desired-languages mechanism and deciding which one wins is a product decision
rather than a bug fix.

Three behaviours worth knowing:

- A file whose profile resolves to the *default* profile is treated as
  unassigned. `GetMediaProfile` falls back to the default rather than reporting
  a miss, so honouring that would silently switch every file in the library to
  profile-driven downloading as soon as any default existed. The consequence is
  real rather than cosmetic: a file explicitly assigned the default profile is
  indistinguishable from an unassigned one and is scanned with the scan's own
  language, not that profile's language list. Telling the two apart needs a
  store method reporting whether an assignment row exists, separate from which
  profile governs the file. Until then, assign a non-default profile to get
  profile-driven languages.
- With `subtitles.single_language` enabled, only the highest-priority language
  is fetched, since every language would otherwise write to the same
  `video.srt` and overwrite the last.
- Scanning never creates a profile as a side effect. `PebbleStore`'s
  `GetDefaultLanguageProfile` *writes* a profile with ID `default` when the
  store is empty, and `GetMediaProfile` falls back to it on a miss — so
  consulting profiles per file would have made a first scan on a fresh install
  conjure a profile row that then cannot be deleted (`handleDeleteProfile`
  refuses to remove the default). Profile resolution is skipped entirely when
  no profiles exist, and the default is identified by reading the list rather
  than through `GetDefaultLanguageProfile`. That also makes the behaviour
  identical on `SQLStore`, which neither falls back nor creates.
