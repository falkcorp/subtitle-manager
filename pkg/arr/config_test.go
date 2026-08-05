// file: pkg/arr/config_test.go
// version: 1.0.0
// guid: 1f8d5b03-42a7-4e69-b0c8-7e94a1d6350f
// last-edited: 2026-08-04

package arr

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func reset(t *testing.T) {
	t.Helper()
	viper.Reset()
	warnOnce.Range(func(k, _ any) bool { warnOnce.Delete(k); return true })
	t.Cleanup(viper.Reset)
}

// TestResolvePrefersIntegrations pins that the canonical scheme wins, including
// the parts the flat keys cannot express — scheme and base path.
func TestResolvePrefersIntegrations(t *testing.T) {
	reset(t)
	viper.Set("integrations.sonarr.enabled", true)
	viper.Set("integrations.sonarr.host", "tv.example")
	viper.Set("integrations.sonarr.port", "8989")
	viper.Set("integrations.sonarr.api_key", "canonical")
	viper.Set("integrations.sonarr.ssl", true)
	viper.Set("integrations.sonarr.base_url", "/sonarr")
	// Present but shadowed.
	viper.Set("sonarr_url", "http://old:8989")
	viper.Set("sonarr_api_key", "legacy")

	conn, ok := Resolve(Sonarr)
	if !ok {
		t.Fatal("not resolved")
	}
	if conn.APIKey != "canonical" {
		t.Errorf("api key = %q, want the integrations one", conn.APIKey)
	}
	if conn.URL != "https://tv.example:8989/sonarr" {
		t.Errorf("url = %q; ssl and base_url must be honoured", conn.URL)
	}
	if conn.Legacy {
		t.Error("reported Legacy for a canonical config")
	}
}

// TestResolveFallsBackToFlatKeys is the half that was broken: the commands read
// only these, the server read only integrations.*, so one scheme always left
// part of the app blind. Every caller resolves through here now, so a flat-key
// config has to keep working.
func TestResolveFallsBackToFlatKeys(t *testing.T) {
	reset(t)
	viper.Set("radarr_url", "http://movies:7878")
	viper.Set("radarr_api_key", "legacy")

	conn, ok := Resolve(Radarr)
	if !ok {
		t.Fatal("flat keys did not resolve; existing configs would break")
	}
	if conn.URL != "http://movies:7878" || conn.APIKey != "legacy" {
		t.Errorf("got %+v", conn)
	}
	if !conn.Legacy {
		t.Error("did not report Legacy for a flat-key config")
	}
}

// TestDisabledIntegrationIsNotResurrectedByFlatKeys guards the one way this
// unification could do harm. Because every caller now falls back to the flat
// keys, a stale sonarr_url alongside an explicitly disabled integration would
// otherwise start syncing against the user's stated wish.
func TestDisabledIntegrationIsNotResurrectedByFlatKeys(t *testing.T) {
	reset(t)
	viper.Set("integrations.sonarr.enabled", false)
	viper.Set("integrations.sonarr.host", "tv.example")
	viper.Set("integrations.sonarr.port", "8989")
	viper.Set("sonarr_url", "http://old:8989")
	viper.Set("sonarr_api_key", "legacy")

	if conn, ok := Resolve(Sonarr); ok {
		t.Errorf("resolved %+v; enabled=false with a host set is an explicit opt-out", conn)
	}
}

func TestResolveUnconfigured(t *testing.T) {
	reset(t)
	if _, ok := Resolve(Radarr); ok {
		t.Error("resolved with nothing configured")
	}
}

// TestSyncIntervalAcceptsBothSpellings is the other half of the split: Radarr
// read sync_interval and Sonarr read episode_sync_interval, so setting
// integrations.sonarr.sync_interval was silently ignored and fell back to 60
// minutes with nothing reported.
func TestSyncIntervalAcceptsBothSpellings(t *testing.T) {
	for _, tc := range []struct {
		name    string
		service string
		key     string
		want    time.Duration
	}{
		{"sonarr episode_sync_interval", Sonarr, "integrations.sonarr.episode_sync_interval", 15 * time.Minute},
		{"sonarr sync_interval", Sonarr, "integrations.sonarr.sync_interval", 15 * time.Minute},
		{"radarr sync_interval", Radarr, "integrations.radarr.sync_interval", 15 * time.Minute},
		{"radarr episode_sync_interval", Radarr, "integrations.radarr.episode_sync_interval", DefaultSyncInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reset(t)
			viper.Set(tc.key, 15)
			if got := SyncInterval(tc.service); got != tc.want {
				t.Errorf("SyncInterval(%s) with %s = %s, want %s", tc.service, tc.key, got, tc.want)
			}
		})
	}
}

// TestSyncIntervalPrefersServiceSpecific pins the tie-break, since the Bazarr
// importer writes episode_sync_interval for Sonarr.
func TestSyncIntervalPrefersServiceSpecific(t *testing.T) {
	reset(t)
	viper.Set("integrations.sonarr.episode_sync_interval", 5)
	viper.Set("integrations.sonarr.sync_interval", 45)
	if got := SyncInterval(Sonarr); got != 5*time.Minute {
		t.Errorf("got %s, want 5m from episode_sync_interval", got)
	}
}

func TestSyncIntervalDefault(t *testing.T) {
	reset(t)
	if got := SyncInterval(Radarr); got != DefaultSyncInterval {
		t.Errorf("got %s, want %s", got, DefaultSyncInterval)
	}
}
