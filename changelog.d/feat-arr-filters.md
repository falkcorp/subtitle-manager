### Added

#### Sonarr/Radarr filtering: monitored-only, tags, series-types, path mappings

Added `pkg/arr` filtering applied when syncing the Sonarr/Radarr library, for
Bazarr parity: `integrations.{sonarr,radarr}.monitored_only` (ingest only
monitored items), `.excluded_series_types` (Sonarr, e.g. `anime`/`daily`),
`.excluded_tags` (tag IDs), and `.path_mappings` (rewrite a path prefix when
*arr and subtitle-manager see files under different roots). Filters are applied
in the client's `Episodes`/`Movies` decode and auto-populated from config in
`Sync`. The zero config is backwards compatible (no filtering).
