// file: pkg/profiles/defaultlang_test.go
// version: 1.0.0
// guid: a81b0a4c-2e73-42e5-8e67-ed6180c2c91f
// last-edited: 2026-07-30

package profiles

import (
	"testing"

	"github.com/spf13/viper"
)

// useLangConfig sets language keys for one test and restores them after.
func useLangConfig(t *testing.T, kv map[string]any) {
	t.Helper()
	keys := []string{"languages.default", "languages.preferred"}
	prev := map[string]any{}
	for _, k := range keys {
		prev[k] = viper.Get(k)
	}
	t.Cleanup(func() {
		for k, v := range prev {
			viper.Set(k, v)
		}
	})
	for _, k := range keys {
		viper.Set(k, nil)
	}
	for k, v := range kv {
		viper.Set(k, v)
	}
}

// TestDefaultLanguage covers the resolution order.
//
// The behaviour that matters most is the last case: with nothing configured
// the answer must still be "en". Every call site hardcoded English before this
// was configurable, so any other default would silently change what an existing
// install downloads.
func TestDefaultLanguage(t *testing.T) {
	for name, tc := range map[string]struct {
		cfg  map[string]any
		want string
	}{
		"explicit default wins": {
			map[string]any{
				"languages.default":   "de",
				"languages.preferred": []string{"fr", "es"},
			}, "de"},
		"preferred list is used when no explicit default": {
			map[string]any{"languages.preferred": []string{"fr", "es"}}, "fr"},
		"unconfigured preserves the previous hardcoded behaviour": {
			map[string]any{}, "en"},
		"blank explicit default falls through": {
			map[string]any{
				"languages.default":   "   ",
				"languages.preferred": []string{"pt-BR"},
			}, "pt-BR"},
		"blank entries in the preferred list are skipped": {
			map[string]any{"languages.preferred": []string{"", " ", "nl"}}, "nl"},
		"empty preferred list falls back": {
			map[string]any{"languages.preferred": []string{}}, "en"},
	} {
		t.Run(name, func(t *testing.T) {
			useLangConfig(t, tc.cfg)
			if got := DefaultLanguage(); got != tc.want {
				t.Errorf("DefaultLanguage() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestDefaultLanguageTrimsWhitespace guards against a config value with stray
// spaces becoming a language code no provider recognises.
func TestDefaultLanguageTrimsWhitespace(t *testing.T) {
	useLangConfig(t, map[string]any{"languages.default": "  es  "})
	if got := DefaultLanguage(); got != "es" {
		t.Errorf("DefaultLanguage() = %q, want %q", got, "es")
	}
}
