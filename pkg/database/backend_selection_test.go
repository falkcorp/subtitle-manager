// file: pkg/database/backend_selection_test.go
// version: 1.1.0
// last-edited: 2026-08-12
// guid: 1a2b3c4d-5e6f-7890-1234-567890abcdef

package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackendSelectionAndCGOSupport tests backend selection logic for both CGO and pure Go builds
func TestBackendSelectionAndCGOSupport(t *testing.T) {
	t.Run("SQLite availability detection", func(t *testing.T) {
		hasSQL := HasSQLite()
		t.Logf("Build configuration: SQLite support = %t", hasSQL)

		// The HasSQLite() function should accurately reflect build configuration
		if hasSQL {
			t.Log("✓ Built with CGO and 'sqlite' build tag - SQLite should be available")
		} else {
			t.Log("✓ Built without CGO or 'sqlite' build tag - using pure Go mode")
		}
	})

	t.Run("Backend compatibility matrix", func(t *testing.T) {
		tempDir := t.TempDir()

		tests := []struct {
			backend     string
			shouldWork  func() bool
			description string
		}{
			{
				backend:     "pebble",
				shouldWork:  func() bool { return true }, // Always works
				description: "Pebble (pure Go) - should always work",
			},
			{
				backend:     "sqlite",
				shouldWork:  func() bool { return HasSQLite() }, // Only works with CGO
				description: "SQLite - depends on CGO build configuration",
			},
			{
				backend:     "postgres",
				shouldWork:  func() bool { return false }, // Never works without connection
				description: "PostgreSQL - will fail without connection string",
			},
		}

		for _, tt := range tests {
			testPath := tempDir + "/" + tt.backend
			if tt.backend == "sqlite" {
				// Use in-memory SQLite for testing (common pattern)
				testPath = ":memory:"
			}
			t.Run(tt.description, func(t *testing.T) {
				store, err := OpenStore(testPath, tt.backend)

				if tt.shouldWork() {
					require.NoError(t, err, "Backend %s should work in current build configuration", tt.backend)
					require.NotNil(t, store, "Store should not be nil when backend works")

					// Test basic functionality
					err = store.InsertTag("test-tag-" + tt.backend)
					assert.NoError(t, err, "Basic store operation should work")

					store.Close()
				} else {
					require.Error(t, err, "Backend %s should fail in current build configuration", tt.backend)
					assert.Nil(t, store, "Store should be nil when backend fails")
					t.Logf("Expected failure for %s: %v", tt.backend, err)
				}
			})
		}
	})

	t.Run("Default backend behavior", func(t *testing.T) {
		tempDir := t.TempDir()

		// Use in-memory SQLite for testing when available
		testPath := ":memory:"
		if !HasSQLite() {
			// For pure Go builds, use a directory path for Pebble fallback test
			testPath = tempDir + "/test.db"
		}

		// Test default backend (should be SQLite, but will fall back based on availability)
		store, err := OpenStore(testPath, "default")

		if HasSQLite() {
			require.NoError(t, err, "Default backend should work when SQLite is available")
			require.NotNil(t, store)
			store.Close()
			t.Log("✓ Default backend uses SQLite (CGO build)")
		} else {
			require.Error(t, err, "Default backend should fail when SQLite is not available")
			assert.Nil(t, store)
			t.Log("✓ Default backend fails without SQLite (pure Go build)")

			// In pure Go builds, explicitly use Pebble
			store, err = OpenStore(tempDir+"/pebble", "pebble")
			require.NoError(t, err, "Pebble should work as fallback in pure Go builds")
			require.NotNil(t, store)
			store.Close()
			t.Log("✓ Pebble works as fallback in pure Go builds")
		}
	})

	t.Run("Build instructions validation", func(t *testing.T) {
		// There is one build configuration now; HasSQLite() is always true.
		require.True(t, HasSQLite(), "SQLite must be available in every build")
		t.Log("✓ Pure Go build:")
		t.Log("  - Built with: CGO_ENABLED=0 go build (no build tags)")
		t.Log("  - SQLite driver: modernc.org/sqlite (pure Go)")
		t.Log("  - Both SQLite and Pebble backends available, chosen via db_backend")
	})
}

// TestCrossBackendCompatibility tests data compatibility between different backends
func TestCrossBackendCompatibility(t *testing.T) {
	if !HasSQLite() {
		t.Skip("Cross-backend compatibility test requires SQLite support")
	}

	tempDir := t.TempDir()

	// Test data that should work across all backends
	testTag := "cross-backend-test"

	t.Run("SQLite to Pebble migration simulation", func(t *testing.T) {
		// This would test migration scenarios, but for now just test basic compatibility
		sqliteDir := tempDir + "/sqlite"
		pebbleDir := tempDir + "/pebble"

		// Create data in SQLite
		{
			store, err := OpenStore(sqliteDir, "sqlite")
			require.NoError(t, err)

			err = store.InsertTag(testTag)
			require.NoError(t, err)

			store.Close()
		}

		// Create same data in Pebble
		{
			store, err := OpenStore(pebbleDir, "pebble")
			require.NoError(t, err)

			err = store.InsertTag(testTag)
			require.NoError(t, err)

			store.Close()
		}

		// Both should persist the data correctly
		{
			sqliteStore, err := OpenStore(sqliteDir, "sqlite")
			require.NoError(t, err)
			defer sqliteStore.Close()

			pebbleStore, err := OpenStore(pebbleDir, "pebble")
			require.NoError(t, err)
			defer pebbleStore.Close()

			sqliteTags, err := sqliteStore.ListTags()
			require.NoError(t, err)

			pebbleTags, err := pebbleStore.ListTags()
			require.NoError(t, err)

			// Both backends should have the same tag
			require.Len(t, sqliteTags, 1)
			require.Len(t, pebbleTags, 1)
			assert.Equal(t, testTag, sqliteTags[0].Name)
			assert.Equal(t, testTag, pebbleTags[0].Name)
		}
	})
}
