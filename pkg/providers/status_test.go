// file: pkg/providers/status_test.go
// version: 1.1.0
// guid: cb3753c6-7b0d-45cd-9137-9528ad6da276
// last-edited: 2026-07-30

package providers

import (
	"encoding/json"
	"testing"
	"time"
)

// useStatusState isolates the instance, backoff and outcome registries for one
// test. All three are package-level globals, so a test that merely adds to them
// leaks into every test that runs afterwards.
func useStatusState(t *testing.T, insts ...Instance) {
	t.Helper()

	instancesMu.Lock()
	prevInsts := instances
	instances = map[string]Instance{}
	for _, i := range insts {
		instances[i.ID] = i
	}
	instancesMu.Unlock()

	backoffMu.Lock()
	prevBackoff := backoffMap
	backoffMap = map[string]time.Time{}
	backoffMu.Unlock()

	outcomeMu.Lock()
	prevOutcome := outcomeMap
	outcomeMap = map[string]outcome{}
	outcomeMu.Unlock()

	t.Cleanup(func() {
		instancesMu.Lock()
		instances = prevInsts
		instancesMu.Unlock()
		backoffMu.Lock()
		backoffMap = prevBackoff
		backoffMu.Unlock()
		outcomeMu.Lock()
		outcomeMap = prevOutcome
		outcomeMu.Unlock()
	})
}

// TestListReportsConfiguredInstances covers the core of the rewrite: status is
// derived from the instance registry, so every configured provider appears and
// no unconfigured one does.
//
// The old implementation reported whatever the last Refresh call had been
// handed, which in practice was a hardcoded pair of names — so real providers
// never appeared at all and one non-functional stub was listed as healthy.
func TestListReportsConfiguredInstances(t *testing.T) {
	useStatusState(t,
		Instance{ID: "opensubtitles-1", Name: "opensubtitles", Enabled: true, Priority: 10},
		Instance{ID: "podnapisi-1", Name: "podnapisi", Enabled: true, Priority: 5},
		Instance{ID: "gestdown-1", Name: "gestdown", Enabled: false, Priority: 1},
	)

	got := List()
	if len(got) != 3 {
		t.Fatalf("got %d entries, want 3: %v", len(got), got)
	}

	if _, ok := got["subscene-1"]; ok {
		t.Error("an unconfigured provider appeared in the status list")
	}
	if s := got["podnapisi-1"]; !s.Available || s.Name != "podnapisi" {
		t.Errorf("podnapisi = %+v, want an available entry", s)
	}

	// A disabled instance is listed, but not as available. Omitting it would
	// make the provider silently vanish rather than show as switched off.
	d := got["gestdown-1"]
	if d.Enabled {
		t.Error("disabled instance reported as enabled")
	}
	if d.Available {
		t.Error("disabled instance reported as available")
	}
}

// TestListNeverInventsAvailability is the regression guard for the specific
// defect this change fixes.
//
// providers.Refresh marked every name it was passed Available: true without
// performing any check. A provider that has never succeeded must not be
// reported as healthy just because it is configured — Available means "would
// be consulted right now", and LastSuccess carries whether it has ever worked.
func TestListNeverInventsAvailability(t *testing.T) {
	useStatusState(t, Instance{ID: "stub-1", Name: "stub", Enabled: true})

	s := List()["stub-1"]
	if s.LastSuccess != nil {
		t.Error("a provider that never ran reports a last success")
	}
	if s.LastFailure != nil {
		t.Error("a provider that never ran reports a last failure")
	}
}

