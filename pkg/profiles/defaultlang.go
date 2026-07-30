// file: pkg/profiles/defaultlang.go
// version: 1.0.0
// guid: b412735d-d6e2-44a9-9f39-e338af25e137
// last-edited: 2026-07-30

package profiles

import (
	"strings"

	"github.com/spf13/viper"
)

// FallbackLanguage is used when no default language has been configured.
//
// English is the fallback of last resort rather than a policy: it is what the
// code hardcoded before this was configurable, so keeping it preserves
// behaviour for anyone who has not set a preference.
const FallbackLanguage = "en"

// DefaultLanguage returns the language to download when nothing more specific
// applies.
//
// "Nothing more specific" means: no language profile is assigned to the media
// item, or the code path in question never had one to consult — the Sonarr and
// Radarr import webhooks, for instance, receive a file path and nothing else.
//
// Before this existed, every such site hardcoded "en", so a non-English library
// fetched English subtitles on import and on any profile miss, with no setting
// anywhere to change it. That is the gap this closes.
//
// Resolution order:
//
//  1. languages.default — the explicit setting.
//  2. The first entry of languages.preferred, if that list is configured. A
//     user who has expressed an ordered preference has already answered this
//     question; making them state it twice is a way to end up with the two
//     disagreeing.
//  3. "en", preserving the previous hardcoded behaviour.
//
// This is deliberately *not* a substitute for language profiles, which remain
// the right mechanism for per-series and per-movie language choices. It is the
// answer for paths that have no profile to consult.
func DefaultLanguage() string {
	if lang := strings.TrimSpace(viper.GetString("languages.default")); lang != "" {
		return lang
	}
	for _, lang := range viper.GetStringSlice("languages.preferred") {
		if lang = strings.TrimSpace(lang); lang != "" {
			return lang
		}
	}
	return FallbackLanguage
}
