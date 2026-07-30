<!-- file: changelog.d/feat-provider-instance-config.md -->
<!-- version: 1.0.0 -->
<!-- guid: 5e8b41d0-9c26-4a73-b0f5-3a17e2d6c845 -->
<!-- last-edited: 2026-07-30 -->

### Fixed

#### Provider settings were saved but never applied

`POST /api/providers` wrote `providers.<name>.enabled` into the config file and
persisted it, and **nothing ever read it back**. The provider *instance*
registry — which carries per-provider enablement, priority, tags and throttle
state — had no production caller at all: outside tests, nothing called
`RegisterInstance`, so `Instances()` was empty on every install. Every search
fell through to the flat list of registered provider names, and enabling or
disabling a provider in the UI had no effect on anything.

Configuration is now loaded into the registry when the server starts, and
re-loaded when provider settings are saved, so a change takes effect
immediately rather than at the next restart.

> This is the third instance of the same bug shape in this repository: settings
> that save successfully and are read by nobody. The others were notification
> keys written under a namespace the runtime did not read, and API routes that
> returned `200` from the SPA catch-all having done nothing. A save that
> silently does nothing is worse than an error, because it reports success.

### ⚠️ Behaviour change to be aware of

**If you have explicitly enabled any provider in your config file, search will
now use only the providers you enabled.** Previously that setting was inert and
every registered provider was tried. This is the intended meaning of the
setting, and it matches Bazarr, but it is a real change for an existing config:
a file that enables only `opensubtitles` now searches only `opensubtitles`.

If you have **not** configured providers, nothing changes — see below.

### The safety property

On an install with no provider configuration, loading registers **zero**
instances, leaving searches on the existing fallback across every registered
provider. This is enforced by a test, because the failure would be silent:
registering even one instance flips the search path from "every provider" to
"only the configured ones", and subtitles would simply stop being found with no
error anywhere.

The trap is specific. `cmd/root.go` sets a default of
`providers.embedded.enabled = true`, so on a bare install both `viper.GetBool`
and `viper.IsSet` report `true` for `embedded` while the operator has configured
nothing. A loader keyed on either would register exactly one instance and
collapse search to embedded-only. Enablement is therefore read with
`viper.InConfig`, which reports whether the key came from the config file — the
question actually being asked. That behaviour was verified empirically against
the pinned viper (v1.21.0) rather than assumed.

### Decisions worth reviewing later

- **Search order is deterministic.** Priority is optional in config, so equal
  priorities are the common case; instances live in a map, whose iteration order
  Go randomises per process, and `sort.Slice` is not stable. Ties are broken by
  provider name, so the order is reproducible across restarts instead of
  differing each time. Set `providers.<name>.priority` to override.
- **Reloading replaces the registry rather than merging into it.**
  `RegisterInstance` only adds or updates, so a provider you disabled would
  otherwise linger and keep being searched. A new `SetInstances` swaps the whole
  map.
- **One instance per provider.** Config is keyed by provider name, so it can
  express exactly one instance of each. Several instances of one provider with
  distinct credentials — what the instance layer was originally built for —
  needs a config schema that does not exist yet (a list, presumably). That is a
  schema decision to make deliberately, not to invent here.
- **The fetch path was not touched.** An attractive alternative was to have
  searches try configured instances first and then fall back to the remaining
  provider names, which would preserve reach while honouring priority. That
  needs changes in `pkg/providers/multi.go`, which is rewritten by the parallel
  fetch change, so it was left out to keep the two independent. Worth revisiting
  once both have landed: it would remove the behaviour change flagged above.