// TestStatusJSONOmitsUnsetTimes asserts on the serialised form, not the Go
// struct.
//
// This is the bug the Go-level assertions could not see. The time fields were
// plain time.Time with `omitempty`, but encoding/json does not apply omitempty
// to structs — a zero time.Time serialises as "0001-01-01T00:00:00Z" rather
// than being omitted. Every one of the 52 providers therefore shipped a
// last_success, so the field meant to separate a working provider from a stub
// was present on all of them and distinguished nothing. IsZero() in Go looked
// fine throughout; only querying a running server showed it.
func TestStatusJSONOmitsUnsetTimes(t *testing.T) {
	useStatusState(t,
		Instance{ID: "never-1", Name: "never", Enabled: true},
		Instance{ID: "worked-1", Name: "worked", Enabled: true},
	)
	SetBackoff("worked-1", 0) // a success

	raw, err := json.Marshal(List())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, present := got["never-1"]["last_success"]; present {
		t.Errorf("a provider that never succeeded still carries last_success in "+
			"JSON (%s); clients cannot tell it from one that works", raw)
	}
	if _, present := got["never-1"]["retry_after"]; present {
		t.Error("an unthrottled provider carries retry_after in JSON")
	}
	if _, present := got["worked-1"]["last_success"]; !present {
		t.Error("a provider that succeeded is missing last_success in JSON")
	}
}

// TestStatusTracksThrottling verifies backoff is reflected as throttled and
// unavailable, and that an expired backoff is not.
func TestStatusTracksThrottling(t *testing.T) {
	useStatusState(t,
		Instance{ID: "thr-1", Name: "thr", Enabled: true},
		Instance{ID: "exp-1", Name: "exp", Enabled: true},
	)

	SetBackoff("thr-1", time.Minute)
	// Already elapsed: recorded, but no longer in force.
	backoffMu.Lock()
	backoffMap["exp-1"] = time.Now().Add(-time.Minute)
	backoffMu.Unlock()

	got := List()

	thr := got["thr-1"]
	if !thr.Throttled {
		t.Error("a backed-off provider is not reported as throttled")
	}
	if thr.Available {
		t.Error("a throttled provider is reported as available; a search would skip it")
	}
	if thr.RetryAfter == nil {
		t.Error("a throttled provider reports no retry time")
	}

	exp := got["exp-1"]
	if exp.Throttled || !exp.Available {
		t.Errorf("expired backoff still counted: %+v", exp)
	}
	if exp.RetryAfter != nil {
		t.Error("an expired backoff still reports a retry time")
	}
}

// TestSetBackoffRecordsOutcomes verifies the seam that gives status real
// history.
//
// SetBackoff is the single point both fetch loops already call: zero after a
// provider returned a subtitle, non-zero after one failed. Recording there
// means status reflects what actually happened without the fetch path having
// to know status exists.
func TestSetBackoffRecordsOutcomes(t *testing.T) {
	useStatusState(t,
		Instance{ID: "ok-1", Name: "ok", Enabled: true},
		Instance{ID: "bad-1", Name: "bad", Enabled: true},
	)

	SetBackoff("bad-1", time.Minute) // a failure
	SetBackoff("ok-1", 0)            // a success

	got := List()
	if got["ok-1"].LastSuccess == nil {
		t.Error("a successful fetch recorded no success")
	}
	if got["ok-1"].LastFailure != nil {
		t.Error("a successful fetch recorded a failure")
	}
	if got["bad-1"].LastFailure == nil {
		t.Error("a failed fetch recorded no failure")
	}
	if got["bad-1"].LastSuccess != nil {
		t.Error("a failed fetch recorded a success")
	}
}

// TestResetThrottlingClearsBackoffOnly verifies the two reset endpoints do
// different things: POST /api/providers/refresh clears throttles so providers
// are retried now, and POST /api/providers/reset clears recorded history.
// Conflating them would mean an operator cannot retry a throttled provider
// without also erasing the record of why it was throttled.
func TestResetThrottlingClearsBackoffOnly(t *testing.T) {
	useStatusState(t, Instance{ID: "r-1", Name: "r", Enabled: true})

	SetBackoff("r-1", time.Minute)
	if !List()["r-1"].Throttled {
		t.Fatal("precondition: provider should be throttled")
	}

	ResetThrottling()
	after := List()["r-1"]
	if after.Throttled {
		t.Error("ResetThrottling left the provider throttled")
	}
	if after.LastFailure == nil {
		t.Error("ResetThrottling erased the failure history; that is Reset's job")
	}

	Reset()
	if List()["r-1"].LastFailure != nil {
		t.Error("Reset did not clear the recorded history")
	}
}
