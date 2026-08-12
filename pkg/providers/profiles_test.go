// file: pkg/providers/profiles_test.go
// version: 1.1.0
// last-edited: 2026-08-12
// guid: a1b2c3d4-5e6f-7a8b-9c0d-1e2f3a4b5c6d

package providers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jdfalk/subtitle-manager/pkg/profiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func initTestSchema(db *sql.DB) {
	// Basic schema for testing
	db.Exec(`CREATE TABLE IF NOT EXISTS media_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL,
		title TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS language_profiles (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		config TEXT NOT NULL,
		cutoff_score INTEGER DEFAULT 80,
		is_default BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS media_profiles (
		media_id TEXT NOT NULL,
		profile_id TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		PRIMARY KEY (media_id)
	)`)
}

func TestGetLanguagesFromProfile(t *testing.T) {
	// Skip if SQLite not available
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Skip("SQLite not available")
	}
	defer db.Close()

	initTestSchema(db)
	service := profiles.NewService(db)

	// Create a test profile
	profile := &profiles.LanguageProfile{
		Name: "Test Profile",
		Languages: []profiles.LanguageConfig{
			{Language: "en", Priority: 1},
			{Language: "es", Priority: 2},
			{Language: "fr", Priority: 3},
		},
		CutoffScore: 80,
		IsDefault:   true,
	}

	err = service.CreateProfile(profile)
	require.NoError(t, err)

	// Create a test media item
	_, err = db.Exec(`INSERT INTO media_items (path, title, created_at) VALUES (?, ?, datetime('now'))`, "/test/movie.mkv", "Test Movie")
	require.NoError(t, err)

	var mediaID string
	err = db.QueryRow(`SELECT id FROM media_items WHERE path = ?`, "/test/movie.mkv").Scan(&mediaID)
	require.NoError(t, err)

	// Assign profile to media
	err = service.AssignProfileToMedia(mediaID, profile.ID)
	require.NoError(t, err)

	// Test getting languages from profile
	languages, err := GetLanguagesFromProfile(context.Background(), db, "/test/movie.mkv")
	require.NoError(t, err)

	expected := []string{"en", "es", "fr"}
	assert.Equal(t, expected, languages)
}

func TestGetLanguagesFromProfile_NoProfile(t *testing.T) {
	// Skip if SQLite not available
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Skip("SQLite not available")
	}
	defer db.Close()

	initTestSchema(db)

	// Test with non-existent media path - should use/create default profile
	languages, err := GetLanguagesFromProfile(context.Background(), db, "/nonexistent/movie.mkv")

	// Should get languages from default profile (which will be created automatically)
	assert.NoError(t, err)              // No error because default profile gets created
	assert.NotEmpty(t, languages)       // Should have at least one language
	assert.Contains(t, languages, "en") // Default profile should contain English
}

// Note: Full integration tests for FetchWithProfile and FetchWithProfileTagged
// would require mock providers and are complex to set up. The core logic is tested
// through the GetLanguagesFromProfile test and the existing profiles package tests.
