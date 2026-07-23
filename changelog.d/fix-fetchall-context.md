### Fixed

#### Provider search honors context cancellation during inter-provider delay

`FetchFromAll`'s fallback loop (used when no provider instances are configured)
waited between provider attempts with a blind `time.Sleep` that ignored context
cancellation, so a cancelled or bounded context could not abort it mid-wait —
making automatic monitoring with many stub providers slow to cancel. The delay
is now context-aware (a `select` on `ctx.Done()`), matching the instance-based
loops.
