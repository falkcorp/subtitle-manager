// file: pkg/providers/multi_test.go
// version: 2.0.0
// guid: 6b1e9c04-8a37-4d52-9f60-2c5d0a7b3841
// last-edited: 2026-07-30

package providers

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/events"
)

type alwaysFailProvider struct{}

func (alwaysFailProvider) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	return nil, fmt.Errorf("fail")
}

// scriptedProvider is a provider whose latency and outcome the test controls,
// and which records when it starts and stops so concurrency can be observed
// rather than inferred from wall-clock timing alone.
type scriptedProvider struct {
	name  string
	delay time.Duration
	body  []byte // non-nil means success
	obs   *observer
}

func (p scriptedProvider) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	p.obs.enter(p.name)
	defer p.obs.exit()

	select {
	case <-time.After(p.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if p.body == nil {
		return nil, fmt.Errorf("%s: no subtitle", p.name)
	}
	return p.body, nil
}

// observer tracks concurrent Fetch calls and the order they were started in.
type observer struct {
	mu      sync.Mutex
	active  int
	peak    int
	started []string
}

func newObserver() *observer { return &observer{} }

func (o *observer) enter(name string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.active++
	if o.active > o.peak {
		o.peak = o.active
	}
	o.started = append(o.started, name)
}

func (o *observer) exit() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.active--
}

// useInstances replaces the global instance registry for one test and restores
// it afterwards. The registry is package-level shared state, so a test that
// merely adds to it leaks into every later test in the package.
func useInstances(t *testing.T, insts ...Instance) {
	t.Helper()
	instancesMu.Lock()
	prev := instances
	instances = map[string]Instance{}
	for _, i := range insts {
		instances[i.ID] = i
	}
	instancesMu.Unlock()

	backoffMu.Lock()
	prevBackoff := backoffMap
	backoffMap = map[string]time.Time{}
	backoffMu.Unlock()

	t.Cleanup(func() {
		instancesMu.Lock()
		instances = prev
		instancesMu.Unlock()
		backoffMu.Lock()
		backoffMap = prevBackoff
		backoffMu.Unlock()
	})
}

// useFactories replaces the whole provider factory registry for one test.
//
// RegisterFactory has no per-test scope, so by the time a test runs, sibling
// tests have added mock factories that succeed. Any test exercising the
// name-based fallback (which walks the entire registry) is otherwise at their
// mercy — it will "pass" because some unrelated mock returned a subtitle.
func useFactories(t *testing.T, fs map[string]func() Provider) {
	t.Helper()
	factoriesMu.Lock()
	prev := factories
	factories = map[string]func() Provider{}
	for n, f := range fs {
		factories[n] = f
	}
	factoriesMu.Unlock()
	t.Cleanup(func() {
		factoriesMu.Lock()
		factories = prev
		factoriesMu.Unlock()
	})
}

// useConcurrency pins maxConcurrentFetches for one test.
func useConcurrency(t *testing.T, n int) {
	t.Helper()
	prev := maxConcurrentFetches
	maxConcurrentFetches = n
	t.Cleanup(func() { maxConcurrentFetches = prev })
}

// TestFetchFromAllHonorsCancelledContext verifies FetchFromAll returns promptly
// with the context error instead of blocking when the context is already
// cancelled.
func TestFetchFromAllHonorsCancelledContext(t *testing.T) {
	RegisterFactory("failtest", func() Provider { return alwaysFailProvider{} })
	useInstances(t, Instance{ID: "failtest-1", Name: "failtest", Enabled: true})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	start := time.Now()
	_, _, err := FetchFromAll(ctx, "/media/movie.mkv", "en", "")
	if err == nil {
		t.Fatal("expected an error for a cancelled context")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("FetchFromAll did not abort promptly: %v", elapsed)
	}
}

