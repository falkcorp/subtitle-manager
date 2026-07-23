### Changed

#### Automatic monitoring loop uses the full download pipeline

The scheduled monitoring loop now downloads subtitles through
`scanner.ProcessFile` instead of a bare fetch-and-write. This means the
automatic "wanted subtitles" loop gets the same behaviour as manual/CLI
downloads: score-gating, Whisper fallback, post-processing (UTF-8/chmod/
auto-sync/custom script), single-language naming, and score-based upgrades —
previously all bypassed by the monitor's own minimal fetch path.
