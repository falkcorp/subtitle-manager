<!-- file: changelog.d/feat-arr-rescan.md -->
<!-- version: 1.0.0 -->
<!-- guid: 7d1320d6-a4d0-48f6-9618-8d06952ece1d -->
<!-- last-edited: 2026-07-25 -->

### Added

#### Sonarr/Radarr are told to rescan after a subtitle is written

Previously a downloaded subtitle stayed invisible to Sonarr/Radarr until their
own scheduled disk scan happened to run. subtitle-manager now issues the same
commands Bazarr does as soon as a subtitle is written or upgraded:
`RescanSeries`/`seriesId` and `RescanMovie`/`movieId` on `POST /api/v3/command`.

This is enabled by default for any enabled *arr integration, matching Bazarr,
which notifies unconditionally. Opt out with
`integrations.sonarr.rescan_after_download: false` (likewise `radarr`).

The new `pkg/arrnotify` package plugs into the existing `events.MultiPublisher`
as another subscriber, alongside webhooks and notifications. The *arr item id is
resolved from the media path when the event fires — the *arr's folder listing is
fetched and cached for five minutes, then matched by longest containing folder.
Matching runs in subtitle-manager's path space, applying `Filters.MapPath` to the
*arr's folders rather than trying to invert it, since `MapPath` is
longest-prefix-wins and not reliably invertible. Ambiguous matches are skipped
rather than guessed, and any failure is logged and swallowed so a rescan can
never break the subtitle pipeline that produced it.
