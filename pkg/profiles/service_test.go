// file: pkg/profiles/service_test.go
// version: 1.1.0
// last-edited: 2026-08-12
// guid: d5e6f7a8-9b0c-1d2e-3f4a-5b6c7d8e9f0a

package profiles

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestServiceIntegration(t *testing.T) {
	// Skip if SQLite not available
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Skip("SQLite not available")
	}
	defer db.Close()

	// Initialize basic schema for testing
	initTestSchema(db)

	service := NewService(db)

	t.Run("CreateAndGetProfile", func(t *testing.T) {
		profile := &LanguageProfile{
			Name:        "Test Profile",
			CutoffScore: 85,
			IsDefault:   false,
			Languages: []LanguageConfig{
				{Language: "en", Priority: 1, Forced: false, HI: false},
				{Language: "es", Priority: 2, Forced: true, HI: false},
			},
		}

		// Create profile
		err := service.CreateProfile(profile)
		if err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		if profile.ID == "" {
			t.Error("Profile ID should be set after creation")
		}

		// Get profile
		retrieved, err := service.GetProfile(profile.ID)
		if err != nil {
			t.Fatalf("Failed to get profile: %v", err)
		}

		if retrieved.Name != profile.Name {
			t.Errorf("Profile name mismatch: got %s, want %s", retrieved.Name, profile.Name)
		}

		if len(retrieved.Languages) != len(profile.Languages) {
			t.Errorf("Languages count mismatch: got %d, want %d", len(retrieved.Languages), len(profile.Languages))
		}
	})

	t.Run("DefaultProfile", func(t *testing.T) {
		// Get default profile (should create one if none exists)
		defaultProfile, err := service.GetDefaultProfile()
		if err != nil {
			t.Fatalf("Failed to get default profile: %v", err)
		}

		if !defaultProfile.IsDefault {
			t.Error("Default profile should have IsDefault = true")
		}

		if defaultProfile.Name != "Default" {
			t.Errorf("Default profile name: got %s, want 'Default'", defaultProfile.Name)
		}
	})

	t.Run("ListProfiles", func(t *testing.T) {
		profiles, err := service.ListProfiles()
		if err != nil {
			t.Fatalf("Failed to list profiles: %v", err)
		}

		if len(profiles) < 1 {
			t.Error("Should have at least one profile (default)")
		}

		// Default profile should be first
		if len(profiles) > 0 && !profiles[0].IsDefault {
			t.Error("First profile should be the default profile")
		}
	})

	t.Run("UpdateProfile", func(t *testing.T) {
		// Create a profile to update
		profile := &LanguageProfile{
			Name:        "Update Test",
			CutoffScore: 75,
			IsDefault:   false,
			Languages: []LanguageConfig{
				{Language: "fr", Priority: 1, Forced: false, HI: false},
			},
		}

		err := service.CreateProfile(profile)
		if err != nil {
			t.Fatalf("Failed to create profile for update test: %v", err)
		}

		// Update the profile
		profile.Name = "Updated Name"
		profile.CutoffScore = 90
		profile.Languages = append(profile.Languages, LanguageConfig{
			Language: "de", Priority: 2, Forced: true, HI: true,
		})

		err = service.UpdateProfile(profile)
		if err != nil {
			t.Fatalf("Failed to update profile: %v", err)
		}

		// Verify the update
		updated, err := service.GetProfile(profile.ID)
		if err != nil {
			t.Fatalf("Failed to get updated profile: %v", err)
		}

		if updated.Name != "Updated Name" {
			t.Errorf("Profile name not updated: got %s, want 'Updated Name'", updated.Name)
		}

		if updated.CutoffScore != 90 {
			t.Errorf("Cutoff score not updated: got %d, want 90", updated.CutoffScore)
		}

		if len(updated.Languages) != 2 {
			t.Errorf("Languages not updated: got %d, want 2", len(updated.Languages))
		}
	})

	t.Run("DeleteProfile", func(t *testing.T) {
		// Create a profile to delete
		profile := &LanguageProfile{
			Name:        "Delete Test",
			CutoffScore: 80,
			IsDefault:   false,
			Languages: []LanguageConfig{
				{Language: "ja", Priority: 1, Forced: false, HI: false},
			},
		}

		err := service.CreateProfile(profile)
		if err != nil {
			t.Fatalf("Failed to create profile for delete test: %v", err)
		}

		// Delete the profile
		err = service.DeleteProfile(profile.ID)
		if err != nil {
			t.Fatalf("Failed to delete profile: %v", err)
		}

		// Verify deletion
		_, err = service.GetProfile(profile.ID)
		if err == nil {
			t.Error("Profile should not exist after deletion")
		}
	})
}

// initTestSchema creates minimal database schema for testing
func initTestSchema(db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS language_profiles (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            config TEXT NOT NULL,
            cutoff_score INTEGER DEFAULT 80,
            is_default BOOLEAN DEFAULT FALSE,
            created_at TIMESTAMP NOT NULL,
            updated_at TIMESTAMP NOT NULL
        )`,
		`CREATE TABLE IF NOT EXISTS media_profiles (
            media_id TEXT NOT NULL,
            profile_id TEXT NOT NULL,
            created_at TIMESTAMP NOT NULL,
            PRIMARY KEY (media_id),
            FOREIGN KEY (profile_id) REFERENCES language_profiles(id) ON DELETE CASCADE
        )`,
		`CREATE TABLE IF NOT EXISTS media_items (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            path TEXT NOT NULL,
            title TEXT NOT NULL,
            season INTEGER,
            episode INTEGER,
            release_group TEXT,
            alt_titles TEXT,
            field_locks TEXT,
            created_at TIMESTAMP NOT NULL
        )`,
	}

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
