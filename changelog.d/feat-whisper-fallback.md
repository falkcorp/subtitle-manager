### Added

#### Whisper fallback when no subtitle provider has a match

When subtitle providers return nothing for a file, subtitle-manager can now
transcribe the media with Whisper and use that as the subtitle, matching
Bazarr's "Whisper fallback". Enable with `whisper.fallback_enabled`; it uses the
self-hosted `whisper.transcribe_url` (native `/asr`, no key) when configured,
otherwise the OpenAI-compatible API with `openai_api_key`. Disabled by default —
a provider failure is returned unchanged when the fallback is off.
