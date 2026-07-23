### Added

#### Download-history retention

Set `history.retention_days` to automatically prune download-history records
older than the given number of days, matching Bazarr's history retention. A
background job (interval configurable via `history.prune_frequency`, default
daily) deletes expired records; it runs only when `history.retention_days` is
positive, so history is kept forever by default. Works across all database
backends (sqlite/postgres/pebble) via the existing store interface.
