### Added

#### Configurable outbound HTTP proxy

Set `proxy_url` (e.g. `http://proxy.local:3128`) to route all outbound HTTP
traffic — subtitle providers, Sonarr/Radarr, Whisper, notifications — through a
proxy, matching Bazarr's proxy setting. It is applied to `http.DefaultTransport`
at startup, so every client that uses the default transport (all of them) picks
it up. When `proxy_url` is unset, the standard `HTTP_PROXY`/`HTTPS_PROXY`/
`NO_PROXY` environment variables continue to apply.
