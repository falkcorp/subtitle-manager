// file: pkg/transcriber/asr_timeout_test.go
// version: 1.0.0
// guid: 21bc33a0-3c6d-49c4-a350-b1792ad6986a
// last-edited: 2026-07-30

package transcriber

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// useWhisperConfig sets whisper timeout keys for one test.
func useWhisperConfig(t *testing.T, kv map[string]any) {
	t.Helper()
	keys := []string{"whisper.connect_timeout", "whisper.transcribe_timeout"}
	prev := map[string]any{}
	for _, k := range keys {
		prev[k] = viper.Get(k)
		viper.Set(k, nil)
	}
	t.Cleanup(func() {
		for k, v := range prev {
			viper.Set(k, v)
		}
	})
	for k, v := range kv {
		viper.Set(k, v)
	}
}

// TestASRTimeoutsAreSeparate verifies the two budgets are read independently
// and have sane defaults.
func TestASRTimeoutsAreSeparate(t *testing.T) {
	useWhisperConfig(t, nil)
	if got := asrConnectTimeout(); got != defaultASRConnectTimeout {
		t.Errorf("default connect timeout = %v, want %v", got, defaultASRConnectTimeout)
	}
	if got := asrTotalTimeout(); got != defaultASRTotalTimeout {
		t.Errorf("default total timeout = %v, want %v", got, defaultASRTotalTimeout)
	}

	// The transcription budget must not become the connect budget: that
	// coupling is the bug — a 30 minute transcription allowance meant a dead
	// server also took 30 minutes to report itself dead.
	useWhisperConfig(t, map[string]any{"whisper.transcribe_timeout": 1800})
	if got := asrConnectTimeout(); got != defaultASRConnectTimeout {
		t.Errorf("connect timeout = %v after setting only transcribe_timeout; "+
			"want it to stay %v", got, defaultASRConnectTimeout)
	}
	if got := asrTotalTimeout(); got != 30*time.Minute {
		t.Errorf("total timeout = %v, want 30m", got)
	}

	useWhisperConfig(t, map[string]any{"whisper.connect_timeout": 3})
	if got := asrConnectTimeout(); got != 3*time.Second {
		t.Errorf("connect timeout = %v, want 3s", got)
	}
}

// TestASRClientAppliesConnectTimeout verifies the configured connect timeout
// actually reaches the transport.
//
// This is asserted structurally rather than by timing a hanging connect. A
// behavioural version was written first and thrown away: it filled a
// listener's accept backlog hoping connections would hang, but the OS accepted
// them anyway, so it passed in 0.02s — and kept passing with the fix reverted.
// A test that cannot fail is worse than no test. Reaching a genuinely
// black-holing address means depending on the network from CI, which trades
// one kind of unreliability for another.
//
// TLSHandshakeTimeout is the observable proxy for the dial deadlines: both are
// set from the same connect budget, and a clone of http.DefaultTransport
// carries 10s, so configuring anything else makes an unapplied setting visible.
func TestASRClientAppliesConnectTimeout(t *testing.T) {
	useWhisperConfig(t, map[string]any{
		"whisper.connect_timeout":    3,
		"whisper.transcribe_timeout": 1800,
	})

	c := newASRClient()
	if c.Timeout != 30*time.Minute {
		t.Errorf("client timeout = %v, want the 30m transcription budget", c.Timeout)
	}

	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatal("ASR client transport is not *http.Transport")
	}
	if tr.TLSHandshakeTimeout != 3*time.Second {
		t.Errorf("TLSHandshakeTimeout = %v, want 3s: the connect budget is not "+
			"reaching the transport, so an unreachable Whisper server would be "+
			"bounded only by the 30 minute transcription budget",
			tr.TLSHandshakeTimeout)
	}
	if tr.DialContext == nil {
		t.Error("DialContext is nil; the dial deadline is not being applied")
	}
}

// TestASRClientInheritsProxy verifies the ASR client does not opt out of the
// configured outbound proxy.
//
// pkg/proxy implements proxy_url by setting Proxy on http.DefaultTransport.
// Building a fresh transport here instead of cloning would silently exclude
// Whisper traffic from it.
func TestASRClientInheritsProxy(t *testing.T) {
	tr, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Skip("default transport is not *http.Transport")
	}
	prev := tr.Proxy
	t.Cleanup(func() { tr.Proxy = prev })

	called := false
	tr.Proxy = func(*http.Request) (*url.URL, error) {
		called = true
		return nil, nil
	}

	c := newASRClient()
	ctr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatal("ASR client transport is not *http.Transport")
	}
	if ctr.Proxy == nil {
		t.Fatal("ASR client dropped the proxy configuration; whisper traffic " +
			"would bypass proxy_url")
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.invalid/", nil)
	if _, err := ctr.Proxy(req); err != nil {
		t.Fatalf("proxy func: %v", err)
	}
	if !called {
		t.Error("ASR client is not using the configured proxy function")
	}
}
