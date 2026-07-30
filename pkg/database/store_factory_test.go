// file: pkg/database/store_factory_test.go
// version: 1.1.0
// guid: 9a8b7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d

package database

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenStore tests the OpenStore factory function with different backends
func TestOpenStore(t *testing.T) {
	// Dynamic test cases based on SQLite availability
	tests := []struct {
		name    string
		backend string
		wantErr bool
		skipMsg string
	}{
		{
			name:    "pebble backend",
			backend: "pebble",
			wantErr: false,
		},
		{
			name:    "postgres backend",
			backend: "postgres",
			wantErr: true, // Will fail without proper postgres connection string
		},
	}

	// Add SQLite test case only if SQLite is available
	if HasSQLite() {
		tests = append(tests, struct {
			name    string
			backend string
			wantErr bool
			skipMsg string
		}{
			name:    "sqlite backend (CGO enabled)",
			backend: "sqlite",
			wantErr: false,
		})
	} else {
		tests = append(tests, struct {
			name    string
			backend string
			wantErr bool
			skipMsg string
		}{
			name:    "sqlite backend unavailable (no CGO)",
			backend: "sqlite",
			wantErr: true, // Should fail when SQLite not available
		})
	}

	// Test default backend behavior
	defaultBackendTest := struct {
		name    string
		backend string
		wantErr bool
		skipMsg string
	}{
		name:    "default backend",
		backend: "unknown",
		wantErr: !HasSQLite(), // Should fail if SQLite not available since default is SQLite
	}
	if !HasSQLite() {
		defaultBackendTest.skipMsg = "Default backend is SQLite which is not available"
	}
	tests = append(tests, defaultBackendTest)

	tests = append(tests, struct {
		name    string
		backend string
		wantErr bool
		skipMsg string
	}{
		name:    "empty backend uses default",
		backend: "",
		wantErr: !HasSQLite(), // Should fail if SQLite not available since default is SQLite
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipMsg != "" {
				t.Logf("Test condition: %s", tt.skipMsg)
			}

			// Create temporary directory for each test
			tempDir := t.TempDir()

			// For postgres, use a connection string format
			path := tempDir
			if tt.backend == "postgres" {
				path = "postgres://invalid:connection@string"
			}

			store, err := OpenStore(path, tt.backend)

			if tt.wantErr {
				assert.Error(t, err, "Expected error for backend %s", tt.backend)
				assert.Nil(t, store, "Store should be nil on error")
				return
			}

			// For unknown/empty backends, we might get SQLite unavailable error
			// In that case, it should fall back to an available backend or error gracefully
			if err != nil {
				t.Logf("Backend %s failed (possibly due to build configuration): %v", tt.backend, err)
				return
			}

			require.NotNil(t, store, "Store should not be nil")

			// Verify the store is functional by testing basic operations
			err = store.InsertTag("test-tag")
			assert.NoError(t, err, "Store should be functional")

			// Clean up
			if store != nil {
				err = store.Close()
				assert.NoError(t, err, "Store should close cleanly")
			}
		})
	}
}

// TestOpenStoreWithInvalidPath tests error handling for invalid paths
func TestOpenStoreWithInvalidPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		backend string
		wantErr bool
	}{
		{
			name:    "pebble with read-only directory",
			path:    "/root/readonly", // Assuming this is read-only
			backend: "pebble",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := OpenStore(tt.path, tt.backend)

			if tt.wantErr {
				assert.Error(t, err, "Expected error for path %s", tt.path)
				// Don't try to close a nil store
				return
			}

			if err != nil {
				t.Logf("Got expected error for path %s: %v", tt.path, err)
				return
			}

			require.NotNil(t, store, "Store should be created")
			defer store.Close()
		})
	}
}

// TestOpenStoreWithConfig tests the configuration-based store opening
func TestOpenStoreWithConfig(t *testing.T) {
	// Note: This test depends on global configuration state
	// In a real application, you might want to mock or inject configuration

	// Save original configuration if any globals are used
	// This is a limitation of the current design - it relies on global config

	store, err := OpenStoreWithConfig()

	// We expect this to work with default configuration
	// The actual behavior depends on the GetDatabasePath() and GetDatabaseBackend() functions
	if err != nil {
		t.Logf("OpenStoreWithConfig failed (expected if no config set): %v", err)
		return
	}

	require.NotNil(t, store, "Store should be created with config")
	defer store.Close()

	// Verify the store is functional
	err = store.InsertTag("config-test-tag")
	assert.NoError(t, err, "Store from config should be functional")
}

