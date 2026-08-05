### Fixed

#### The `opensubtitlescom` provider was a stub that could never work

It pointed at `api.opensubtitlescom.com` — not a real host; the API lives at
`api.opensubtitles.com` — and issued `GET /subtitles/{name}/{lang}`, an endpoint
that exists in no version of the API. It could never have returned a subtitle,
and because the registry offered it as a separate provider, every fetch wave
tried it, consumed one of the concurrent slots, and failed.

Meanwhile `pkg/providers/opensubtitles` is already a complete REST v1 client. So
`opensubtitlescom` is now a thin wrapper over it rather than a second
implementation. Credentials are read from `opensubtitles.*`, falling back to
`providers.opensubtitlescom.*` — the spelling a Bazarr import produces for this
provider name — with the former winning, so this cannot change a working setup.

The fallback is read inline rather than written back into viper. Provider
constructors run inside the fetch wave's goroutines, so mutating global config
there would be a data race against viper's non-thread-safe map and would
silently rewrite configuration for every other component too.

#### A legacy `opensubtitles.api_url` no longer breaks every request

The client speaks REST v1 only. A config carried over from the XML-RPC or legacy
REST API keeps a host like `rest.opensubtitles.org`, which cannot serve those
requests — verified 2026-08-04: it answers the v1 search path with 400, while
`api.opensubtitles.com/api/v1` answers 403, i.e. exists and merely wants
credentials. Known-legacy hosts are now replaced with the v1 base and a warning
naming the setting. Any other configured value is honoured untouched.
