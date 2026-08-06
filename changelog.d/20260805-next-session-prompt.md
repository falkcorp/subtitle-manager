### Changed

#### Next-session prompt rewritten around what is actually unproven

`docs/NEXT_SESSION_PROMPT.md` now leads with the biggest gap in confidence: the
web UI has only ever been exercised with curl against its API. No page has been
rendered or clicked, so scanning, Mass Edit and the rest are unverified as a
user experiences them.

It also records findings that save the next session from repeating work:

- An OpenSubtitles API key is **mandatory and cannot be harvested**. Verified
  live: 403 without the header. Bazarr stores only a username and password and
  embeds its own key in its source; the 32-char keys in the *arr configs are
  Sonarr's and Radarr's own.
- "Multiplex multiple subtitles" is three different features — separate files
  per language (works), one bilingual file (`cmd/dualsub.go` exists but is
  unwired), or muxing into the container (nothing). Needs a decision, not a
  guess.
- GitHub OAuth already exists; Cloudflare Access JWT verification is entirely
  absent, with a note that the obvious implementation mistake is trusting the
  header when verification is unconfigured.
- A Bazarr API *compatibility layer* does not exist — the current Bazarr
  endpoints are config import only.

Plus ZFS tuning as a measure-first investigation, and the process lessons from
2026-08-04, including never chaining verification with publication.
