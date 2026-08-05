### Fixed

#### Sonarr and Radarr settings now mean the same thing everywhere

There were two independent *arr config schemes and neither read the other:

- `integrations.sonarr.*` / `integrations.radarr.*` — read by the web server's
  background sync, the wanted list, *arr rescan notifications, and both clients'
  HTTP timeouts.
- `sonarr_url` / `sonarr_api_key` (and the Radarr equivalents) — read by
  `subtitle-manager monitor`, `sonarr-sync` and `radarr-sync`.

So configuring the app through the settings UI or the Bazarr importer — both of
which write `integrations.*` — left the monitor loop and the sync commands
believing nothing was configured, while configuring the flat keys left the web
server blind. Each half looked correct on its own, which is exactly how the
notifications config bug behaved.

`pkg/arr.Resolve` is now the single source of truth. `integrations.*` wins: it
is what most of the codebase already read, what the Bazarr importer writes, and
the only one of the two that can express `ssl`, `base_url` and `timeout`. The
flat keys still work so existing configs keep running, but they now log a
deprecation notice naming the key to move to.

**Behaviour change on upgrade:** because every caller resolves through one place,
a config that only has the flat keys now also drives the web server's background
sync, which previously ignored it. An `integrations.<service>` block with a host
set but `enabled: false` stays an explicit opt-out and is never resurrected by
stale flat keys.

Also fixed: the Radarr sync interval was read from `sync_interval` while
Sonarr's came from `episode_sync_interval`, so setting the obvious-looking
`integrations.sonarr.sync_interval` was silently ignored and fell back to 60
minutes. Both spellings are now accepted for both services.
