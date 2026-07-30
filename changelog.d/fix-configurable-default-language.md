<!-- file: changelog.d/fix-configurable-default-language.md -->
<!-- version: 1.0.0 -->
<!-- guid: 9b3e57a2-4c81-4d06-a9f3-2e70b8146dc5 -->
<!-- last-edited: 2026-07-30 -->

### Fixed

#### Non-English libraries silently got English subtitles

Several paths that need a language but have no profile to consult hardcoded
`"en"`, with no setting anywhere to change it:

- the **Sonarr** and **Radarr** import webhooks — everything imported through
  them was fetched in English;
- **`FetchWithProfile`'s fallback**, taken whenever a media item has no
  language profile assigned, which is the common case since profiles are not
  auto-assigned to new items;
- the same fallback in the tag-filtered variant;
- `GetLanguagesFromProfile`'s error path.

The Plex webhook was the only one with a setting (`webhooks.plex.language`),
and even that fell back to English.

All of these now resolve through `profiles.DefaultLanguage()`:

1. `languages.default` — the explicit setting.
2. The first entry of `languages.preferred`, if configured. Someone who has
   stated an ordered preference has already answered this question; requiring
   them to state it twice is a way to end up with the two disagreeing.
3. `"en"` — unchanged from the previous hardcoded behaviour, so an install that
   configures nothing downloads exactly what it did before.

### What this does *not* fix

The Sonarr and Radarr handlers still use a single default language rather than
the **language profile assigned to that series or movie**, which is what Bazarr
does on import. The handlers receive a file path and have no database handle to
resolve a profile with, so wiring that up means threading a dependency through
the webhook constructors — a larger change than making the language
configurable, and a separate one. This closes the "no setting at all" half of
the gap; the profile-aware half remains open and is recorded in the parity
matrix.

### On testing

The guard in `pkg/webhooks` is a source-level check that no hardcoded language
literal reaches `ProcessFile`, rather than a behavioural test. That is
deliberate: with no provider configured nothing is downloaded whatever language
is requested, so a behavioural assertion in that package passes with the bug
present. The behavioural tests for how the default resolves live in
`pkg/profiles`, where they can actually observe the answer.
