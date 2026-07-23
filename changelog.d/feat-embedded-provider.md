### Added

#### Real embedded-subtitle provider (extract tracks from the container)

The `embedded` provider now extracts subtitle tracks already muxed into the
media file via `ffmpeg`, matching Bazarr's "use embedded subtitles" feature,
instead of the previous stub that hit a fake HTTP endpoint. It probes streams
with `ffprobe` (`pkg/video.SubtitleStreams`), skips image-based tracks
(PGS/VobSub/DVD) that cannot be converted to text, selects the track whose
language matches the request (tolerating ISO 639-1 vs 639-2, e.g. `en` vs
`eng`), and returns it as SRT. When the language is unconstrained and exactly
one text track exists, that track is used.
