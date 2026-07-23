### Added

#### Persistent subtitle blacklist with expiry

Blacklist entries now persist across restarts, with their reason and optional
expiry, instead of only flipping a monitored item's status (the old
`AddToBlacklist` built a rich entry then discarded it, and expiry was never
honored). A new `BlacklistStore` optional-capability interface is implemented
across all three database backends (sqlite/postgres/pebble). `IsBlacklisted`
now honors expiry (expired entries no longer count and are language-scoped),
`RemoveFromBlacklist` clears persisted entries, and `CleanupExpiredBlacklist`
actually removes expired entries. Stores that don't implement the capability
fall back to the previous item-status behaviour.

The Pebble implementation uses a bounded prefix scan (adopted from
audiobook-organizer's Pebble store) so listing the blacklist touches only the
blacklist rows rather than the whole keyspace.
