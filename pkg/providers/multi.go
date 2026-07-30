// file: pkg/providers/multi.go
// version: 2.0.0
// guid: 2f8c1a05-6d43-4b79-9e20-3a7c5d8b0164
// last-edited: 2026-07-30

package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/events"
)

// maxConcurrentFetches bounds how many providers are queried at once.
//
// Four is a deliberate middle: the previous behaviour was strictly serial, so
// a library scan paid the full latency of every provider that failed before
// the one that worked — with ~55 registered providers and a 15s per-provider
// timeout, a miss could take many minutes. Firing all of them at once instead
// would open ~55 simultaneous connections from a background scan, which is
// both unfriendly to the services and liable to trip local connection limits.
//
// It is a var rather than a const so tests can pin it to 1 and assert the
// concurrent path selects exactly what the old serial path did.
var maxConcurrentFetches = 4

const (
	// providerTimeout bounds a single provider's Fetch. Unchanged from the
	// serial implementation.
	providerTimeout = 15 * time.Second
	// backoffStep scales the per-instance backoff applied after a failure.
	backoffStep = time.Second
)

// fetchOptions carries the per-caller behaviour that used to be encoded by
// having three near-duplicate copies of the provider loop.
type fetchOptions struct {
	// useBackoff reads and records per-instance backoff. The name-based
	// fallback has no real instance IDs to key it on.
	useBackoff bool
	// publishFailure emits a SearchFailed event when nothing is found.
	//
	// This is not on for every caller, deliberately. SearchFailed reaches
	// pkg/webhooks, which POSTs it to the operator's configured endpoint, and
	// pkg/scanner calls into here once per file — so turning it on for the
	// name-based fallback (the default on a fresh install) would fire one
	// outbound webhook per file of a full library scan. The pre-existing
	// emission points are preserved exactly; see the changelog for the
	// inconsistency this leaves behind.
	publishFailure bool
}

// fetchOutcome is one provider's result, carried back from its goroutine.
type fetchOutcome struct {
	data []byte
	err  error
}

// FetchFromAll tries the known providers until one returns a subtitle,
// querying up to maxConcurrentFetches of them at a time. The provider API key
// is reused when applicable. The name (or instance ID) of the provider that
// succeeded is returned along with the subtitle bytes.
//
// Providers are still consulted in priority order: see fetchFromInstances for
// why concurrency does not mean "whoever answers first wins".
func FetchFromAll(ctx context.Context, mediaPath, lang, key string) ([]byte, string, error) {
	insts := Instances()
	if len(insts) == 0 {
		// No configured instances: fall back to every registered provider
		// name. This is the default path on a fresh install, not an edge
		// case — pkg/scanner reaches it for any library with no provider
		// instances set up.
		//
		// Backoff is deliberately not applied here. The backoff map is keyed
		// by instance ID, and synthesising IDs from bare provider names would
		// start recording state for a path that has never had any, changing
		// which providers a subsequent call even attempts.
		names := All()
		insts = make([]Instance, 0, len(names))
		for _, name := range names {
			insts = append(insts, Instance{ID: name, Name: name, Enabled: true})
		}
		return fetchFromInstances(ctx, insts, mediaPath, lang, key,
			fetchOptions{useBackoff: false, publishFailure: false})
	}
	return fetchFromInstances(ctx, insts, mediaPath, lang, key,
		fetchOptions{useBackoff: true, publishFailure: true})
}

// FetchFromTagged limits provider attempts to instances matching all tags.
func FetchFromTagged(ctx context.Context, mediaPath, lang, key string, tags []string, tm interface {
	FilterByTags(string, []string) ([]string, error)
}) ([]byte, string, error) {
	insts, err := InstancesByTags(tm, tags)
	if err != nil {
		return nil, "", err
	}
	if len(insts) == 0 {
		return nil, "", fmt.Errorf("no subtitle found")
	}
	return fetchFromInstances(ctx, insts, mediaPath, lang, key,
		fetchOptions{useBackoff: true, publishFailure: false})
}