// TestFetchFromAllPrefersHigherPriority is the test that distinguishes a
// correct concurrent implementation from a plausible one.
//
// Running providers concurrently makes it natural to return whichever answers
// first. That would silently invert the operator's configured priority: the
// preferred provider is usually the slower one, because it is the one actually
// searching. Here the high-priority provider takes 150ms and the low-priority
// one returns instantly, so "first success wins" picks the wrong provider
// while every other test in this file still passes.
func TestFetchFromAllPrefersHigherPriority(t *testing.T) {
	obs := newObserver()
	RegisterFactory("slowgood", func() Provider {
		return scriptedProvider{name: "slowgood", delay: 150 * time.Millisecond,
			body: []byte("preferred"), obs: obs}
	})
	RegisterFactory("fastgood", func() Provider {
		return scriptedProvider{name: "fastgood", delay: 0,
			body: []byte("secondary"), obs: obs}
	})

	useConcurrency(t, 4)
	useInstances(t,
		Instance{ID: "slowgood-1", Name: "slowgood", Enabled: true, Priority: 100},
		Instance{ID: "fastgood-1", Name: "fastgood", Enabled: true, Priority: 1},
	)

	data, id, err := FetchFromAll(context.Background(), "/media/movie.mkv", "en", "")
	if err != nil {
		t.Fatalf("FetchFromAll: %v", err)
	}
	if id != "slowgood-1" {
		t.Errorf("winner = %q, want slowgood-1: the faster low-priority provider "+
			"was preferred, which inverts the configured provider order", id)
	}
	if string(data) != "preferred" {
		t.Errorf("data = %q, want the high-priority provider's body", data)
	}
}

// TestFetchFromAllQueriesProvidersConcurrently verifies the wave actually
// overlaps distinct providers.
//
// Asserted on observed concurrent Fetch calls rather than elapsed time: a
// timing-only assertion passes on a fast machine even if the calls are serial.
func TestFetchFromAllQueriesProvidersConcurrently(t *testing.T) {
	obs := newObserver()
	var insts []Instance
	for i := 0; i < 4; i++ {
		name := fmt.Sprintf("concurrent%d", i)
		RegisterFactory(name, func() Provider {
			return scriptedProvider{name: name, delay: 100 * time.Millisecond, obs: obs}
		})
		insts = append(insts, Instance{
			ID: name + "-1", Name: name, Enabled: true, Priority: 100 - i,
		})
	}

	useConcurrency(t, 4)
	useInstances(t, insts...)

	start := time.Now()
	if _, _, err := FetchFromAll(context.Background(), "/m.mkv", "en", ""); err == nil {
		t.Fatal("expected no subtitle found; every provider fails here")
	}
	elapsed := time.Since(start)

	obs.mu.Lock()
	peak := obs.peak
	obs.mu.Unlock()

	if peak < 2 {
		t.Errorf("peak concurrent fetches = %d, want at least 2: providers are "+
			"still being queried one at a time", peak)
	}
	// Four 100ms providers serially is >=400ms plus the old inter-provider
	// sleeps (1s+2s+3s), so ~6.4s. One wave is ~100ms. The bound is loose
	// because it only needs to separate "one wave" from "serial" on a
	// contended CI runner; the peak assertion above is the precise one.
	if elapsed > time.Second {
		t.Errorf("elapsed = %v, want roughly one provider's latency", elapsed)
	}
}

// TestFetchNeverRunsOneProviderConcurrently verifies two instances of the same
// provider are not queried at the same time.
//
// Overlapping distinct providers is fine — they are separate services. Two
// instances of one provider share a single upstream host and rate limit, so
// firing them together turns the speed-up into a 429.
func TestFetchNeverRunsOneProviderConcurrently(t *testing.T) {
	obs := newObserver()
	RegisterFactory("shared", func() Provider {
		return scriptedProvider{name: "shared", delay: 60 * time.Millisecond, obs: obs}
	})

	useConcurrency(t, 4)
	useInstances(t,
		Instance{ID: "shared-a", Name: "shared", Enabled: true, Priority: 30},
		Instance{ID: "shared-b", Name: "shared", Enabled: true, Priority: 20},
		Instance{ID: "shared-c", Name: "shared", Enabled: true, Priority: 10},
	)

	if _, _, err := FetchFromAll(context.Background(), "/m.mkv", "en", ""); err == nil {
		t.Fatal("expected no subtitle found")
	}

	obs.mu.Lock()
	peak := obs.peak
	obs.mu.Unlock()
	if peak > 1 {
		t.Errorf("peak concurrent fetches against one provider = %d, want 1: "+
			"instances of the same provider share a rate limit", peak)
	}
}

