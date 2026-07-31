### Added

#### Write downloaded subtitles as WebVTT or ASS, not only SRT

Downloads were always written as `.srt` regardless of what the provider sent or
what the operator wanted. `subtitles.format` now selects the container —
`srt` (default), `vtt` or `ass` — and applies to every subtitle the application
writes: manual downloads, library scans and the monitoring loop all go through
the same path.

Details worth knowing:

- **Format is detected from the bytes.** The download path has an anonymous
  `[]byte` from a provider with no file name to go on, and astisub's readers
  each require the format to be known before they are called, so a payload is
  sniffed (`WEBVTT` signature, `[Script Info]`/`[V4+ Styles]` sections, or a
  `-->` cue arrow) before being parsed.
- **Payloads already in the target format are passed through untouched**
  rather than parsed and re-serialised. A round trip through astisub is lossy
  for anything it does not model, and there is no reason to pay that cost on
  the default path.
- **Bytes are normalised to UTF-8 before parsing.** A legacy-codepage payload
  parsed as UTF-8 round-trips into mojibake that is then written into the new
  container, where nothing downstream can tell it from correct text.
- **A conversion failure falls back to SRT** and the output path is recomputed
  to match, so a `.vtt` never holds SRT bytes and a subtitle astisub cannot
  round-trip is still written rather than dropped.
- **The "already downloaded" check follows the configured extension**, so
  running with `vtt` does not re-download the whole library on every scan.

`postprocess.EncodeUTF8` now delegates to `subtitles.EncodeUTF8`; the behaviour
and the name callers use are unchanged.

The setting is config-file only for now, matching its sibling
`subtitles.single_language`, and the gRPC conversion path remains stubbed.
