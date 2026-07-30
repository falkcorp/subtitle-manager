<!-- file: changelog.d/fix-flaky-context-race-test.md -->
<!-- version: 1.0.0 -->
<!-- guid: 8f2c614b-95a7-4e30-b8d1-c0a3572e94f6 -->
<!-- last-edited: 2026-07-30 -->

### Fixed

#### Flaky data race in the provider context test

`TestProviderContextHandling/context_timeout` reddened CI intermittently on
unrelated pull requests. It was a genuine data race, not a slow-runner
artefact.

The test built a context with `WithTimeout(ctx, 1*time.Millisecond)` and slept
2ms. A *pending* deadline is delivered by the context package's own timer
goroutine, which writes to the context's internals when it fires. testify's
mock formats every argument with `fmt.Sprintf` when recording a call, and
formatting a context reads those same internals by reflection. When the timer
fired while the mock was formatting, the race detector tripped:

```
Write at ... context.(*timerCtx).cancel   <- WithDeadlineCause.func2
Previous read at ... testify/mock.callString -> fmt.Sprintf
```

The 2ms sleep usually let the timer win, which is why this passed locally and
failed only under load.

The fix uses a deadline that has **already passed**. `context.WithDeadline`
cancels synchronously on the calling goroutine in that case, so no timer
goroutine is ever created and nothing concurrent touches the context. Waiting
on `ctx.Done()` was considered and rejected: it shrinks the window but does not
close it, since `cancel()` is still unlocking the context's mutex as the test
proceeds.

The test still asserts what it did before — that a matcher sees an expired
context and the timeout error propagates.

> Honest note: this could not be reproduced locally, including at
> `-count=800` under deliberate CPU contention. The diagnosis and fix come from
> the race detector's stack trace in CI, which names both sides precisely.
