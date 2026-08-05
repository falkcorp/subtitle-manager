### Fixed

#### Bazarr-imported OpenSubtitles credentials are actually used

The Bazarr importer writes provider credentials under
`providers.opensubtitles.username` / `.password` / `.api_key`
(`pkg/bazarr.mapProviderSettings`), but the OpenSubtitles client only ever read
the top-level `opensubtitles.*` spelling. Importing a Bazarr config — the
documented way to migrate — therefore produced a config with a username and
password sitting in it that nothing read, and the provider stayed
unauthenticated with no indication why. Confirmed on a real imported config.

The client now reads `opensubtitles.<key>` and falls back to
`providers.opensubtitles.<key>`. The top-level spelling wins, so a config that
already works cannot be changed by the fallback.
