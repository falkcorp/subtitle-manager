### Added

#### Plex webhook receiver

New incoming webhook endpoint `POST /api/webhooks/plex` accepts Plex webhooks
and, on a `library.new` event (new media added to a Plex library), fetches
subtitles for the newly-added file. Plex posts the event as the `payload` field
of a multipart/form-data request; a raw JSON body is also accepted. Events other
than `library.new` are acknowledged and ignored. The subtitle language defaults
to English and can be set with `webhooks.plex.language`.
