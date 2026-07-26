<!-- file: changelog.d/fix-notifications-wiring.md -->
<!-- version: 1.0.0 -->
<!-- guid: d5093e6b-71a4-4c82-9f37-8ae2b04c1d95 -->
<!-- last-edited: 2026-07-26 -->

### Fixed

#### Notification settings were saved where nothing read them

The Notifications settings page saved through `POST /api/config`, which hands
each key to `viper.Set` exactly as received, and the page sent bare keys
(`discord_webhook`). The runtime has always read namespaced ones
(`notifications.discord_webhook`). Nothing read what the page wrote, so
configuring notifications appeared to succeed and then never delivered
anything.

The page now saves dotted keys, which `viper.Set` nests correctly. Settings
saved by an older build are still read as a fallback, so an upgrade does not
silently disable a channel that was working; re-saving the page migrates them.

#### Email notifications could not work at all

`SMTPNotifier` has existed in `pkg/notifications`, with unit tests, since before
this change — but nothing ever constructed one. The runtime only knew about
`notifications.email_url`, a webhook that happens to send mail, while the
settings page collects SMTP host, port, credentials, sender and recipients. The
two halves never met, so the email tab could not work regardless of the key
namespace. SMTP settings now build a notifier, with recipients accepted as a
comma or semicolon separated list.

The SMTP host is deliberately **not** run through `validateWebhookURL`: it is
not a URL, and an operator's own mail relay is usually on a private address,
which that allowlist exists to reject. It is treated as operator-supplied
infrastructure, like the Apprise endpoint.

### Added

#### `POST /api/notifications/test/{type}`

The "Send test notification" buttons have always called this; no handler was
registered, so the request fell through to the single-page-app catch-all and
returned `index.html` with a `200` — the page reported *"notification sent
successfully"* having sent nothing.

It reports what actually happened rather than a blanket success:

- an unconfigured channel is a `400` that says so, not a fake success;
- a provider that rejects the message is a `502`, not a success;
- `push` (Pushover) returns `501`: only the hostname is allowlisted, there is
  no sender implementation;
- `webhook` returns `501` pointing at `/api/webhooks/config` and
  `/api/webhooks/test`, the existing subsystem that already implements generic
  webhooks with methods, headers and retries. That settings tab duplicates it.

Supported channels are `discord`, `telegram`, `email` and `apprise`.

The test uses **saved** configuration, not the values currently in the form, so
settings must be saved before testing. The request carries only a message;
matching Bazarr, which tests live form values, would mean putting credentials in
the request body.

Both the test endpoint and the event publisher build their notification service
through the same helper, so "the test succeeded" and "events are delivered"
cannot disagree — a test button exercising a different code path than production
is worse than no button, because it reports confidence it has not earned.

Delivery goes through `notifications.New`, which enforces HTTPS, rejects private
and link-local addresses, and applies a provider allowlist. Skipping it would
have made this endpoint an SSRF primitive, since it fetches an operator-supplied
URL on demand. There are tests for the link-local metadata address, private
addresses, plain HTTP and non-allowlisted hosts.

#### Settings that are saved but still have no effect

Now that the page writes where the runtime reads, three fields it collects are
persisted but not yet consumed, and are called out rather than left looking
functional:

- `smtp_tls` — `SMTPNotifier` uses `smtp.SendMail`, which negotiates STARTTLS
  opportunistically. The behaviour is right; the switch does not control it.
- `discord_username` and `discord_avatar` — Discord webhooks accept `username`
  and `avatar_url` overrides, but the payload does not send them yet.

Pushover and generic-webhook fields on that page are likewise persisted with no
sender behind them; see the `501` responses above.

### Changed

`Service.Send` no longer stops at the first failing channel. A single broken
webhook previously suppressed delivery to every channel configured after it; all
channels are now attempted and the first error returned. Response bodies are
also drained and closed — three of the four delivery paths leaked their
connection on every notification.
