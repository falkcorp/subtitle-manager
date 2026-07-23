### Added

#### Design doc: Whisper-centric subtitle pipeline

Added `docs/WHISPER_PIPELINE_DESIGN.md` — a grounded assessment of what it takes
to deliver the Whisper-centric workflow (library scan, Sonarr/Radarr REST pull,
provider search, user-provided or self-hosted Whisper, subtitle drift
verification, and Whisper-extract → Mandarin translation → Esperanto-tagged
double subs). Includes a current-state capability matrix with file/line
references, six scoped work items with effort sizing, detailed designs for
self-hosted transcription, drift verification, and the double-subs generator,
and a recommended build sequence.
