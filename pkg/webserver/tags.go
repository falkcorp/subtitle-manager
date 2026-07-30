// file: pkg/webserver/tags.go
// version: 1.1.0
// guid: 5321afeb-275b-4923-a773-52d629dfa245
// last-edited: 2026-07-30
package webserver

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/tagging"
)

// tagsHandler manages global tags with enhanced metadata support.
func tagsHandler(db *sql.DB) http.Handler {
	tm := tagging.NewTagManager(db)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Support filtering by entity_type and tag type
			entityType := r.URL.Query().Get("entity_type")
			tagType := r.URL.Query().Get("type")

			var tags []database.Tag
			var err error

			if tagType != "" {
				tags, err = tm.GetTagsByType(tagType)
			} else {
				tags, err = database.ListTags(db)
			}

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Filter by entity type if specified
			if entityType != "" {
				var filtered []database.Tag
				for _, tag := range tags {
					if tag.EntityType == "all" || tag.EntityType == entityType {
						filtered = append(filtered, tag)
					}
				}
				tags = filtered
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tags)

		case http.MethodPost:
			var in struct {
				Name        string `json:"name"`
				Type        string `json:"type"`
				EntityType  string `json:"entity_type"`
				Color       string `json:"color"`
				Description string `json:"description"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			tag, err := tm.CreateTag(in.Name, in.Type, in.EntityType, in.Color, in.Description)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(tag)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// tagItemHandler updates or deletes a tag by ID with enhanced metadata support.
func tagItemHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := pathSegment(r, "/api/tags/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodDelete:
			if err := database.DeleteTag(db, id); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		case http.MethodPatch:
			var in struct {
				Name        string `json:"name"`
				Type        string `json:"type"`
				EntityType  string `json:"entity_type"`
				Color       string `json:"color"`
				Description string `json:"description"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Use enhanced update if metadata provided, otherwise simple name update
			if in.Type != "" || in.EntityType != "" || in.Color != "" || in.Description != "" {
				store, err := database.OpenSQLStore(database.GetDatabasePath())
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				defer store.Close()

				if err := store.UpdateTagWithMetadata(id, in.Name, in.Type, in.EntityType, in.Color, in.Description); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			} else {
				if err := database.UpdateTag(db, id, in.Name); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}

			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// universalTagsHandler manages tag associations for any entity type.
func universalTagsHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse URL: /api/{entityType}/{id}/tags
		parts := pathSegments(r, "/api/")
		if len(parts) < 3 || parts[2] != "tags" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		entityType := parts[0]
		entityID := parts[1]

		switch r.Method {
		case http.MethodGet:
			tags, err := database.ListTagsForEntity(db, entityType, entityID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tags)

		case http.MethodPost:
			var in struct {
				TagID int64 `json:"tag_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.TagID == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if err := database.AssignTagToEntity(db, in.TagID, entityType, entityID); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)

		case http.MethodDelete:
			var in struct {
				TagID int64 `json:"tag_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.TagID == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if err := database.RemoveTagFromEntity(db, in.TagID, entityType, entityID); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// bulkTagsHandler handles bulk tagging operations.
func bulkTagsHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var in struct {
			TagIDs   []int64 `json:"tag_ids"`
			Entities []struct {
				Type string `json:"type"`
				ID   string `json:"id"`
			} `json:"entities"`
		}

		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.TagIDs) == 0 || len(in.Entities) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Convert to TagAssociation format
		var associations []database.TagAssociation
		for _, entity := range in.Entities {
			associations = append(associations, database.TagAssociation{
				EntityType: entity.Type,
				EntityID:   entity.ID,
			})
		}

		store, err := database.OpenSQLStore(database.GetDatabasePath())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer store.Close()

		if err := store.BulkAssignTags(in.TagIDs, associations); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})
}

// userTagsHandler manages tag assignments for a user (legacy compatibility wrapper).
func userTagsHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := pathSegments(r, "/api/users/")
		if len(parts) < 2 || parts[1] != "tags" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Use universal system for new implementations, legacy for backward compatibility
		entityID := strconv.FormatInt(id, 10)

		switch r.Method {
		case http.MethodGet:
			tags, err := database.ListTagsForEntity(db, "user", entityID)
			if err != nil || len(tags) == 0 {
				// Fallback to legacy system
				tags, err = database.ListTagsForUser(db, id)
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tags)

		case http.MethodPost:
			var in struct {
				TagID int64 `json:"tag_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.TagID == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Add to both systems for compatibility
			_ = database.AssignTagToEntity(db, in.TagID, "user", entityID)
			if err := database.AssignTagToUser(db, id, in.TagID); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)

		case http.MethodDelete:
			var in struct {
				TagID int64 `json:"tag_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.TagID == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Remove from both systems for compatibility
			_ = database.RemoveTagFromEntity(db, in.TagID, "user", entityID)
			if err := database.RemoveTagFromUser(db, id, in.TagID); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// mediaTagsHandler manages tag assignments for a media item (legacy compatibility wrapper).
func mediaTagsHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := pathSegments(r, "/api/media/")
		if len(parts) < 2 || parts[1] != "tags" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Use universal system for new implementations, legacy for backward compatibility
		entityID := strconv.FormatInt(id, 10)

		switch r.Method {
		case http.MethodGet:
			tags, err := database.ListTagsForEntity(db, "media", entityID)
			if err != nil || len(tags) == 0 {
				// Fallback to legacy system
				tags, err = database.ListTagsForMedia(db, id)
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tags)

		case http.MethodPost:
			var in struct {
				TagID int64 `json:"tag_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.TagID == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Add to both systems for compatibility
			_ = database.AssignTagToEntity(db, in.TagID, "media", entityID)
			if err := database.AssignTagToMedia(db, id, in.TagID); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)

		case http.MethodDelete:
			var in struct {
				TagID int64 `json:"tag_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.TagID == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Remove from both systems for compatibility
			_ = database.RemoveTagFromEntity(db, in.TagID, "media", entityID)
			if err := database.RemoveTagFromMedia(db, id, in.TagID); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// libraryTagsHandler manages tag assignments for media items identified by path.
func libraryTagsHandler(db *sql.DB) http.Handler {
	type req struct {
		Path  string `json:"path"`
		TagID int64  `json:"tag_id"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var path string
		var tagID int64
		if r.Method == http.MethodGet {
			path = r.URL.Query().Get("path")
		} else {
			var in req
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Path == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			path = in.Path
			tagID = in.TagID
			if tagID == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		if path == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		id, err := database.EnsureMediaItem(db, path)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		switch r.Method {
		case http.MethodGet:
			tags, err := database.ListTagsForMedia(db, id)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(tags)
		case http.MethodPost:
			if err := database.AssignTagToMedia(db, id, tagID); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		case http.MethodDelete:
			if err := database.RemoveTagFromMedia(db, id, tagID); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}
