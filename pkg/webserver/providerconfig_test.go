// file: pkg/webserver/providerconfig_test.go
// version: 1.1.0
// guid: 4e2a7c81-93f5-4d06-b7ac-58e1230fd946
// last-edited: 2026-08-12

package webserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/providers"
)

// TestEnablingAProviderTakesEffect is the whole point of the Settings →
// Providers page: after you switch a provider on, searches must actually use
// it.
//
// The page is the only route to a working provider — only `embedded` ships
// enabled, and that needs ffmpeg. So if this does not work, a fresh install has
// no way to fetch a subtitle at all.
//
// This asserts the observable effect (providers.Instances) rather than the
// response status. POST /api/config returns 204 regardless: it writes the key
// verbatim into viper, so a key nothing reads still looks like a success.
func TestEnablingAProviderTakesEffect(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("db_backend: pebble\n"), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	viper.Reset()
	t.Cleanup(func() {
		viper.Reset()
		providers.SetInstances(nil)
	})
	viper.SetConfigFile(cfgPath)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("reading config: %v", err)
	}
	providers.SetInstances(nil)
	providers.LoadFromConfig()

	if got := len(providers.Instances()); got != 0 {
		t.Fatalf("precondition: expected no configured providers, got %d", got)
	}

	body := []byte(`{"providers.gestdown.enabled": true, "providers.gestdown.priority": 5}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	configHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST /api/config = %d, want %d", rec.Code, http.StatusNoContent)
	}

	insts := providers.Instances()
	if len(insts) != 1 {
		t.Fatalf("after enabling gestdown, Instances() has %d entries, want 1; "+
			"the setting was saved but nothing acted on it", len(insts))
	}
	if insts[0].Name != "gestdown" {
		t.Errorf("enabled provider = %q, want %q", insts[0].Name, "gestdown")
	}
	if insts[0].Priority != 5 {
		t.Errorf("priority = %d, want 5", insts[0].Priority)
	}
}

// TestConfigHierarchyIsPreserved pins down that nested option trees merge
// rather than overwrite each other. Settings are keyed by dotted path
// (providers.<name>.<field>, integrations.<name>.<field>), so a save from one
// page must never disturb a sibling subtree owned by another page.
//
// Verified behaviour, not assumption: viper deep-merges a dotted Set into the
// existing tree, and normalises key case, so none of these collide.
func TestConfigHierarchyIsPreserved(t *testing.T) {
	existing := `db_backend: pebble
providers:
    gestdown:
        api_key: SECRET
        priority: 7
    opensubtitles:
        api_key: OTHER
integrations:
    sonarr:
        api_key: SONARRKEY
        url: http://sonarr:8989
`
	for _, tc := range []struct {
		name string
		body string
	}{
		{"dotted leaf", `{"providers.gestdown.enabled": true}`},
		{"nested object at the parent key", `{"providers":{"gestdown":{"enabled":true}}}`},
		{"mixed-case key", `{"providers.GestDown.enabled": true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(cfgPath, []byte(existing), 0o600); err != nil {
				t.Fatalf("writing config: %v", err)
			}
			viper.Reset()
			t.Cleanup(func() {
				viper.Reset()
				providers.SetInstances(nil)
			})
			viper.SetConfigFile(cfgPath)
			if err := viper.ReadInConfig(); err != nil {
				t.Fatalf("reading config: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader([]byte(tc.body)))
			rec := httptest.NewRecorder()
			configHandler().ServeHTTP(rec, req)
			if rec.Code != http.StatusNoContent {
				t.Fatalf("POST /api/config = %d, want %d", rec.Code, http.StatusNoContent)
			}

			raw, err := os.ReadFile(cfgPath)
			if err != nil {
				t.Fatalf("reading back: %v", err)
			}
			got := string(raw)
			// Everything the request did not mention must survive verbatim.
			for _, want := range []string{"SECRET", "priority: 7", "OTHER", "SONARRKEY", "sonarr:8989", "db_backend"} {
				if !strings.Contains(got, want) {
					t.Errorf("saving one option destroyed %q from another part of the tree.\nfile:\n%s", want, got)
				}
			}
			if !strings.Contains(got, "enabled: true") {
				t.Errorf("the submitted option was not persisted.\nfile:\n%s", got)
			}
		})
	}
}

