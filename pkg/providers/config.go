// file: pkg/providers/config.go
// version: 1.0.0
// guid: 1c179991-4713-46c7-8b56-052eea65030f
// last-edited: 2026-07-30

package providers

import (
	"sort"

	"github.com/spf13/viper"
)

// LoadFromConfig populates the provider instance registry from configuration.
//
// # Why this exists
//
// The instance layer — per-provider enablement, priority, tags and throttle
// state — had no production caller at all: nothing outside tests ever called
// RegisterInstance, so Instances() was always empty on a real install. Every
// search therefore fell through to the flat list of registered provider names,
// and the provider settings the UI writes (POST /api/providers persists
// providers.<name>.enabled into the config file) were read by nothing.
//
// # The safety property that governs the whole design
//
// On an install with no provider configuration, this MUST register zero
// instances. Registering even one flips FetchFromAll out of its name-based
// fallback and into "only the configured instances", silently cutting search
// from every registered provider down to that one. There is a test asserting
// exactly this, because the failure is invisible until subtitles stop being
// found.
//
// That is why enablement is read with viper.InConfig rather than IsSet or
// GetBool. cmd/root.go sets viper.SetDefault("providers.embedded.enabled",
// true), so on a bare install GetBool and IsSet both report true for embedded
// while nothing is configured — a loader keyed on those would register exactly
// one instance and collapse search to embedded-only. InConfig reports whether
// the key came from the config file itself, which is the question actually
// being asked: "did the operator choose this?"
//
// # What it deliberately does not do
//
// Configuration is keyed by provider name (providers.<name>), so it can express
// exactly one instance per provider. Several instances of the same provider
// with distinct credentials — the thing the instance layer was built for —
// needs a config schema that does not exist yet (a list, presumably). That is a
// design decision for the operator, not one to invent here.
func LoadFromConfig() {
	type candidate struct {
		name     string
		priority int
	}
	var chosen []candidate

	for _, name := range All() {
		key := "providers." + name
		// Only an explicit entry in the config file counts. See above.
		if !viper.InConfig(key+".enabled") || !viper.GetBool(key+".enabled") {
			continue
		}
		chosen = append(chosen, candidate{
			name: name,
			// Absent priority is 0; ties are broken by name in Instances(),
			// so search order stays stable across restarts.
			priority: viper.GetInt(key + ".priority"),
		})
	}

	if len(chosen) == 0 {
		// Nothing configured: leave the registry empty so searches keep using
		// every registered provider. This is the default-install path.
		SetInstances(nil)
		return
	}

	sort.Slice(chosen, func(i, j int) bool {
		if chosen[i].priority != chosen[j].priority {
			return chosen[i].priority > chosen[j].priority
		}
		return chosen[i].name < chosen[j].name
	})

	insts := make([]Instance, 0, len(chosen))
	for _, c := range chosen {
		insts = append(insts, Instance{
			// One instance per provider is what the config shape supports, so
			// the provider name is a stable, meaningful instance ID. It also
			// keeps backoff and status keys readable.
			ID:       c.name,
			Name:     c.name,
			Enabled:  true,
			Priority: c.priority,
			Tags:     viper.GetStringSlice("providers." + c.name + ".tags"),
		})
	}
	SetInstances(insts)
}
