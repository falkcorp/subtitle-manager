### Changed

#### Provider buildout notes record the OpenSubtitles work and a stub-shell audit

`docs/PROVIDER_BUILDOUT_PROMPT.md` now records that OpenSubtitles is genuinely
working — the download handshake, file id and `Api-Key` header are all fixed,
and both Bazarr config spellings are read — and documents a measurement worth
acting on: **22 of the 51 hostnames referenced by provider packages do not
resolve**, following the same fabricated `api.<name>.com` pattern as the
`opensubtitlescom` stub.

That matters beyond tidiness: with no provider instances configured,
`FetchFromAll` falls back to every registered provider, so each shell takes a
slot in a fetch wave and costs a DNS failure before real providers are reached.

Deciding which to implement, retire, or repoint is a product call, so the finding
is recorded with its data rather than acted on — a wrong guess would silently
remove a provider someone relies on.
