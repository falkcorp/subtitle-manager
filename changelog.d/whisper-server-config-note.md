### Changed

#### Document Whisper-server config convention (address kept out of the repo)

Added a deployment note to `docs/WHISPER_PIPELINE_DECISIONS.md`: the internal
Whisper server address lives only in the uncommitted `~/.subtitle-manager.yaml`
(`whisper.transcribe_url`), never in the repo. Also records that the discovered
internal server is a custom text-only `/transcribe` endpoint (no timestamps), so
the timed features (drift verification, sync) should point at an SRT-returning
Whisper (OpenAI-compatible or the ASR webservice), while the text-only server
suits extraction/translation.
