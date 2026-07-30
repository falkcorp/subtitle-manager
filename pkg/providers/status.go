// file: pkg/providers/status.go
// version: 2.0.0
// guid: f23c08ee-e040-4796-b45c-5b550e7b8282
// last-edited: 2026-07-30

package providers

import (
	"sync"
	"time"
)

// Status describes what is known about one provider instance.
//
// Every field is derived from state the process already holds — the instance
// registry, the backoff map, and the outcome of fetches that have actually
// happened. Nothing here is probed.
//
// The previous implementation stored the result of a "check" that did not
// exist: Refresh marked every name it was given as Available regardless of
// reality, and it was called with a hardcoded list of two providers. The status
// map was therefore empty until someone POSTed to /api/providers/refresh, and
// then reported exactly two providers as healthy — one of which (subscene) is a
// non-functional stub. Reporting nothing would have been more informative.
type Status struct {
	// Name is the provider name, e.g. "opensubtitles".
	Name string `json:"name"`
	// InstanceID identifies the configured instance. Several instances of one
	// provider can exist, each with its own credentials and throttle state.
	InstanceID string `json:"instance_id,omitempty"`
	// Enabled reflects the instance's configuration.
	Enabled bool `json:"enabled"`
	// Available is true when the provider would be consulted by a search right
	// now: enabled, and not currently backed off.
	Available bool `json:"available"`
	// Throttled is true while the provider is in backoff after a failure.
	Throttled bool `json:"throttled"`
	// RetryAfter is when the backoff expires. Zero when not throttled.
	RetryAfter time.Time `json:"retry_after,omitempty"`
	// LastSuccess is when this instance last returned a subtitle. Zero means
	// it has never succeeded — which, for the many providers that are still
	// stubs, is the honest answer and needs no hardcoded list to produce.
	LastSuccess time.Time `json:"last_success,omitempty"`
	// LastFailure is when this instance last failed.
	LastFailure time.Time `json:"last_failure,omitempty"`
	// CheckedAt is when this snapshot was computed.
	CheckedAt time.Time `json:"checked_at"`
}

// outcome records what has actually happened to one provider instance.
type outcome struct {
	lastSuccess time.Time
	lastFailure time.Time
}

var (
	outcomeMu  sync.RWMutex
	outcomeMap = map[string]outcome{}
)

// recordSuccess notes that an instance returned a subtitle.
func recordSuccess(id string) {
	if id == "" {
		return
	}
	outcomeMu.Lock()
	defer outcomeMu.Unlock()
	o := outcomeMap[id]
	o.lastSuccess = time.Now()
	outcomeMap[id] = o
}

// recordFailure notes that an instance failed to return a subtitle.
func recordFailure(id string) {
	if id == "" {
		return
	}
	outcomeMu.Lock()
	defer outcomeMu.Unlock()
	o := outcomeMap[id]
	o.lastFailure = time.Now()
	outcomeMap[id] = o
}

// Reset clears the recorded provider history. Throttle state is left alone;
// ResetThrottling handles that.
func Reset() {
	outcomeMu.Lock()
	defer outcomeMu.Unlock()
	outcomeMap = map[string]outcome{}
}

// ResetThrottling clears every provider's backoff so throttled providers are
// retried immediately.
//
// This mirrors Bazarr's "reset provider throttling" action. It is what
// POST /api/providers/refresh now does: there is no availability to go and
// fetch — status is computed on read — but "stop waiting out the backoff and
// try again" is a real thing an operator wants after fixing credentials.
func ResetThrottling() {
	backoffMu.Lock()
	defer backoffMu.Unlock()
	backoffMap = map[string]time.Time{}
}

// List returns the status of every provider a search would currently consult,
// keyed by instance ID (or by provider name when no instances are configured).
//
// Disabled instances are included: "configured but switched off" is
// information, and omitting them would make a provider silently vanish from
// the list rather than show up as disabled.
//
// Note on what Available means: "would be consulted by a search right now" —
// enabled and not throttled. It deliberately does *not* mean "known to work".
// Whether a provider has ever actually produced a subtitle is carried by
// LastSuccess, which stays zero for the many providers that are still stubs.
// Conflating the two is what made the old implementation useless.
func List() map[string]Status {
	now := time.Now()

	insts := ListInstances() // enabled and disabled, priority order
	if len(insts) == 0 {
		// No configured instances. This is not an edge case: nothing in the
		// running server calls RegisterInstance, so on a real install the
		// instance registry is always empty and searches fall back to the
		// registered provider names. Reporting an empty map here would be
		// accurate about instances and useless about the system — the status
		// endpoint would answer "{}" on every deployment.
		for _, name := range All() {
			insts = append(insts, Instance{ID: name, Name: name, Enabled: true})
		}
	}

	backoffMu.RLock()
	deadlines := make(map[string]time.Time, len(insts))
	for _, inst := range insts {
		if until, ok := backoffMap[inst.ID]; ok {
			deadlines[inst.ID] = until
		}
	}
	backoffMu.RUnlock()

	outcomeMu.RLock()
	outcomes := make(map[string]outcome, len(insts))
	for _, inst := range insts {
		if o, ok := outcomeMap[inst.ID]; ok {
			outcomes[inst.ID] = o
		}
	}
	outcomeMu.RUnlock()

	out := make(map[string]Status, len(insts))
	for _, inst := range insts {
		until, hasBackoff := deadlines[inst.ID]
		throttled := hasBackoff && now.Before(until)
		o := outcomes[inst.ID]

		s := Status{
			Name:        inst.Name,
			InstanceID:  inst.ID,
			Enabled:     inst.Enabled,
			Throttled:   throttled,
			Available:   inst.Enabled && !throttled,
			LastSuccess: o.lastSuccess,
			LastFailure: o.lastFailure,
			CheckedAt:   now,
		}
		if throttled {
			s.RetryAfter = until
		}
		out[inst.ID] = s
	}
	return out
}
