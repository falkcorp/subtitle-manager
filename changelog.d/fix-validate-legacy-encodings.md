### Fixed

#### Stop discarding subtitles that are not encoded in UTF-8

Provider response validation required the downloaded bytes to be valid UTF-8.
Subtitles are routinely distributed in legacy single-byte codepages instead —
Windows-1255 for Hebrew, 1251 for Cyrillic, 1252 for Western European — so
every one of those was rejected as "not a subtitle".

The loss was silent. A rejected response is recorded as a provider *failure*,
so the search moved on to the next provider and nothing was written: a working
provider serving a real subtitle was indistinguishable from a dead one.

Validation now keys on structure rather than character encoding. The markers it
looks for (`-->` for SRT/WebVTT, `[Script Info]`/`[Events]`/`Dialogue:` for
SubStation Alpha) are pure ASCII and survive any 8-bit codepage intact, so
coverage is unchanged; binary payloads are still rejected, now by a NUL-byte
test rather than a UTF-8 validity test. Converting to UTF-8 remains
`postprocess.EncodeUTF8`'s job, downstream of validation and gated on
`postprocess.utf8_encoding` as before.
