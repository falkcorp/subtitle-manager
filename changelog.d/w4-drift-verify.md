### Added

#### Subtitle drift verification (`verify`)

Added the `pkg/subsync` package and the `verify [media] [subtitle] [lang]`
command, which check whether a subtitle stays aligned with the audio across a
media file's runtime — detecting both a constant offset and linear speed-up /
slow-down drift (e.g. a 23.976↔25 fps mismatch), including a likely framerate
cause. It samples audio windows, transcribes each with Whisper, measures the
local offset per window, and fits `offset = slope·t + intercept`. This also
wires up the previously-unused `audio.ExtractTrackWithDuration`. The drift
classifier is a pure, fully-tested function; the audio/Whisper measurement is
injectable. See `docs/WHISPER_PIPELINE_DECISIONS.md` (W3+W4).
