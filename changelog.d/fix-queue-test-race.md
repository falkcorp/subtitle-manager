### Fixed

#### Fix data race in `TestQueueProcessJob`

The test signalled job execution by writing a plain `bool` from the worker
goroutine and reading it from the test goroutine after a `time.Sleep`, with no
synchronization — a data race that the standard CI's `go test -race` flags (it
was previously hidden because the old CI never ran the Go tests). Replaced the
bool+sleep with a channel closed on execution and a `select` with timeout,
which is race-free and no longer depends on a fixed sleep.
