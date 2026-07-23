### Fixed

#### Honor `whisper.transcribe_url` for an external self-hosted Whisper server

Pointing `whisper.transcribe_url` at an already-running
whisper-asr-webservice instance now actually works. Previously the key was
read by nothing: transcription only used the OpenAI-compatible API or an
app-managed Docker container, so a configured external server was silently
ignored and the web "transcribe" flow failed with "OpenAI API key not
configured and container not available".

`WhisperTranscribe` now prefers `whisper.transcribe_url` when set — speaking
the webservice's native `/asr` protocol (no API key required) — so every
transcription entry point (CLI, sync, verify, and the web endpoint) uses the
external server. When the URL is unset, behaviour is unchanged. An optional
`whisper.transcribe_timeout` (seconds) overrides the default 30-minute client
timeout.

`whisper.transcribe_model` is now wired to the OpenAI-compatible transcription
path via `SetWhisperModel`. Note: the self-hosted `/asr` server's model is
fixed at its own startup and cannot be selected per request, so this setting
does not affect the `whisper.transcribe_url` path.
