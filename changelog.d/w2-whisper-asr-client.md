### Fixed

#### Real self-hosted Whisper transcription (ASR webservice client)

Added `transcriber.ASRTranscribe`, a client that speaks the native `/asr`
protocol of the self-hosted `onerahmet/openai-whisper-asr-webservice` image
(multipart `audio_file` upload, SRT/VTT/JSON output), and rewired the container
transcription path to use it. Previously the container path was a stub that
pointed the OpenAI SDK at the container with a dummy key (wrong protocol), had a
base-URL restore bug, and discarded the result — so "start our own Whisper"
never produced a subtitle. Both the container and external-API paths now write
the transcript sidecar (`<media>.<lang>.srt`). See
`docs/WHISPER_PIPELINE_DECISIONS.md` (W2).
