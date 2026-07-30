<!-- file: changelog.d/fix-provider-status-real.md -->
<!-- version: 1.1.0 -->
<!-- guid: 3a9d61b8-52c7-4e0f-8b34-7d1e05f4a2c9 -->
<!-- last-edited: 2026-07-30 -->

### Fixed

#### `/api/providers/status` reported invented data

The endpoint was backed by a cache that could only ever hold fiction.
`providers.Refresh` did not check anything — it marked every provider name it
was handed as `available: true` — and the one caller passed it a hardcoded
`[]string{"opensubtitles", "subscene"}`. Nothing scheduled it, so the endpoint
returned `{}` until someone POSTed to `/api/providers/refresh`, and after that
it reported exactly two providers as healthy: one real, and one
(`subscene`) that is a non-functional stub which has never been able to return a
subtitle. Every genuinely working provider — `napiprojekt`, `gestdown`,
`podnapisi`, `embedded` — was absent.

Status is now **computed on read** from state the process already holds, so
there is nothing to refresh and nothing to schedule:

- `enabled` — from the provider's configuration.
- `available` — whether a search would consult it *right now*: enabled and not
  currently throttled. It deliberately does **not** mean "known to work".
- `throttled` / `retry_after` — from the existing backoff map.
- `last_success` / `last_failure` — from fetches that actually happened.

`last_success` is what distinguishes a working provider from a stub, and it
needs no hand-maintained list of "real" providers to do it: a stub simply never
succeeds. A second copy of that list in Go would have gone stale the first time
someone implemented a provider and forgot to update it.

Outcomes are recorded in `SetBackoff`, which both fetch loops already call —
zero after a provider returned a subtitle, non-zero after one failed. No change
to the fetch path was needed.

### Changed

#### The two reset endpoints now do different, real things

- `POST /api/providers/refresh` clears provider **throttling**, so backed-off
  providers are retried immediately. This mirrors Bazarr's "reset provider
  throttling" and is the useful action that remains once availability is
  computed rather than cached. It is what an operator wants after fixing
  credentials.
- `POST /api/providers/reset` clears the recorded **success/failure history**.

Keeping these separate matters: conflating them would mean an operator cannot
retry a throttled provider without also erasing the record of why it was
throttled.

Response codes are unchanged (`202` and `204`).

### Unset times are omitted from the JSON, not zero-valued

`last_success`, `last_failure` and `retry_after` are pointers so that "never
happened" is an absent field rather than `0001-01-01T00:00:00Z`.

`encoding/json` does **not** apply `omitempty` to structs, so a plain
`time.Time` is always serialised. With the obvious field types, every one of the
52 providers shipped a `last_success` — and the field whose entire purpose is to
separate a working provider from a stub was present on all of them and
distinguished nothing. The Go-level tests asserted `IsZero()` and passed
throughout; this was only visible by querying a running server. There is now a
test that asserts on the marshalled JSON.

### Known issue this surfaced

**Nothing in the running server calls `providers.RegisterInstance`.** The
provider *instance* layer — multiple configured instances of a provider, each
with its own credentials, priority, tags and throttle state — is populated only
by tests. On a real deployment `Instances()` is always empty, so:

- searches always take the name-based fallback over the registered provider
  names, and per-instance priority, tags and backoff never apply; and
- a status implementation keyed purely on instances would have returned `{}` on
  every real install.

`List` therefore falls back to reporting the registered provider names, which is
what a search on that deployment would actually consult. That keeps this change
honest, but it is a workaround: the instance layer being unreachable is a
separate gap, and wiring provider instances to configuration is its own piece of
work. Flagged here rather than fixed quietly, because it also explains why
per-provider configuration in the UI cannot currently take effect.
