// file: pkg/scanner/profile.go
// version: 1.2.0
// guid: 4f7c2a91-8d05-4e63-b7a2-90c1e5d3846b
// last-edited: 2026-08-04

package scanner

import (
	"context"
	"sort"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/providers"
	"github.com/jdfalk/subtitle-manager/pkg/security"
)

// anyLanguageProfiles reports whether the store holds any language profiles at
// all, so a scan can skip per-file profile resolution entirely when none exist.
//
// This used to also return the default profile's ID, which callers passed back
// in so assignedProfileLanguages could treat "resolved to the default" as "not
// assigned". GetAssignedProfileID answers that directly now, so only the
// short-circuit remains.
//
// It once carried a second, load-bearing reason: PebbleStore's
// GetDefaultLanguageProfile *created* a "default" profile when the store was
// empty, and GetMediaProfile called it on every miss, so on an empty store even
// a plain lookup wrote a row — one that was then flagged default and could not
// be deleted. Pebble no longer creates on read, so this guard is now only an
// optimisation and no longer protects correctness.
func anyLanguageProfiles(store database.SubtitleStore) bool {
	list, err := store.ListLanguageProfiles()
	return err == nil && len(list) > 0
}

// assignedProfileLanguages returns the language codes a media file's assigned
// language profile asks for, highest priority first, and whether the file has
// an assignment at all.
//
// # Why "assigned" is not the same as "has a profile"
//
// GetMediaProfile falls back to the default profile when a file has no
// assignment of its own, so it never reports a miss. That is the right answer
// for "which profile governs this file", but the wrong one for deciding
// whether to change how a scan behaves: it would silently switch every file in
// the library to profile-driven downloading the moment any default exists.
//
// This used to be approximated by treating "resolved to the default" as "not
// assigned", which made a file explicitly assigned the *default* profile
// indistinguishable from an unassigned one — it got the scan's own language
// rather than that profile's list. GetAssignedProfileID reports whether a row
// exists, separately from which profile governs the file, so the explicit
// assignment is now honoured.
//
// The lookup uses the *sanitized* path. ProcessFile sanitizes before doing
// anything, and the web handler stores a filepath.Clean-ed key, so looking up
// a raw path here would miss what the UI wrote — the same key disagreement
// that made per-item assignment collide before.
func assignedProfileLanguages(path string, store database.SubtitleStore) ([]string, bool) {
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

	assignedID, err := store.GetAssignedProfileID(sanitized)
	if err != nil || assignedID == "" {
		return nil, false
	}

	profile, err := store.GetLanguageProfile(assignedID)
	if err != nil || profile == nil {
		// A dangling assignment (profile deleted out from under it) is not an
		// error worth failing the scan over — fall back to the scan's language.
		logger.Debugf("assignment for %s names missing profile %s", sanitized, assignedID)
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

// ProcessWithProfileIfAssigned runs the download pipeline using the file's own
// language profile when it has one, and reports whether it did. A caller that
// gets false back should fall through to its own single-language behaviour.
//
// # Precedence
//
// This is the single place that decides an explicit profile assignment beats
// whatever language the calling path would otherwise have used — including
// MonitoredItem.Languages, which is the monitor loop's independent list of
// desired languages.
//
// That field loses because it is *accumulated*, not curated: sync.go unions in
// every language it is ever asked for and never removes one, so it cannot
// express "stop wanting French". A profile assignment is a deliberate per-file
// choice made in the UI. An append-only accumulation should not override a
// deliberate choice.
//
// The exception is a request that names a language directly — the web
// download endpoint's ?lang=, `fetch <file> <lang>`. Those are the user asking
// for one specific thing right now, not a policy about the file, so they are
// left alone and do not call this.
func ProcessWithProfileIfAssigned(ctx context.Context, path string, providerName string,
	p providers.Provider, upgrade bool, store database.SubtitleStore) (bool, error) {
	lookupStore := store
	if lookupStore == nil {
		var err error
		if lookupStore, err = database.GetSharedStore(); err != nil {
			return false, nil
		}
	}
	if !anyLanguageProfiles(lookupStore) {
		return false, nil
	}
	langs, ok := assignedProfileLanguages(path, lookupStore)
	if !ok {
		return false, nil
	}
	return true, processWithAssignedProfile(ctx, path, langs, providerName, p, upgrade, store)
}
