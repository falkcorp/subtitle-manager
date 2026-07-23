### Added

#### Notifications now fire on subtitle events, plus Apprise support

Subtitle download / upgrade / failure events are now delivered to the
operator's configured notification channels. Previously the notification
`Service` was only ever invoked on user creation, so Discord/Telegram/email
notifications never fired for actual subtitle activity.

A new `events.MultiPublisher` fans each event out to multiple subscribers, so
the notifications publisher runs alongside the existing webhook publisher
without either owning the global event slot. Per-kind delivery can be disabled
via `notifications.notify_on_download`, `notify_on_upgrade`, and
`notify_on_failure` (all default on).

Adds **Apprise** as a notification target: set `notifications.apprise_url` to a
self-hosted Apprise API notify endpoint (e.g.
`http://apprise:8000/notify/mykey`). Because Apprise is operator-run
infrastructure, its URL is validated leniently and is not subject to the strict
public-webhook allowlist that guards the Discord/email URLs.
