<!-- file: changelog.d/feat-whisper-connect-timeout.md -->
<!-- version: 1.0.0 -->
<!-- guid: 4d70e935-16ba-4c82-9f01-8b3c2a5e7d64 -->
<!-- last-edited: 2026-07-30 -->

### Fixed

#### An unreachable Whisper server stalled transcription for 30 minutes

Reaching the Whisper server and transcribing on it shared a single timeout.
Transcription legitimately takes many minutes, so that budget has to be large —
but it was also what bounded *connecting*. With the server down, misconfigured,
or behind a black-holing firewall, every transcription attempt hung for the
full budget (30 minutes by default) before failing, and a library scan that hit
that path appeared to be frozen rather than erroring.

The two are now separate:

- `whisper.connect_timeout` (default **10s**) bounds establishing the
  connection, so an unreachable server fails in seconds.
- `whisper.transcribe_timeout` (default **30m**, unchanged) bounds the whole
  request, so a running transcription still gets its full budget.

### Decisions worth reviewing

- **`ResponseHeaderTimeout` is deliberately not set.** It looks like the right
  knob for "the server accepted the connection then went quiet", but
  whisper-asr-webservice sends response headers only once transcription has
  finished. Any value short enough to be useful as a liveness check would abort
  valid work. The overall timeout bounds that phase instead.
- **The transport is cloned from `http.DefaultTransport`, not constructed.**
  `pkg/proxy` implements `proxy_url` by setting `Proxy` on the default
  transport. Building a fresh `http.Transport` would silently opt Whisper
  traffic out of the configured proxy — the kind of gap that surfaces as "why
  does everything except transcription go through the proxy?". There is a test
  for this.

### On testing

The behavioural test written first for the connect timeout was **vacuous** and
was thrown away: it filled a listener's accept backlog hoping connections would
hang, but the OS accepted them anyway, so it passed in 0.02s — and kept passing
with the fix reverted. Reaching a genuinely black-holing address would mean
depending on the network from CI, trading one unreliability for another. The
timeout is therefore asserted structurally, where the assertion can actually
fail: a clone of the default transport carries a 10s handshake timeout, so
configuring anything else makes an unapplied setting visible.
