### Fixed

#### Web UI deep links / navigation (SPA routing)

Fixed the web server so client routes (`/library`, `/tools/verify`,
`/settings/...`, etc.) load the correct page. The SPA fallback rewrote unknown
paths to `/index.html`, which `http.FileServer` then 301-redirects to `./` —
bouncing every deep link and in-app navigation back to the dashboard, so no page
but the dashboard ever rendered. The fallback now writes `index.html`'s bytes
directly with a 200.
