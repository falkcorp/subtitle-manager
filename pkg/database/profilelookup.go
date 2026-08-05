// file: pkg/database/profilelookup.go
// version: 1.0.0
// guid: 0a6d24b7-8f19-4c53-b2e0-5d719f3ac846
// last-edited: 2026-08-04

package database

import (
	"database/sql"
	"errors"
)

// ProfileMissing reports whether a GetLanguageProfile result means "there is no
// such profile".
//
// # Why this is needed
//
// The implementations disagree about how a miss is signalled, and nothing in
// the interface said so:
//
//	SQLStore / PostgresStore  ->  (nil, sql.ErrNoRows)
//	PebbleStore              ->  (nil, nil)
//
// Every caller was written against the SQL behaviour — "err != nil means not
// found" — which silently does the wrong thing on Pebble, the *default*
// backend. The bulk profile-assign endpoint accepted a nonexistent profile ID
// and reported success for every item, writing assignments that point at
// nothing; verified live on a Pebble store before this was fixed.
//
// It is easy for a test to miss: the webserver profile tests configure SQLite,
// so they exercise the one backend where the naive check happens to work.
//
// Normalising here rather than changing PebbleStore keeps the blast radius to
// the callers that ask this question. GetMediaProfile returns
// GetLanguageProfile's result directly, so making Pebble return an error would
// change what a dangling assignment looks like to several unrelated callers.
func ProfileMissing(profile *LanguageProfile, err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	return err == nil && profile == nil
}