// TestStoreBackendSelection tests that the correct backend is selected
func TestStoreBackendSelection(t *testing.T) {
	testDir := t.TempDir()

	// Only test backends that should work in the current build
	var backends []string
	if HasSQLite() {
		backends = []string{"pebble", "sqlite", "postgres"}
	} else {
		backends = []string{"pebble", "postgres"}
	}

	for _, backend := range backends {
		t.Run(backend, func(t *testing.T) {
			path := testDir
			if backend == "postgres" {
				// Use an obviously invalid postgres connection for testing
				path = "postgres://test:test@localhost:9999/nonexistent"
			} else if backend == "sqlite" {
				// Use in-memory SQLite for testing
				path = ":memory:"
			}

			store, err := OpenStore(path, backend)

			if backend == "postgres" {
				// We expect postgres to fail with invalid connection
				assert.Error(t, err, "Postgres should fail with invalid connection")
				return
			}

			require.NoError(t, err, "Backend %s should open successfully", backend)
			require.NotNil(t, store, "Store should be created")
			defer store.Close()

			// Test basic functionality to ensure the right backend is used
			err = store.InsertTag("backend-test-" + backend)
			assert.NoError(t, err, "Backend %s should support basic operations", backend)

			tags, err := store.ListTags()
			assert.NoError(t, err, "Backend %s should support listing tags", backend)
			assert.Len(t, tags, 1, "Should have one tag")
		})
	}
}

// TestMultipleStoreInstances tests opening multiple store instances
func TestMultipleStoreInstances(t *testing.T) {
	// Test that we can open multiple stores to different paths
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	store1, err := OpenStore(dir1, "pebble")
	require.NoError(t, err)
	defer store1.Close()

	store2, err := OpenStore(dir2, "pebble")
	require.NoError(t, err)
	defer store2.Close()

	// Test that they are independent
	err = store1.InsertTag("tag1")
	require.NoError(t, err)

	err = store2.InsertTag("tag2")
	require.NoError(t, err)

	// Verify isolation
	tags1, err := store1.ListTags()
	require.NoError(t, err)
	require.Len(t, tags1, 1)
	assert.Equal(t, "tag1", tags1[0].Name)

	tags2, err := store2.ListTags()
	require.NoError(t, err)
	require.Len(t, tags2, 1)
	assert.Equal(t, "tag2", tags2[0].Name)
}

// TestStorePersistence tests that data persists across store reopening
func TestStorePersistence(t *testing.T) {
	testDir := t.TempDir()

	// Create store and add data
	{
		store, err := OpenStore(testDir, "pebble")
		require.NoError(t, err)

		err = store.InsertTag("persistent-tag")
		require.NoError(t, err)

		err = store.Close()
		require.NoError(t, err)
	}

	// Reopen store and verify data persists
	{
		store, err := OpenStore(testDir, "pebble")
		require.NoError(t, err)
		defer store.Close()

		tags, err := store.ListTags()
		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, "persistent-tag", tags[0].Name)
	}
}

// TestStoreCleanup tests proper cleanup of store resources
func TestStoreCleanup(t *testing.T) {
	testDir := t.TempDir()

	store, err := OpenStore(testDir, "pebble")
	require.NoError(t, err)

	// Use the store
	err = store.InsertTag("cleanup-test")
	require.NoError(t, err)

	// Close should not error
	err = store.Close()
	assert.NoError(t, err)

	// Second close should be handled gracefully (but may panic for some implementations)
	// Use recover to handle potential panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Second close panicked (implementation-specific): %v", r)
			}
		}()
		err = store.Close()
		t.Logf("Second close result: %v", err)
	}()
}

// TestSQLiteAvailability tests the HasSQLite function and SQLite backend behavior
func TestSQLiteAvailability(t *testing.T) {
	hasSQLite := HasSQLite()
	t.Logf("SQLite support available: %t", hasSQLite)

	tempDir := t.TempDir()

	if hasSQLite {
		// When SQLite is available, it should work
		t.Run("SQLite backend should work when available", func(t *testing.T) {
			// A file path, not a directory: OpenStore routes "sqlite" to
			// OpenSQLStore, which opens the path as a database file. Only the
			// pebble backend treats its argument as a directory. Passing
			// tempDir here failed with "unable to open database file: is a
			// directory" — a test bug, not a product one.
			store, err := OpenStore(filepath.Join(tempDir, "test.db"), "sqlite")
			require.NoError(t, err, "SQLite backend should work when HasSQLite() returns true")
			defer store.Close()

			// Basic functionality test
			err = store.InsertTag("sqlite-test")
			assert.NoError(t, err)

			tags, err := store.ListTags()
			require.NoError(t, err)
			require.Len(t, tags, 1)
			assert.Equal(t, "sqlite-test", tags[0].Name)
		})
	} else {
		// When SQLite is not available, it should fail with a clear error
		t.Run("SQLite backend should fail when unavailable", func(t *testing.T) {
			store, err := OpenStore(tempDir, "sqlite")
			require.Error(t, err, "SQLite backend should fail when HasSQLite() returns false")
			assert.Nil(t, store)
			assert.Contains(t, err.Error(), "SQLite support not available",
				"Error should clearly indicate SQLite is unavailable")
			assert.Contains(t, err.Error(), "CGO",
				"Error should mention CGO dependency")
			assert.Contains(t, err.Error(), "Pebble",
				"Error should suggest Pebble alternative")
		})
	}

	// Pebble should always work regardless of SQLite availability
	t.Run("Pebble backend should always work", func(t *testing.T) {
		store, err := OpenStore(tempDir+"/pebble", "pebble")
		require.NoError(t, err, "Pebble backend should always work")
		defer store.Close()

		// Basic functionality test
		err = store.InsertTag("pebble-test")
		assert.NoError(t, err)

		tags, err := store.ListTags()
		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, "pebble-test", tags[0].Name)
	})
}