// fetchFromInstances queries insts in priority order, up to
// maxConcurrentFetches at a time, and returns the first subtitle found.
//
// This is the single implementation behind both FetchFromAll and
// FetchFromTagged, which previously carried near-duplicate copies of the loop
// that had drifted apart. Their remaining differences are now explicit in
// opts rather than implicit in three separately-maintained loop bodies.
func fetchFromInstances(ctx context.Context, insts []Instance, mediaPath, lang, key string, opts fetchOptions) ([]byte, string, error) {
	var lastError error

	next := 0
	for next < len(insts) {
		// Assemble the next wave, walking the list in priority order.
		wave := make([]int, 0, maxConcurrentFetches)
		inWave := make(map[string]bool, maxConcurrentFetches)
		for next < len(insts) && len(wave) < maxConcurrentFetches {
			inst := insts[next]
			if opts.useBackoff && IsInBackoff(inst.ID) {
				next++
				continue
			}
			// Two instances of the same provider share one upstream service
			// and one rate limit, so they must not be queried concurrently —
			// the point of running in parallel is to overlap *different*
			// hosts. Stop the wave here and let this instance start the next
			// one, which keeps the overall priority order intact.
			if inWave[inst.Name] {
				break
			}
			inWave[inst.Name] = true
			wave = append(wave, next)
			next++
		}
		if len(wave) == 0 {
			break
		}

		data, id, done, err := runWave(ctx, insts, wave, mediaPath, lang, key, opts.useBackoff)
		if done {
			return data, id, err
		}
		if err != nil {
			lastError = err
		}
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
	}

	if opts.publishFailure {
		errorMsg := "no subtitle found"
		if lastError != nil {
			errorMsg = lastError.Error()
		}
		events.PublishSearchFailed(ctx, events.SearchFailedData{
			Query:     fmt.Sprintf("media:%s lang:%s", mediaPath, lang),
			Language:  lang,
			Error:     errorMsg,
			Timestamp: time.Now(),
		})
	}

	return nil, "", fmt.Errorf("no subtitle found")
}

// runWave queries every instance in one wave concurrently and resolves the
// results in priority order. It reports (data, id, done, err); done is true
// when the caller should stop, either because a subtitle was found or because
// the context ended.
//
// Resolving in priority order rather than completion order is the whole
// correctness question here. Taking whichever provider answers first would
// silently invert the operator's configured priority whenever a lower-ranked
// provider happens to be faster — which is most of the time, since the
// preferred provider is usually the one doing real work. So slot k is only
// consulted once every slot before it has failed.
//
// The cost is that a wave takes as long as its slowest member rather than its
// fastest, bounded by providerTimeout. That is still far better than the
// serial path, which paid every provider's latency plus a growing sleep
// between each one.
func runWave(ctx context.Context, insts []Instance, wave []int, mediaPath, lang, key string, useBackoff bool) ([]byte, string, bool, error) {
	// Nothing to do if the caller already gave up; without this an aborted
	// search would still fire a full wave of provider requests.
	if err := ctx.Err(); err != nil {
		return nil, "", true, err
	}

	// Cancelling the wave stops the losers as soon as a winner is known,
	// rather than leaving them running against a provider nobody is waiting
	// on any more.
	wctx, cancelWave := context.WithCancel(ctx)

	// Deferred before cancelWave so it runs *after* it (defers are LIFO):
	// cancel first, then wait for the goroutines to observe it and exit. No
	// wave outlives the call that started it — otherwise a cancelled search
	// leaves provider requests running in the background, and the goroutines
	// keep touching package state after the caller has moved on.
	var wg sync.WaitGroup
	defer wg.Wait()
	defer cancelWave()

	results := make([]chan fetchOutcome, len(wave))
	for slot, idx := range wave {
		// Buffered so a goroutine whose result is never read still exits
		// instead of blocking forever on the send.
		ch := make(chan fetchOutcome, 1)
		results[slot] = ch
		inst := insts[idx]
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := Get(inst.Name, key)
			if err != nil {
				ch <- fetchOutcome{err: err}
				return
			}
			c, cancel := context.WithTimeout(wctx, providerTimeout)
			defer cancel()
			data, err := p.Fetch(c, mediaPath, lang)
			// A 200 is not a subtitle. Most providers are stubs pointing at
			// hostnames nobody controls, and several answer 200 to anything —
			// which was accepted as a successful download and, worse, ended the
			// search before a real provider was tried. Treat it as a failure so
			// the next provider gets its turn.
			if err == nil && !looksLikeSubtitle(data) {
				err = ErrNotSubtitle
				data = nil
			}
			ch <- fetchOutcome{data: data, err: err}
		}()
	}

	var lastErr error
	for slot, idx := range wave {
		var out fetchOutcome
		select {
		case out = <-results[slot]:
		case <-ctx.Done():
			return nil, "", true, ctx.Err()
		}

		inst := insts[idx]
		if out.err == nil {
			if useBackoff {
				SetBackoff(inst.ID, 0)
			}
			return out.data, inst.ID, true, nil
		}
		lastErr = out.err
		if useBackoff {
			// Derived from the instance's static position in the priority
			// list, not from the order goroutines happened to finish in, so
			// the recorded backoff is deterministic under concurrency.
			SetBackoff(inst.ID, time.Duration(idx+1)*backoffStep)
		}
	}
	return nil, "", false, lastErr
}
