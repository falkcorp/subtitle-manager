### Fixed

#### Make the standard Go CI pass (`-race` data race + coverage gate)

The standard CI adopted in #2167 runs `go test -race` and never did before, so it
surfaced two pre-existing issues that blocked Go CI on every Go PR:

- **Data race in `TestQueueProcessJob`** — the test wrote a plain `bool` from the
  queue worker goroutine and read it from the test goroutine after a
  `time.Sleep`, with no synchronization. Replaced with a channel closed on
  execution plus a `select`-with-timeout: race-free and no longer sleep-bound.
- **Coverage gate** — the `ci.yml` caller inherited the standard 80% threshold,
  but the repo is at ~26% and `repository-config.yml` already sets
  `fail_below_threshold: false`. Set `coverage-threshold: '0'` so coverage is
  reported but not gated; raising real coverage is tracked separately.
