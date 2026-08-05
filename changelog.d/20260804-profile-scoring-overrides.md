### Fixed

#### A language profile's cutoff score and Forced/HI flags now reach the scorer

A language profile carries a `CutoffScore` and per-language `Forced`/`HI`
preferences, and none of them affected anything. The download path scored every
candidate with the global `scoring.*` settings, so a profile saying "Spanish,
forced, minimum score 90" was honoured only in its choice of language — the
threshold and the flags were dropped silently, which made profiles look far more
configurable than they were.

The file's assigned profile is now layered over the global scoring profile:
`CutoffScore` replaces the minimum score, and a language marked `Forced` or `HI`
turns on the matching allow/prefer flags for that language only.

Two deliberate limits. A *false* `Forced`/`HI` is not treated as a prohibition —
the flags mean "this is preferred", not "reject everything else", so turning one
off returns to the global policy rather than narrowing to nothing. And a file
with no assignment is scored exactly as before, so nothing changes for anyone
not using profiles.