// TestFetchSerialEquivalence pins concurrency to 1 and checks the wave path
// selects exactly what the old serial loop did: strict priority order, first
// success wins, and a deterministic per-instance backoff derived from list
// position rather than completion order.
func TestFetchSerialEquivalence(t *testing.T) {
	obs := newObserver()
	RegisterFactory("eq_fail_a", func() Provider {
		return scriptedProvider{name: "eq_fail_a", obs: obs}
	})
	RegisterFactory("eq_fail_b", func() Provider {
		return scriptedProvider{name: "eq_fail_b", obs: obs}
	})
	RegisterFactory("eq_ok", func() Provider {
		return scriptedProvider{name: "eq_ok", body: []byte("ok"), obs: obs}
	})

	useConcurrency(t, 1)
	useInstances(t,
		Instance{ID: "eq_fail_a-1", Name: "eq_fail_a", Enabled: true, Priority: 30},
		Instance{ID: "eq_fail_b-1", Name: "eq_fail_b", Enabled: true, Priority: 20},
		Instance{ID: "eq_ok-1", Name: "eq_ok", Enabled: true, Priority: 10},
	)

	_, id, err := FetchFromAll(context.Background(), "/m.mkv", "en", "")
	if err != nil {
		t.Fatalf("FetchFromAll: %v", err)
	}
	if id != "eq_ok-1" {
		t.Errorf("winner = %q, want eq_ok-1", id)
	}

	obs.mu.Lock()
	order := append([]string(nil), obs.started...)
	obs.mu.Unlock()
	want := []string{"eq_fail_a", "eq_fail_b", "eq_ok"}
	if len(order) != len(want) {
		t.Fatalf("providers queried = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("providers queried = %v, want strict priority order %v", order, want)
		}
	}

	// The two failures must be in backoff; the success must have cleared it.
	if !IsInBackoff("eq_fail_a-1") || !IsInBackoff("eq_fail_b-1") {
		t.Error("failed instances should be in backoff")
	}
	if IsInBackoff("eq_ok-1") {
		t.Error("a successful instance must have its backoff cleared")
	}

	// The durations, not just the fact of a backoff. They are derived from
	// each instance's static position in the priority list precisely so they
	// stay deterministic under concurrency; deriving them from completion
	// order instead would still leave both instances "in backoff" and pass
	// the checks above. The second instance is one backoffStep later than the
	// first, so their deadlines must differ by about that much.
	backoffMu.RLock()
	first, second := backoffMap["eq_fail_a-1"], backoffMap["eq_fail_b-1"]
	backoffMu.RUnlock()
	gap := second.Sub(first)
	if gap < backoffStep/2 || gap > backoffStep*2 {
		t.Errorf("backoff deadlines differ by %v, want about %v: the duration is "+
			"not being derived from the instance's position in the list", gap, backoffStep)
	}
}

// TestFetchSkipsInstancesInBackoff verifies a backed-off instance is passed
// over without occupying a slot in the wave.
func TestFetchSkipsInstancesInBackoff(t *testing.T) {
	obs := newObserver()
	RegisterFactory("bo_skipped", func() Provider {
		return scriptedProvider{name: "bo_skipped", body: []byte("nope"), obs: obs}
	})
	RegisterFactory("bo_used", func() Provider {
		return scriptedProvider{name: "bo_used", body: []byte("yes"), obs: obs}
	})

	useConcurrency(t, 4)
	useInstances(t,
		Instance{ID: "bo_skipped-1", Name: "bo_skipped", Enabled: true, Priority: 99},
		Instance{ID: "bo_used-1", Name: "bo_used", Enabled: true, Priority: 1},
	)
	SetBackoff("bo_skipped-1", time.Minute)

	_, id, err := FetchFromAll(context.Background(), "/m.mkv", "en", "")
	if err != nil {
		t.Fatalf("FetchFromAll: %v", err)
	}
	if id != "bo_used-1" {
		t.Errorf("winner = %q, want bo_used-1; the backed-off instance was queried anyway", id)
	}

	obs.mu.Lock()
	defer obs.mu.Unlock()
	for _, n := range obs.started {
		if n == "bo_skipped" {
			t.Error("an instance in backoff was queried")
		}
	}
}

