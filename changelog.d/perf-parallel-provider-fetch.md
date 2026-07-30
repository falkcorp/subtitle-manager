<!-- file: changelog.d/perf-parallel-provider-fetch.md -->
<!-- version: 1.0.0 -->
<!-- guid: 6c623f40-2b17-4636-b077-633333e6a190 -->
<!-- last-edited: 2026-07-30 -->

### Changed

#### Providers are now queried in parallel

Subtitle search consulted providers strictly one at a time, waiting up to 15s
for each, and then *sleeping* an extra second per provider already tried before
starting the next. A search that had to pass ten providers before finding a hit
spent most of its time asleep; against the full registry a miss could take many
minutes. This is the main reason library scans felt stalled.

Providers are now queried in waves of four concurrently, and the inter-provider
sleep is gone entirely. It was never load protection — consecutive requests go
to *different* services — and per-provider retry pressure is already handled by
the existing backoff map.

Decisions worth reviewing later:

- **Results are resolved in priority order, not completion order.** Returning
  whichever provider answers first would be faster still, but it silently
  inverts the configured provider priority: the preferred provider is usually
  the slower one, because it is the one actually searching. A wave therefore
  takes as long as its slowest member rather than its fastest. If you would
  rather have raw speed than honour priority, this is the knob to change.
- **Wave size is four.** Large enough to matter, small enough that a background
  scan does not open ~55 simultaneous connections. It is a package-level `var`,
  so it can be made configurable if that turns out to be the wrong number.
- **Two instances of the same provider are never queried at once.** Distinct
  providers are distinct services, but two instances of one provider share an
  upstream host and rate limit, so overlapping them trades latency for 429s.
- **The no-instances fallback path is parallelised too, but records no
  backoff.** That path (the default on a fresh install, and the one
  `pkg/scanner` takes) identifies providers by bare name; the backoff map is
  keyed by instance ID, so writing to it there would invent state a code path
  that never had any, changing which providers later calls even attempt.

### Unchanged on purpose

The three provider loops are now one implementation, but which of them emits a
`SearchFailed` event is **exactly as before**: only a search over configured
provider instances does. The name-based fallback and tag-filtered searches stay
silent.

That inconsistency looks like a bug worth fixing while the code is open, and it
deliberately was not. `SearchFailed` reaches `pkg/webhooks`, which POSTs it to
the operator's configured endpoint, and `pkg/scanner` calls into here **once per
file**. Emitting from the fallback path — the default on a fresh install — would
turn a single scan of a 10,000-file library into 10,000 outbound webhooks. That
is a real behavioural change that deserves its own decision, not a side effect of
a performance change. There is a test pinning the current emission points so the
question has to be answered deliberately.

### Fixed

#### Data race in the provider registry

The `factories` map was unguarded, which was safe only because providers were
consulted serially. Querying them concurrently makes `Get` run from several
goroutines at once, and `RegisterFactory` writes the same map. It is now behind
a `RWMutex`. This was latent before the change rather than caused by it, but
parallel fetching is what makes it reachable.

#### Provider requests could outlive a cancelled search

Provider calls in a wave are now waited on before the wave returns, so a
cancelled or completed search leaves nothing running in the background against
providers no caller is waiting on.
