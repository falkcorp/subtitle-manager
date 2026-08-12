<!-- file: changelog.d/20260812-settings-providers.md -->
<!-- version: 1.0.0 -->
<!-- guid: 5b7e21ca-8d43-4f96-a0c7-3e15904bd782 -->
<!-- last-edited: 2026-08-12 -->

### Fixed

#### Saving any setting could silently collapse subtitle search to one provider

This is the serious one, and it is independent of the UI wiring below.

`viper.WriteConfig()` serialises `viper.AllSettings()`, which merges in defaults,
and `cmd/root.go` set `providers.embedded.enabled` to `true`. So saving a single
unrelated setting — from any settings page — rewrote the operator's config file
with:

```yaml
providers:
    embedded:
        enabled: true
```

On the next start `viper.InConfig` reported true for that key, so
`LoadFromConfig` registered exactly one provider instance. Per the safety
property documented at `pkg/providers/config.go`, registering even one instance
flips `FetchFromAll` out of its name-based fallback and into "only the
configured instances" — cutting search from every registered provider down to
`embedded` alone, which additionally needs ffmpeg to do anything.

That file deliberately chose `InConfig` over `IsSet` to stop a *default* from
enabling a provider, and its comment warns the failure "is invisible until
subtitles stop being found". The protection was being defeated from the other
end: by the config **writer** putting the default into the file, after which
`InConfig` legitimately reported true.

`POST /api/config` now persists only the keys the request actually submitted,
through a fresh viper instance that carries no defaults, preserving whatever
else is already in the file. The dead `providers.embedded.enabled` default is
removed — nothing read it, and the zero-instance fallback already includes
embedded on a bare install, so it was pure downside.

#### Settings → Providers could not enable or configure anything

`Settings.jsx` PATCHed `/api/providers/{name}` and POSTed
`/api/providers/{name}/config`. Neither is a mounted route; both were absorbed
by the `/api/providers/` **subtree** handler and 404'd. Every call site was an
`if (response.ok)` with no `else`, and `ProviderConfigDialog` called `onSave`
and `onClose` back to back — so a failed save closed the dialog and reported
success. Since only `embedded` shipped enabled, there was no route to a working
provider through the UI at all.

Both now POST `/api/config` with the flat dotted keys the server reads
(`providers.<name>.enabled`, `providers.<name>.<field>`). Failures surface: the
toggle no longer moves to a state the server rejected, the dialog stays open,
and the snackbar reports the status code. A save handler that returns nothing
still closes its dialog, so other callers are unchanged.

#### Provider changes needed a restart

`viper.Set` populates only viper's override layer, while provider enablement is
read with `viper.InConfig`, which inspects the file layer — so even a correctly
saved setting stayed invisible to the running process. Nothing re-read the
config or called `LoadFromConfig` after startup.

The config is now re-read after a successful write and the provider registry is
reloaded, so a toggle applies immediately.

### Verification

Browser-driven against a release-style binary (`CGO_ENABLED=0`, no build tags)
serving a real embedded frontend, on a Pebble deployment with a real config file:

- **Add Provider → Gestdown → Save** wrote exactly
  `providers.gestdown.enabled: true` to the config file, preserved the
  pre-existing `db_backend`/`db_path`, and added **no** `embedded` key.
- The Gestdown card appeared on reload with its switch on — the registry picked
  it up **without a restart**.
- Toggling the switch off wrote `enabled: false`. Zero server-side errors
  throughout.

`TestSavingAnUnrelatedSettingDoesNotCollapseSearch`,
`TestEnablingAProviderTakesEffect` and `TestDisablingAProviderTakesEffect`
assert the observable effect — `providers.Instances()` and the bytes on disk —
rather than the response status, because `POST /api/config` returns 204 whether
or not anything reads the key it stored.

The frontend tests assert the observable effect and never `console.error`:
React reports a throw inside an event handler as an unhandled error that Vitest
does not count as a failure. Confirmed non-vacuous by reinstating the
unconditional `onClose`, which fails "stays open when the save fails" while the
success-path tests keep passing. The dialog fixture carries a real `api_key` and
`user_agent`, because `isValid()` would otherwise leave Save disabled and every
assertion would pass for the wrong reason.