// TestFetchFromAllFallsBackToRegisteredNames covers the path taken when no
// provider instances are configured, which is the default on a fresh install
// and the one pkg/scanner reaches. Backoff must not be recorded there: the
// backoff map is keyed by instance ID and this path has none.
func TestFetchFromAllFallsBackToRegisteredNames(t *testing.T) {
	useConcurrency(t, 4)
	useInstances(t) // no instances at all

	useFactories(t, map[string]func() Provider{
		"fb_fail": func() Provider { return alwaysFailProvider{} },
	})

	if _, _, err := FetchFromAll(context.Background(), "/m.mkv", "en", ""); err == nil {
		t.Fatal("expected no subtitle found")
	}

	backoffMu.RLock()
	n := len(backoffMap)
	backoffMu.RUnlock()
	if n != 0 {
		t.Errorf("backoff recorded %d entries on the name-based path, want 0", n)
	}
}

// countingPublisher records SearchFailed emissions. Only the methods used here
// do anything; the rest satisfy the interface.
type countingPublisher struct {
	mu     sync.Mutex
	failed int
}

func (c *countingPublisher) PublishSearchFailed(context.Context, events.SearchFailedData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failed++
}
func (c *countingPublisher) PublishSubtitleDownloaded(context.Context, events.SubtitleDownloadedData) {
}
func (c *countingPublisher) PublishSubtitleUpgraded(context.Context, events.SubtitleUpgradedData) {}
func (c *countingPublisher) PublishSubtitleFailed(context.Context, events.SubtitleFailedData)     {}

func (c *countingPublisher) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failed
}

// useEventCounter installs a counting publisher for one test.
func useEventCounter(t *testing.T) *countingPublisher {
	t.Helper()
	c := &countingPublisher{}
	prev := events.GetGlobalPublisher()
	events.SetGlobalPublisher(c)
	t.Cleanup(func() { events.SetGlobalPublisher(prev) })
	return c
}

// TestSearchFailedEmissionIsUnchanged pins which code paths publish a
// SearchFailed event.
//
// This matters more than it looks: SearchFailed reaches pkg/webhooks, which
// POSTs it to the operator's configured endpoint, and pkg/scanner calls
// FetchFromAll once per file. Unifying the three provider loops made it easy
// to accidentally emit from the name-based fallback — the default on a fresh
// install — which would turn one library scan into one outbound webhook per
// file. The emission points are therefore exactly the pre-existing ones.
func TestSearchFailedEmissionIsUnchanged(t *testing.T) {
	RegisterFactory("emit_fail", func() Provider { return alwaysFailProvider{} })

	t.Run("configured instances publish", func(t *testing.T) {
		useConcurrency(t, 4)
		useInstances(t, Instance{ID: "emit_fail-1", Name: "emit_fail", Enabled: true})
		c := useEventCounter(t)

		if _, _, err := FetchFromAll(context.Background(), "/m.mkv", "en", ""); err == nil {
			t.Fatal("expected no subtitle found")
		}
		if got := c.count(); got != 1 {
			t.Errorf("SearchFailed emitted %d times, want 1", got)
		}
	})

	t.Run("name-based fallback stays silent", func(t *testing.T) {
		useConcurrency(t, 4)
		useInstances(t) // none configured -> fallback over registered names
		// Every provider must fail, or the search succeeds and no failure
		// event would be published regardless of the option under test.
		useFactories(t, map[string]func() Provider{
			"emit_fail": func() Provider { return alwaysFailProvider{} },
		})
		c := useEventCounter(t)

		if _, _, err := FetchFromAll(context.Background(), "/m.mkv", "en", ""); err == nil {
			t.Fatal("expected no subtitle found; the fallback registry only fails here")
		}
		if got := c.count(); got != 0 {
			t.Errorf("SearchFailed emitted %d times on the fallback path, want 0: "+
				"a full library scan would fire one webhook per file", got)
		}
	})
}
