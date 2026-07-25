// file: pkg/webserver/profiles.go
// version: 1.1.0
// guid: 0a9b8c7d-6e5f-1a2b-4c3d-7e6f8a9b0c1d
// last-edited: 2026-07-25

package webserver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/profiles"
)

// profilesHandler handles language profile management endpoints.
// Supports:
// GET /api/profiles - List all language profiles
// POST /api/profiles - Create a new language profile
// GET /api/profiles/{id} - Get a specific language profile
// PUT /api/profiles/{id} - Update a language profile
// DELETE /api/profiles/{id} - Delete a language profile
// POST /api/profiles/{id}/default - Set as default profile
func profilesHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse URL path: /api/profiles or /api/profiles/{id} or /api/profiles/{id}/default
		path := strings.TrimPrefix(r.URL.Path, "/api/profiles")

		if path == "" || path == "/" {
			// Handle collection operations
			switch r.Method {
			case http.MethodGet:
				handleListProfiles(w, r, db)
			case http.MethodPost:
				handleCreateProfile(w, r, db)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}

		// Extract profile ID and action
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		profileID := parts[0]
		action := ""
		if len(parts) > 1 {
			action = parts[1]
		}

		if action == "default" {
			// Handle setting default profile
			if r.Method == http.MethodPost {
				handleSetDefaultProfile(w, r, db, profileID)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}

		// Handle individual profile operations
		switch r.Method {
		case http.MethodGet:
			handleGetProfile(w, r, db, profileID)
		case http.MethodPut:
			handleUpdateProfile(w, r, db, profileID)
		case http.MethodDelete:
			handleDeleteProfile(w, r, db, profileID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// handleListProfiles returns all language profiles.
// GET /api/profiles
func handleListProfiles(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	store, err := database.OpenStoreWithConfig()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	profiles, err := store.ListLanguageProfiles()
	if err != nil {
		http.Error(w, "Failed to list profiles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

// handleCreateProfile creates a new language profile.
// POST /api/profiles
func handleCreateProfile(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var profile profiles.LanguageProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Generate ID if not provided
	if profile.ID == "" {
		profile.ID = uuid.NewString()
	}

	// Set timestamps
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	// Validate profile
	if err := profile.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	store, err := database.OpenStoreWithConfig()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := store.CreateLanguageProfile(ConvertProfilesToDatabase(&profile)); err != nil {
		http.Error(w, "Failed to create profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(profile)
}

// handleGetProfile returns a specific language profile.
// GET /api/profiles/{id}
func handleGetProfile(w http.ResponseWriter, r *http.Request, db *sql.DB, profileID string) {
	store, err := database.OpenStoreWithConfig()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	profile, err := store.GetLanguageProfile(profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Profile not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get profile", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// handleUpdateProfile updates an existing language profile.
// PUT /api/profiles/{id}
func handleUpdateProfile(w http.ResponseWriter, r *http.Request, db *sql.DB, profileID string) {
	var profile profiles.LanguageProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Ensure ID matches URL
	profile.ID = profileID
	profile.UpdatedAt = time.Now()

	// Validate profile
	if err := profile.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	store, err := database.OpenStoreWithConfig()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := store.UpdateLanguageProfile(ConvertProfilesToDatabase(&profile)); err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// handleDeleteProfile deletes a language profile.
// DELETE /api/profiles/{id}
func handleDeleteProfile(w http.ResponseWriter, r *http.Request, db *sql.DB, profileID string) {
	store, err := database.OpenStoreWithConfig()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	// Check if this is the default profile
	defaultProfile, err := store.GetDefaultLanguageProfile()
	if err == nil && defaultProfile.ID == profileID {
		http.Error(w, "Cannot delete the default profile", http.StatusBadRequest)
		return
	}

	if err := store.DeleteLanguageProfile(profileID); err != nil {
		http.Error(w, "Failed to delete profile", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleSetDefaultProfile sets a profile as the default.
// POST /api/profiles/{id}/default
func handleSetDefaultProfile(w http.ResponseWriter, r *http.Request, db *sql.DB, profileID string) {
	store, err := database.OpenStoreWithConfig()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := store.SetDefaultLanguageProfile(profileID); err != nil {
		http.Error(w, "Failed to set default profile", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// mediaProfilesHandler handles media profile assignment endpoints.
// Supports:
// GET /api/media/profile/{id} - Get profile assigned to media
// PUT /api/media/profile/{id} - Assign profile to media
// DELETE /api/media/profile/{id} - Remove profile assignment
func mediaProfilesHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse URL path: /api/media/profile/{id}
		path := strings.TrimPrefix(r.URL.Path, "/api/media/profile/")
		if path == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mediaID := strings.Split(path, "/")[0]

		switch r.Method {
		case http.MethodGet:
			handleGetMediaProfile(w, r, db, mediaID)
		case http.MethodPut:
			handleAssignMediaProfile(w, r, db, mediaID)
		case http.MethodDelete:
			handleRemoveMediaProfile(w, r, db, mediaID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// handleGetMediaProfile returns the profile assigned to a media item.
// GET /api/media/profile/{id}
func handleGetMediaProfile(w http.ResponseWriter, r *http.Request, db *sql.DB, mediaID string) {
	store, err := database.OpenStoreWithConfig()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	profile, err := store.GetMediaProfile(mediaID)
	if err != nil {
		http.Error(w, "Failed to get media profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// handleAssignMediaProfile assigns a profile to a media item.
// PUT /api/media/profile/{id}
func handleAssignMediaProfile(w http.ResponseWriter, r *http.Request, db *sql.DB, mediaID string) {
	var request struct {
		ProfileID string `json:"profile_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.ProfileID == "" {
		http.Error(w, "Profile ID is required", http.StatusBadRequest)
		return
	}

	store, err := database.OpenStoreWithConfig()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	// Verify profile exists
	_, err = store.GetLanguageProfile(request.ProfileID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Profile not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to verify profile", http.StatusInternalServerError)
		}
		return
	}

	if err := store.AssignProfileToMedia(mediaID, request.ProfileID); err != nil {
		http.Error(w, "Failed to assign profile", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleRemoveMediaProfile removes profile assignment from a media item.
// DELETE /api/media/profile/{id}
func handleRemoveMediaProfile(w http.ResponseWriter, r *http.Request, db *sql.DB, mediaID string) {
	store, err := database.OpenStoreWithConfig()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := store.RemoveProfileFromMedia(mediaID); err != nil {
		http.Error(w, "Failed to remove profile assignment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ConvertProfilesToDatabase converts a profiles.LanguageProfile to a database.LanguageProfile.
func ConvertProfilesToDatabase(p *profiles.LanguageProfile) *database.LanguageProfile {
	if p == nil {
		return nil
	}
	return &database.LanguageProfile{
		ID:          p.ID,
		Name:        p.Name,
		Languages:   convertLanguages(p.Languages),
		CutoffScore: p.CutoffScore,
		IsDefault:   p.IsDefault,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func convertLanguages(src []profiles.LanguageConfig) []profiles.LanguageConfig {
	out := make([]profiles.LanguageConfig, len(src))
	for i, l := range src {
		out[i] = profiles.LanguageConfig{
			Language: l.Language,
			Priority: l.Priority,
			Forced:   l.Forced,
			HI:       l.HI,
		}
	}
	return out
}

// extractIDFromPath extracts the ID from a URL path.
func extractIDFromPath(path, prefix string) string {
	if len(path) <= len(prefix) {
		return ""
	}
	id := path[len(prefix):]
	// Handle paths with trailing slashes or additional segments
	for i, c := range id {
		if c == '/' {
			return id[:i]
		}
	}
	return id
}

// methodRouter routes requests based on HTTP method.
func methodRouter(methods map[string]http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler, ok := methods[r.Method]; ok {
			handler.ServeHTTP(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// rewritePrefix returns a handler that replaces the leading from segment of
// the request path with to before delegating to h.
//
// It exists so /api/language-profiles (what the frontend calls) and
// /api/profiles (what the handlers parse) can share one implementation
// instead of being kept in sync by hand. The request is cloned rather than
// mutated: the original *http.Request is visible to middleware up the chain,
// and rewriting its URL in place would corrupt their view of the path.
func rewritePrefix(from, to string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, from) {
			h.ServeHTTP(w, r)
			return
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = to + strings.TrimPrefix(r.URL.Path, from)
		h.ServeHTTP(w, r2)
	})
}

