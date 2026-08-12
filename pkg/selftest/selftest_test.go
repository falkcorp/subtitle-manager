// file: pkg/selftest/selftest_test.go
// version: 1.1.0
// guid: f1e2d3c4-b5a6-789b-c0d1-e2f3a4b5c6d7
// last-edited: 2026-08-12

package selftest

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/testutil"
)

// TestStartPeriodicSuccess ensures the self test runs without exiting when the database is healthy.
func TestStartPeriodicSuccess(t *testing.T) {
	db := testutil.GetTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exitCalled := false
	orig := ExitFunc
	ExitFunc = func(code int) { exitCalled = true }
	defer func() { ExitFunc = orig }()

	StartPeriodic(ctx, db, "10ms")
	time.Sleep(20 * time.Millisecond)
	if exitCalled {
		t.Fatalf("exit should not be called on healthy db")
	}
}

// TestStartPeriodicShutdownIsNotAFailure ensures a cancelled context is treated
// as shutdown rather than as a dead database.
//
// The self-test goroutine outlives the call that started it. On shutdown the
// context is cancelled and the database is closed, so the next tick's ping
// fails with "sql: database is closed" — and the goroutine used to call
// ExitFunc(1) for it. In production that turns a clean shutdown into exit
// status 1, which a supervisor reads as a crash and may restart-loop on.
//
// It also made this package's own tests exit the test binary: the deferred
// restore of ExitFunc puts the real os.Exit back while the goroutine is still
// running, so the ping-after-close called it for real. That shows up as a bare
// "FAIL" with no --- FAIL line, because nothing failed — the process died.
func TestStartPeriodicShutdownIsNotAFailure(t *testing.T) {
	db := testutil.GetTestDB(t)

	ctx, cancel := context.WithCancel(context.Background())

	var mu sync.Mutex
	exits := 0
	orig := ExitFunc
	ExitFunc = func(code int) { mu.Lock(); exits++; mu.Unlock() }
	defer func() { ExitFunc = orig }()

	StartPeriodic(ctx, db, "5ms")

	// Shut down the way the server does: cancel, then release the database.
	cancel()
	db.Close()
	time.Sleep(60 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if exits != 0 {
		t.Errorf("ExitFunc called %d times during shutdown; a clean shutdown must not report a crash", exits)
	}
}

// TestStartPeriodicFailure ensures the process exits when the database ping fails.
func TestStartPeriodicFailure(t *testing.T) {
	db := testutil.GetTestDB(t)
	db.Close() // force ping failure

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	called := make(chan int, 1)
	orig := ExitFunc
	ExitFunc = func(code int) { called <- code }
	defer func() { ExitFunc = orig }()

	StartPeriodic(ctx, db, "10ms")
	select {
	case <-called:
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("exit not called on failure")
	}
}
