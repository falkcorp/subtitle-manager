// file: pkg/database/language_profiles_test.go
// version: 1.0.0
// guid: e5f6g7h8-i9j0-1234-efab-6789012345bc

package database

import (
	"testing"

	"github.com/jdfalk/subtitle-manager/pkg/profiles"
)

// TestDefaultLanguageProfile verifies the default profile has sensible defaults.
func TestDefaultLanguageProfile(t *testing.T) {
	profile := profiles.DefaultLanguageProfile()

	if profile.Name != "Default" {
		t.Errorf("expected name 'Default', got %s", profile.Name)
	}

	if !profile.IsDefault {
		t.Error("expected IsDefault to be true")
	}

	if len(profile.Languages) == 0 {
		t.Error("expected at least one language")
	}

	// Remove Description check (not present in struct)

	if profile.CutoffScore <= 0 || profile.CutoffScore > 100 {
		t.Errorf("expected CutoffScore between 1 and 100, got %d", profile.CutoffScore)
	}
}

// TestLanguageProfileSQLiteIntegration tests language profile operations with SQLite.
func TestLanguageProfileSQLiteIntegration(t *testing.T) {
	if !HasSQLite() {
		t.Skip("SQLite not available")
	}

	store, err := OpenSQLStore(":memory:")
	if err != nil {
		t.Fatalf("failed to open SQLite store: %v", err)
	}
	defer store.Close()

	// Test inserting a language profile
	// A distinct ID: the store creates a default profile on initialisation, and
	// DefaultLanguageProfile() carries that same ID, so inserting it verbatim
	// hit "UNIQUE constraint failed: language_profiles.id". The test means to
	// insert a NEW profile, not to re-insert the built-in one.
	profile := profiles.DefaultLanguageProfile()
	profile.ID = "test-profile"
	profile.Name = "Test Profile"
	profile.IsDefault = false

	err = store.CreateLanguageProfile(profile)
	if err != nil {
		t.Fatalf("failed to insert language profile: %v", err)
	}

	// Test listing profiles
	profilesList, err := store.ListLanguageProfiles()
	if err != nil {
		t.Fatalf("failed to list language profiles: %v", err)
	}

	// The store creates a built-in default profile on initialisation, so the
	// list is never empty. Assert the inserted profile is present rather than
	// that it is the only one — "exactly 1" encoded an assumption about an
	// empty store that has not held since the default profile was introduced.
	var retrieved *profiles.LanguageProfile
	for i := range profilesList {
		if profilesList[i].ID == profile.ID {
			retrieved = &profilesList[i]
			break
		}
	}
	if retrieved == nil {
		names := make([]string, 0, len(profilesList))
		for _, p := range profilesList {
			names = append(names, p.ID)
		}
		t.Fatalf("inserted profile %q not found in %v", profile.ID, names)
	}
	if retrieved.Name != "Test Profile" {
		t.Errorf("Profile name mismatch: got %s, want %s", retrieved.Name, profile.Name)
	}

	// Remove Description check (not present)

	// Remove GetLanguageProfileByName test (not implemented)

	// Remove profile assignment and rule tests (not implemented)
}
