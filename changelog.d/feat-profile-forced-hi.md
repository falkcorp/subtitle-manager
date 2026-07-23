### Added

#### Honor per-language Forced/HI preferences and profile cutoff score

Language-profile downloads now respect each language's Forced and Hearing
Impaired preferences and the profile's cutoff score, matching Bazarr. When
`scoring.enabled` is set, `FetchWithProfile` performs a score-gated OpenSubtitles
search that maps the language config's Forced/HI flags onto the scoring profile
and gates acceptance on the profile `CutoffScore`; it falls back to the previous
behaviour when scoring is off or no candidate clears the cutoff. Added
`scoring.SelectBestResult`, which returns the best-scoring search result above
the minimum score so the exact chosen candidate is downloaded.
