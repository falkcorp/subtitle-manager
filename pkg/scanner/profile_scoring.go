// file: pkg/scanner/profile_scoring.go
// version: 1.0.0
// guid: 3b7e02c9-14a6-4d58-9e07-8c15f2a4d6b3
// last-edited: 2026-08-04

package scanner

import (
	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/scoring"
	"github.com/jdfalk/subtitle-manager/pkg/security"
)

// applyAssignedProfileOverrides layers a media file's assigned language profile
// on top of the global scoring profile.
//
// # Why this exists
//
// A language profile carries a CutoffScore and per-language Forced/HI
// preferences, and none of them reached the scorer. The download path scored
// every candidate with scoring.LoadProfileFromConfig() — the global scoring.*
// settings — so a profile that said "Spanish, forced, minimum score 90" was
// honoured only in its choice of language. The threshold and the flags were
// silently dropped, which made the profile look far more configurable than it
// was.
//
// The lookup happens here rather than being threaded through ProcessFile
// because ProcessFile has a dozen callers and only needs the media path and
// language, both of which it already has. That also means every caller benefits,
// including the single-language paths where a file happens to have a profile.
//
// A false Forced/HI is deliberately not treated as a prohibition. The flags mean
// "this is preferred", not "reject everything else"; turning a preference off
// should widen the candidate set back to the global policy, not narrow it to
// nothing.
func applyAssignedProfileOverrides(p scoring.Profile, mediaPath, lang string, store database.SubtitleStore) scoring.Profile {
	logger := logging.GetLogger("scanner")

	lookupStore := store
	if lookupStore == nil {
		var err error
		if lookupStore, err = database.GetSharedStore(); err != nil {
			return p
		}
	}

	// The assignment is keyed on the sanitized path, the same key the web
	// handler writes and assignedProfileLanguages reads.
	sanitized, err := security.ValidateAndSanitizePath(mediaPath)
	if err != nil {
		return p
	}

	assignedID, err := lookupStore.GetAssignedProfileID(sanitized)
	if err != nil || assignedID == "" {
		return p
	}
	profile, err := lookupStore.GetLanguageProfile(assignedID)
	if err != nil || profile == nil {
		return p
	}

	if profile.CutoffScore > 0 {
		p.MinScore = profile.CutoffScore
	}

	for _, l := range profile.Languages {
		if l.Language != lang {
			continue
		}
		if l.HI {
			p.AllowHI = true
			p.PreferHI = true
		}
		if l.Forced {
			p.AllowForced = true
			p.PreferForced = true
		}
		break
	}

	logger.Debugf("scoring %s (%s) with profile %s: min score %d, preferHI=%t preferForced=%t",
		sanitized, lang, profile.ID, p.MinScore, p.PreferHI, p.PreferForced)
	return p
}
