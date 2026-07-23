### Added

#### Real napiprojekt subtitle provider (keyless)

The `napiprojekt` provider is now a genuine integration instead of a stub. It
identifies media by napiprojekt's hash (MD5 of the first 10 MiB plus the
service's fixed digest transform) and downloads the matching subtitle from
`napiprojekt.pl`'s anonymous `dl.php` endpoint — the same keyless, hash-based
protocol subliminal and Bazarr use. No account or API key is required. A `NPc`
response is treated as "no match".

`docs/BAZARR_PARITY_STATUS.md` now documents the provider boundary: which of the
remaining stubs are keyless (implementable), credential-gated (need operator
accounts/keys), or site-scraping only.
