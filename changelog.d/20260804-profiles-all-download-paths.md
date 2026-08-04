### Changed

#### Language profiles now govern every automated download, not just library scans

An assigned language profile previously affected only a library scan. The
monitor loop, the filesystem watcher, the Sonarr and Radarr webhooks, and
`subtitle-manager sonarr`/`radarr` all called the pipeline with one language and
ignored the assignment. They now download every language the file's profile asks
for, in priority order, through the same pipeline — so scoring, the Whisper
fallback, post-processing and download history apply unchanged.

**The precedence decision.** `MonitoredItem.Languages` is a second, independent
desired-languages mechanism, and something had to win. **The profile wins.**
`MonitoredItem.Languages` is *accumulated* rather than curated — the monitor's
sync code unions in every language it is ever asked for and never removes one,
so it cannot express "stop wanting German". A profile assignment is a deliberate
per-file choice made in the UI, and an append-only accumulation should not
override a deliberate choice.

**What is deliberately left alone:** requests that name a language directly —
the web download endpoint's `?lang=`, the custom webhook's validated `lang`
field, and `fetch <file> <lang>`. Those are someone asking for one specific
thing right now rather than stating a policy about the file, so a profile does
not override them.

A profile-assigned monitored item still retries, fails and auto-blacklists
exactly as before; that bookkeeping was factored into a helper both paths share
rather than duplicated.
