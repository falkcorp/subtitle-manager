// file: pkg/providers/config_test.go
// version: 1.0.0
// guid: 1739868a-7ed1-45e2-897c-ecc92a00e135
// last-edited: 2026-07-30

package providers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// useConfigFile points viper at a config file written from body, applying the
// same defaults cmd/root.go sets, and restores global state afterwards.
//
// The defaults matter: they are what makes the "no configuration" case
// dangerous, so a test that omits them is testing a situation that never
// occurs in production.
func useConfigFile(t *testing.T, body string) {
	t.Helper()

	instancesMu.Lock()
	prevInsts := instances
	instancesMu.Unlock()
	t.Cleanup(func() {
		instancesMu.Lock()
		instances = prevInsts
		instancesMu.Unlock()
	})

	viper.Reset()
	t.Cleanup(viper.Reset)

	// Mirrors cmd/root.go.
	viper.SetDefault("providers.embedded.enabled", true)
	viper.SetDefault("providers.generic.api_url", "")
	viper.SetDefault("providers.whisper.api_url", "http://localhost:9000")

	if body != "" {
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}
		viper.SetConfigFile(path)
		if err := viper.ReadInConfig(); err != nil {
			t.Fatalf("read config: %v", err)
		}
	}
}

// TestLoadFromConfigRegistersNothingByDefault is the most important test in
// this package.
//
// If loading produces even one instance on an unconfigured install,
// FetchFromAll stops using its name-based fallback over every registered
// provider and consults only that instance — search silently collapses from
// ~55 providers to one, and nothing reports an error. Subtitles just stop
// being found.
//
// The trap is concrete: cmd/root.go sets a default of
// providers.embedded.enabled = true, so on a bare install both
// viper.GetBool and viper.IsSet report true for embedded while the operator
// has configured nothing at all. A loader keyed on either would register
// embedded and collapse search to embedded-only. Enablement is therefore read
// with viper.InConfig, which asks whether the key came from the config file.
func TestLoadFromConfigRegistersNothingByDefault(t *testing.T) {
	useConfigFile(t, "") // no config file at all: defaults only

	// Precondition: the trap is real, not hypothetical.
	if !viper.GetBool("providers.embedded.enabled") {
		t.Fatal("precondition: the embedded default should read as true")
	}
	if !viper.IsSet("providers.embedded.enabled") {
		t.Fatal("precondition: IsSet should also be fooled by the default")
	}

	LoadFromConfig()

	if got := Instances(); len(got) != 0 {
		t.Fatalf("loaded %d instances from an unconfigured install: %v\n"+
			"search would be restricted to these instead of using every "+
			"registered provider", len(got), got)
	}
}

// TestLoadFromConfigHonoursExplicitConfig verifies an operator's explicit
// choices are applied: only enabled providers become instances.
func TestLoadFromConfigHonoursExplicitConfig(t *testing.T) {
	useConfigFile(t, `
providers:
  opensubtitles:
    enabled: true
  podnapisi:
    enabled: true
  subscene:
    enabled: false
`)
	LoadFromConfig()

	got := Instances()
	if len(got) != 2 {
		t.Fatalf("got %d instances, want 2: %v", len(got), got)
	}
	names := map[string]bool{}
	for _, i := range got {
		names[i.Name] = true
	}
	if !names["opensubtitles"] || !names["podnapisi"] {
		t.Errorf("enabled providers missing: %v", got)
	}
	if names["subscene"] {
		t.Error("an explicitly disabled provider was registered")
	}
	// The default-only provider must not sneak in alongside real config.
	if names["embedded"] {
		t.Error("embedded registered from its default while it was never configured")
	}
}

// TestLoadFromConfigOrderIsDeterministic guards against search order changing
// between restarts.
//
// Configuration does not require a priority, so equal priorities are the
// common case. Instances are held in a map, whose iteration order Go
// randomises per process, and sort.Slice is not stable — without a tiebreak
// the provider consulted first would differ on every restart, which is both
// unreproducible for the operator and untestable.
func TestLoadFromConfigOrderIsDeterministic(t *testing.T) {
	cfg := `
providers:
  opensubtitles:
    enabled: true
  podnapisi:
    enabled: true
  napiprojekt:
    enabled: true
  gestdown:
    enabled: true
`
	var first []string
	for run := 0; run < 8; run++ {
		useConfigFile(t, cfg)
		LoadFromConfig()

		var order []string
		for _, i := range Instances() {
			order = append(order, i.Name)
		}
		if run == 0 {
			first = order
			continue
		}
		for k := range order {
			if order[k] != first[k] {
				t.Fatalf("run %d ordered providers %v, first run had %v: "+
					"search order is not reproducible across restarts", run, order, first)
			}
		}
	}
	// With no priorities set, the tiebreak is by ID, i.e. alphabetical.
	want := []string{"gestdown", "napiprojekt", "opensubtitles", "podnapisi"}
	for k := range want {
		if first[k] != want[k] {
			t.Fatalf("order = %v, want %v (ties broken by name)", first, want)
		}
	}
}

// TestLoadFromConfigAppliesPriority verifies configured priority wins over the
// name tiebreak.
func TestLoadFromConfigAppliesPriority(t *testing.T) {
	useConfigFile(t, `
providers:
  zimuku:
    enabled: true
    priority: 100
  gestdown:
    enabled: true
    priority: 1
`)
	LoadFromConfig()

	got := Instances()
	if len(got) != 2 {
		t.Fatalf("got %d instances, want 2", len(got))
	}
	// Alphabetically gestdown sorts first; priority must override that.
	if got[0].Name != "zimuku" {
		t.Errorf("first provider = %q, want zimuku: configured priority is not applied", got[0].Name)
	}
}

// TestLoadFromConfigReplacesRatherThanMerges verifies a provider removed from
// configuration stops being consulted.
//
// RegisterInstance only adds or updates, so a loader built on it would leave a
// disabled provider in the registry forever — it would keep being searched
// after the operator turned it off, with the settings page showing it as
// disabled.
func TestLoadFromConfigReplacesRatherThanMerges(t *testing.T) {
	useConfigFile(t, `
providers:
  opensubtitles:
    enabled: true
  podnapisi:
    enabled: true
`)
	LoadFromConfig()
	if len(Instances()) != 2 {
		t.Fatalf("precondition: want 2 instances, got %d", len(Instances()))
	}

	// The operator turns podnapisi off and reloads.
	useConfigFile(t, `
providers:
  opensubtitles:
    enabled: true
  podnapisi:
    enabled: false
`)
	LoadFromConfig()

	got := Instances()
	if len(got) != 1 || got[0].Name != "opensubtitles" {
		t.Fatalf("after disabling podnapisi, instances = %v; a removed provider "+
			"is still registered and would still be searched", got)
	}
}

// TestLoadFromConfigEmptiesRegistryWhenConfigCleared verifies going from
// "some providers configured" back to none restores the fallback rather than
// leaving a stale registry that restricts search.
func TestLoadFromConfigEmptiesRegistryWhenConfigCleared(t *testing.T) {
	useConfigFile(t, "providers:\n  opensubtitles:\n    enabled: true\n")
	LoadFromConfig()
	if len(Instances()) != 1 {
		t.Fatalf("precondition: want 1 instance, got %d", len(Instances()))
	}

	useConfigFile(t, "")
	LoadFromConfig()
	if got := Instances(); len(got) != 0 {
		t.Fatalf("clearing provider config left %v registered; search would stay "+
			"restricted instead of returning to every provider", got)
	}
}
