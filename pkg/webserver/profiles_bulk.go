// file: pkg/webserver/profiles_bulk.go
// version: 1.0.0
// guid: 5f2c1a7e-9b34-4d68-a0e1-3c7d92f45b18
// last-edited: 2026-08-04

package webserver

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/jdfalk/subtitle-manager/pkg/database"
)

// maxBulkProfileItems caps one bulk assignment request.
//
// The work is one store write per item with no batching underneath, so an
// unbounded list is an unbounded request: a client could pin a request
// goroutine (and, on SQLite, the write lock) for as long as it liked. 1000 is
// far above any plausible mass-edit from the library UI — which selects within
// a single directory listing — while still being a bound.
const maxBulkProfileItems = 1000

// bulkProfileRequest is the body of POST /api/media/profiles/bulk.
type bulkProfileRequest struct {
	// ProfileID is the profile to assign to every listed item.
	//
	// The empty string is meaningful and is NOT an error: it clears the
	// assignment, mirroring DELETE on the single-item route. Bazarr's mass-edit
	// offers "None" as a profile choice and this is the equivalent; without it
	// the only way to unassign in bulk would be N separate DELETEs.
	ProfileID string `json:"profile_id"`
	// MediaIDs are the media file paths to operate on.
	MediaIDs []string `json:"media_ids"`
}

// bulkProfileItemResult reports the outcome for a single media item.
type bulkProfileItemResult struct {
	MediaID string `json:"media_id"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

// bulkProfileResponse is returned with HTTP 200 whenever the request itself was
// well-formed, even if every individual assignment failed.
//
// Partial failure is a normal outcome here, not a transport error: media paths
// come from a directory listing that another process can change underneath us.
// Collapsing that into a single status code would leave the UI unable to say
// which items landed, so the status code describes the *request* and the body
// describes the *items*.
type bulkProfileResponse struct {
	ProfileID string                  `json:"profile_id"`
	Succeeded int                     `json:"succeeded"`
	Failed    int                     `json:"failed"`
	Results   []bulkProfileItemResult `json:"results"`
}

// bulkMediaProfilesHandler assigns (or clears) one language profile across many
// media items in a single request.
//
// POST /api/media/profiles/bulk
//
//	{"profile_id": "<id or empty to clear>", "media_ids": ["/media/a.mkv", ...]}
//
// # Why this is its own exact route
//
// It deliberately does NOT live inside mediaProfilesHandler behind a string
// comparison. That handler treats everything after /api/media/profile/ as a
// media path, so a "bulk" literal there would be ambiguous with a real file
// named "bulk" — and the path-as-identifier parsing on that route has already
// produced one collision bug (see pathRest).
//
// Note the plural: the single-item route is /api/media/profile/ and this is
// /api/media/profiles/bulk, so neither is a prefix of the other.
//
// Registration matters as much as the code. /api/media/ is registered as a
// SUBTREE for mediaTagsHandler, so without an exact pattern of its own this
// path would be silently answered by the tag handler — a 200 with the wrong
// body rather than a 404. TestBulkProfileRouteIsNotSwallowedByTags pins that.
func bulkMediaProfilesHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req bulkProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if len(req.MediaIDs) == 0 {
			http.Error(w, "media_ids is required", http.StatusBadRequest)
			return
		}
		if len(req.MediaIDs) > maxBulkProfileItems {
			http.Error(w, "too many media_ids", http.StatusRequestEntityTooLarge)
			return
		}

		store, err := database.GetSharedStore()
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Validate the profile ONCE rather than per item. It is the same
		// profile for every item, so a bad ID is a property of the request,
		// not of any particular file — reporting it as N identical per-item
		// errors would bury a single fixable mistake in a results array.
		clearing := req.ProfileID == ""
		if !clearing {
			if _, err := store.GetLanguageProfile(req.ProfileID); err != nil {
				if err == sql.ErrNoRows {
					http.Error(w, "Profile not found", http.StatusNotFound)
				} else {
					http.Error(w, "Failed to verify profile", http.StatusInternalServerError)
				}
				return
			}
		}

		resp := bulkProfileResponse{
			ProfileID: req.ProfileID,
			Results:   make([]bulkProfileItemResult, 0, len(req.MediaIDs)),
		}

		for _, raw := range req.MediaIDs {
			// Normalise with the same helper the single-item route uses, so a
			// bulk assignment and a per-item assignment of the same file land
			// on the same storage key — and so the scanner, which looks up the
			// cleaned path, finds either.
			mediaID := normalizeMediaID(raw)

			// Results stay 1:1 with the submitted list, in order, including
			// duplicates. The UI zips them against its own selection, so
			// silently collapsing entries would misalign that mapping; the
			// underlying writes are idempotent, so a repeat is harmless.
			item := bulkProfileItemResult{MediaID: raw}
			switch {
			case mediaID == "":
				item.Error = "empty media id"
			case clearing:
				if err := store.RemoveProfileFromMedia(mediaID); err != nil {
					item.Error = "failed to remove profile assignment"
				} else {
					item.OK = true
				}
			default:
				if err := store.AssignProfileToMedia(mediaID, req.ProfileID); err != nil {
					item.Error = "failed to assign profile"
				} else {
					item.OK = true
				}
			}

			if item.OK {
				resp.Succeeded++
			} else {
				resp.Failed++
			}
			resp.Results = append(resp.Results, item)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}
