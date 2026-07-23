### Added

#### Configurable Sonarr/Radarr request timeout

The Sonarr and Radarr API clients' request timeout is now configurable via
`integrations.sonarr.timeout` and `integrations.radarr.timeout` (in seconds),
matching Bazarr. It defaults to 30 seconds when unset, so behaviour is unchanged
unless configured.
