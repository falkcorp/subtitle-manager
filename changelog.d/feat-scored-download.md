### Added

#### Score-gated auto-download and score-based upgrades

The subtitle scanner now uses the quality-scoring engine (`pkg/scoring`) to pick
which subtitle to download, matching Bazarr's scoring behaviour. When
`scoring.enabled` is set and the provider supports it (OpenSubtitles today), the
scanner searches for candidates, scores each against the media, and downloads
the highest-scoring candidate **that clears `scoring.min_score`** — downloading
nothing when no candidate is good enough, instead of grabbing the first result.
Upgrades now compare the new candidate's score against the score persisted for
the previously downloaded subtitle (`DownloadRecord.MatchScore`) rather than
merely comparing file sizes. The feature is off by default; existing behaviour
is unchanged when `scoring.enabled` is false.

### Fixed

#### `fetch-scored` now downloads the selected candidate

The `fetch-scored` command scored candidates but then downloaded the *first*
search result regardless of the winner, making the scoring decorative. It now
downloads the specific selected candidate via the provider's new
`FetchByResult`.
