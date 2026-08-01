// file: pkg/scanner/profile.go
// version: 1.0.0
// guid: 4f7c2a91-8d05-4e63-b7a2-90c1e5d3846b
// last-edited: 2026-08-01

package scanner

import (
	"context"
	"sort"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/providers"
	"github.com/jdfalk/subtitle-manager/pkg/security"
)

// defaultProfileID returns the ID of the profile marked default (or "" when
// none is) and whether any profiles exist at all.
//
// It reads the list and looks for the flag rather than calling
// GetDefaultLanguageProfile, which the backends disagree about: PebbleStore
// falls back to the first profile when none is flagged and *writes a new
// "default" profile* when the store is empty, while SQLStore strictly requires
// is_default and reports a miss. A scan must not create rows as a side effect
// of looking something up — especially not a profile that then cannot be
// deleted — and it must behave the same on either backend.
//
// The "any profiles at all" answer matters for the same reason: GetMediaProfile
// itself falls back to GetDefaultLanguageProfile on a miss, so on an empty
// store even a plain lookup would create the row. Callers skip profile
// resolution entirely when there are none.
func defaultProfileID(store database.SubtitleStore) (string, bool) {
	list, err := store.ListLanguageProfiles()
	if err != nil || len(list) == 0 {
		return "", false
	}
	for _, p := range list {
		if p.IsDefault {
			return p.ID, true
		}
	}
	return "", true
}

// assignedProfileLanguages returns the language codes a media file's assigned
// language profile asks for, highest priority first, and whether the file has
// an assignment at all. defaultID is the ID from defaultProfileID, hoisted out
// of the caller's per-file loop.
//
// # Why "assigned" is not the same as "has a profile"
//
// GetMediaProfile falls back to the default profile when a file has no
// assignment of its own, so it never reports a miss. That is the right answer
// for "which profile governs this file", but the wrong one for deciding
// whether to change how a scan behaves: it would silently switch every file in
// the library to profile-driven downloading the moment any default exists.
//
// A file is therefore treated as assigned only when the profile it resolves to
// is not the default one.
//
// This has a real consequence, not a cosmetic one: a file the user explicitly
// assigned the *default* profile is indistinguishable from an unassigned file,
// and will be scanned with the scan's own language rather than that profile's
// language list. Telling the two apart needs a store method that reports
// whether a row exists, separate from which profile governs the file. Until
// then, assigning a non-default profile is the way to get profile-driven
// languages.
//
// The lookup uses the *sanitized* path. ProcessFile sanitizes before doing
// anything, and the web handler stores a filepath.Clean-ed key, so looking up
// a raw path here would miss what the UI wrote — the same key disagreement
// that made per-item assignment collide before.
func assignedProfileLanguages(path string, store database.SubtitleStore, defaultID string) ([]string, bool) {
	logger := logging.GetLogger("scanner")

	if store == nil {
		var err error
		store, err = database.GetSharedStore()
		if err != nil {
			logger.Debugf("no store for profile lookup: %v", err)
			return nil, false
		}
	}

	sanitized, err := security.ValidateAndSanitizePath(path)
	if err != nil {
		return nil, false
	}

	profile, err := store.GetMediaProfile(sanitized)
	if err != nil || profile == nil {
		return nil, false
	}

	// A resolved profile that is merely the default means "no assignment".
	if defaultID != "" && profile.ID == defaultID {
		return nil, false
	}

	ordered := make([]struct {
		code     string
		priority int
	}, 0, len(profile.Languages))
	for _, l := range profile.Languages {
		if l.Language == "" {
			continue
		}
		ordered = append(ordered, struct {
			code     string
			priority int
		}{l.Language, l.Priority})
	}
	if len(ordered) == 0 {
		return nil, false
	}
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].priority < ordered[j].priority })

	codes := make([]string, 0, len(ordered))
	for _, o := range ordered {
		codes = append(codes, o.code)
	}

	// Single-language naming writes every language to the same <video>.srt, so
	// fetching more than one would have each overwrite the last and leave only
	// whichever finished second. Honour the highest-priority language alone.
	if singleLanguageNaming() && len(codes) > 1 {
		logger.Debugf("single-language naming: %s limited to %s of %d profile languages",
			sanitized, codes[0], len(codes))
		codes = codes[:1]
	}

	return codes, true
}

// processWithAssignedProfile downloads every language the file's assigned
// profile asks for, in priority order, through the ordinary ProcessFile
// pipeline — so scoring, the Whisper fallback, post-processing, format
// conversion and download-history persistence all apply exactly as they do to
// a single-language download. The provider selection the scan was started with
// is carried through unchanged — only the language varies per iteration.
//
// Every language is attempted even if an earlier one fails: a profile lists
// languages that are all wanted, not alternatives to each other. The call
// succeeds if any language was obtained, and reports the first failure only
// when none were.
func processWithAssignedProfile(ctx context.Context, path string, langs []string,
	providerName string, p providers.Provider, upgrade bool, store database.SubtitleStore) error {
	logger := logging.GetLogger("scanner")

	var firstErr error
	got := 0
	for _, lang := range langs {
		if err := ProcessFile(ctx, path, lang, providerName, p, upgrade, store); err != nil {
			logger.Debugf("profile language %s for %s: %v", lang, path, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		got++
	}

	if got > 0 {
		return nil
	}
	return firstErr
}