// TestConfigRejectsOverwritingASubtreeWithAScalar guards the one shape that
// genuinely clobbers. Setting a key that is a *prefix* of an existing subtree
// to a non-map value replaces the whole subtree: posting
// {"providers.gestdown": "x"} silently deletes gestdown's api_key and
// priority.
//
// Nothing in the UI should send that, which is exactly why it must fail loudly
// rather than quietly destroy credentials — a settings page with a
// wrong-shaped key would otherwise wipe an operator's provider config and
// report success.
func TestConfigRejectsOverwritingASubtreeWithAScalar(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	existing := "providers:\n    gestdown:\n        api_key: SECRET\n        priority: 7\n"
	if err := os.WriteFile(cfgPath, []byte(existing), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	viper.Reset()
	t.Cleanup(func() {
		viper.Reset()
		providers.SetInstances(nil)
	})
	viper.SetConfigFile(cfgPath)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("reading config: %v", err)
	}

	body := []byte(`{"providers.gestdown": "scalar"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	configHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /api/config = %d, want %d for a scalar over a subtree", rec.Code, http.StatusBadRequest)
	}

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if !strings.Contains(string(raw), "SECRET") {
		t.Errorf("the rejected write still destroyed the subtree.\nfile:\n%s", raw)
	}
}

// TestSavingAnUnrelatedSettingDoesNotCollapseSearch is the serious one.
//
// viper.WriteConfig serialises viper.AllSettings(), which merges in defaults —
// and cmd/root.go sets providers.embedded.enabled to true. So saving any single
// setting from any settings page used to write
//
//	providers:
//	    embedded:
//	        enabled: true
//
// into the operator's config file. On the next start InConfig reports true for
// it, LoadFromConfig registers exactly one instance, and per the safety
// property documented at pkg/providers/config.go:25-40 that flips FetchFromAll
// out of its name-based fallback: subtitle search silently drops from every
// registered provider to embedded alone, which additionally needs ffmpeg.
//
// The comment there chose InConfig over IsSet precisely to stop this. The
// protection was defeated from the other end — by the config *writer* putting
// the default into the file, after which InConfig legitimately reports true.
//
// Saving a setting must never enable a provider the operator did not choose.
func TestSavingAnUnrelatedSettingDoesNotCollapseSearch(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("db_backend: pebble\n"), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	viper.Reset()
	t.Cleanup(func() {
		viper.Reset()
		providers.SetInstances(nil)
	})
	viper.SetConfigFile(cfgPath)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("reading config: %v", err)
	}
	// Reproduce the defaults cmd/root.go installs.
	viper.SetDefault("providers.embedded.enabled", true)
	viper.SetDefault("whisper.model", "base")

	body := []byte(`{"whisper_api_url":"http://example:9000"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	configHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST /api/config = %d, want %d", rec.Code, http.StatusNoContent)
	}

	// The operator's file must contain what the operator set, and nothing else.
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("reading config back: %v", err)
	}
	if strings.Contains(string(raw), "embedded") {
		t.Errorf("saving an unrelated setting wrote a provider default into the config file; "+
			"search will collapse to embedded-only on restart.\nfile:\n%s", raw)
	}
	if !strings.Contains(string(raw), "example:9000") {
		t.Errorf("the setting that was actually posted is missing.\nfile:\n%s", raw)
	}
	if !strings.Contains(string(raw), "db_backend") {
		t.Errorf("a pre-existing setting was dropped.\nfile:\n%s", raw)
	}

	// And nothing may have been registered as a result.
	if insts := providers.Instances(); len(insts) != 0 {
		t.Errorf("saving an unrelated setting registered %d provider instance(s): %+v", len(insts), insts)
	}
}

// TestDisablingAProviderTakesEffect covers the other direction. Turning a
// provider off has to remove it from the search set, or a provider that is
// rate-limiting or returning garbage cannot be switched out from the UI.
func TestDisablingAProviderTakesEffect(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	cfg := "db_backend: pebble\nproviders:\n  gestdown:\n    enabled: true\n"
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	viper.Reset()
	t.Cleanup(func() {
		viper.Reset()
		providers.SetInstances(nil)
	})
	viper.SetConfigFile(cfgPath)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("reading config: %v", err)
	}
	providers.LoadFromConfig()

	if got := len(providers.Instances()); got != 1 {
		t.Fatalf("precondition: expected gestdown enabled, got %d instances", got)
	}

	body := []byte(`{"providers.gestdown.enabled": false}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	configHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST /api/config = %d, want %d", rec.Code, http.StatusNoContent)
	}

	if insts := providers.Instances(); len(insts) != 0 {
		t.Errorf("after disabling gestdown, Instances() still has %d entries", len(insts))
	}
}
