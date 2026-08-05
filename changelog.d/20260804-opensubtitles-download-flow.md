### Fixed

#### OpenSubtitles downloads now actually work

The client could search but never download. Three separate defects on the
download path, all confirmed against the published REST v1 contract rather than
inferred:

- **The download step skipped the API's handshake.** It issued
  `GET {api}/download?file_id=...` and returned that response body verbatim. The
  v1 API does not serve subtitles from `/download`: it takes `POST /download`
  with a JSON body `{"file_id": N}` and answers with `{"link": "..."}`, which the
  caller then fetches. What landed on disk was the JSON envelope at best.
- **The wrong identifier was sent.** The file id lives at
  `attributes.files[].file_id`; the client used `attributes.subtitle_id`. Those
  are different values, so even a correct request would have asked for a file
  that does not exist.
- **The `Api-Key` header was never sent.** The v1 API requires it alongside the
  bearer token. `opensubtitles.api_key` was read by the CLI and then never
  given to the client, which set the header to the empty string on login and
  omitted it everywhere else.

Quota exhaustion is now reported as an error too. A depleted account answers
`200` with a message and no link, which the old code would have written to the
media directory as if it were a subtitle.

The existing tests hid all of this: the mock served the file directly from
`GET /download`, so it modelled a protocol nothing speaks, and the
`FetchByResult` test set only `subtitle_id`. Both are corrected, and the new
mock rejects anything that departs from the documented contract.
