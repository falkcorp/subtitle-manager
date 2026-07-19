// file: pkg/gcommon/config/config_test.go
// version: 1.0.0
// guid: af6f4b9b-3c1b-4c2e-99f5-d45e6b7c8d9f

package config

import (
	"os"
	"path/filepath"
	"testing"

	common "github.com/falkcorp/gcommon/v2/pkg/commonpb/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad tests the Load function with various configurations
func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		cfgFile    string
		setupEnv   func()
		setupFiles func(t *testing.T) string // returns temp dir
		wantError  bool
		validate   func(t *testing.T)
	}{
		{
			name:    "load with explicit config file",
			cfgFile: "test-config.yaml",
			setupFiles: func(t *testing.T) string {
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "test-config.yaml")
				content := `db_path: /custom/db
db_backend: sqlite
log_file: /custom/logs/app.log
`
				err := os.WriteFile(configPath, []byte(content), 0644)
				require.NoError(t, err)
				return configPath
			},
			validate: func(t *testing.T) {
				assert.Equal(t, "/custom/db", viper.GetString("db_path"))
				assert.Equal(t, "sqlite", viper.GetString("db_backend"))
				assert.Equal(t, "/custom/logs/app.log", viper.GetString("log_file"))
			},
		},
		{
			name: "load with environment variables",
			setupEnv: func() {
				os.Setenv("SM_DB_PATH", "/env/db")
				os.Setenv("SM_DB_BACKEND", "pebble")
				os.Setenv("SM_GOOGLE_API_KEY", "env-google-key")
			},
			validate: func(t *testing.T) {
				assert.Equal(t, "/env/db", viper.GetString("db_path"))
				assert.Equal(t, "pebble", viper.GetString("db_backend"))
				assert.Equal(t, "env-google-key", viper.GetString("google_api_key"))
			},
		},
		{
			name: "load with defaults when no config file",
			validate: func(t *testing.T) {
				// Should have default values set
				dbPath := viper.GetString("db_path")
				assert.NotEmpty(t, dbPath)
				assert.Contains(t, dbPath, ".subtitle-manager")

				assert.Equal(t, "pebble", viper.GetString("db_backend"))
				assert.Equal(t, "subtitle-manager.db", viper.GetString("sqlite3_filename"))
				assert.Equal(t, "/config/logs/subtitle-manager.log", viper.GetString("log_file"))
			},
		},
		{
			name: "load with config file from environment",
			setupEnv: func() {
				// This would be set by SM_CONFIG_FILE environment variable
				// but we'll test it indirectly
				viper.Set("config_file", "should-not-exist.yaml")
			},
			validate: func(t *testing.T) {
				// Should still load defaults even if config file doesn't exist
				assert.Equal(t, "pebble", viper.GetString("db_backend"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper state before each test
			viper.Reset()

			// Clean up environment variables
			defer func() {
				os.Unsetenv("SM_DB_PATH")
				os.Unsetenv("SM_DB_BACKEND")
				os.Unsetenv("SM_GOOGLE_API_KEY")
				os.Unsetenv("SM_CONFIG_FILE")
			}()

			// Setup environment if needed
			if tt.setupEnv != nil {
				tt.setupEnv()
			}

			// Setup files if needed
			var cfgFile string
			if tt.setupFiles != nil {
				cfgFile = tt.setupFiles(t)
			} else {
				cfgFile = tt.cfgFile
			}

			// Create a test command
			cmd := &cobra.Command{}

			// Test the Load function
			err := Load(cmd, cfgFile)

			if tt.wantError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Validate results
			if tt.validate != nil {
				tt.validate(t)
			}
		})
	}
}

// TestToProto tests the ToProto function
func TestToProto(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		validate func(t *testing.T, result map[string]*common.ConfigValue)
	}{
		{
			name: "all config values set",
			setup: func() {
				viper.Reset()
				viper.Set("db_path", "/test/db")
				viper.Set("db_backend", "sqlite")
				viper.Set("sqlite3_filename", "test.db")
				viper.Set("log_file", "/test/logs/app.log")
				viper.Set("google_api_key", "test-google-key")
				viper.Set("openai_api_key", "test-openai-key")
			},
			validate: func(t *testing.T, result map[string]*common.ConfigValue) {
				assert.Len(t, result, 6)

				assert.NotNil(t, result["db_path"])
				assert.Equal(t, "/test/db", result["db_path"].GetStringValue())

				assert.NotNil(t, result["db_backend"])
				assert.Equal(t, "sqlite", result["db_backend"].GetStringValue())

				assert.NotNil(t, result["sqlite3_filename"])
				assert.Equal(t, "test.db", result["sqlite3_filename"].GetStringValue())

				assert.NotNil(t, result["log_file"])
				assert.Equal(t, "/test/logs/app.log", result["log_file"].GetStringValue())

				assert.NotNil(t, result["google_api_key"])
				assert.Equal(t, "test-google-key", result["google_api_key"].GetStringValue())

				assert.NotNil(t, result["openai_api_key"])
				assert.Equal(t, "test-openai-key", result["openai_api_key"].GetStringValue())
			},
		},
		{
			name: "partial config values",
			setup: func() {
				viper.Reset()
				viper.Set("db_path", "/partial/db")
				viper.Set("google_api_key", "partial-key")
			},
			validate: func(t *testing.T, result map[string]*common.ConfigValue) {
				assert.Len(t, result, 2)

				assert.NotNil(t, result["db_path"])
				assert.Equal(t, "/partial/db", result["db_path"].GetStringValue())

				assert.NotNil(t, result["google_api_key"])
				assert.Equal(t, "partial-key", result["google_api_key"].GetStringValue())

				// These should not be present
				assert.Nil(t, result["db_backend"])
				assert.Nil(t, result["openai_api_key"])
			},
		},
		{
			name: "empty config",
			setup: func() {
				viper.Reset()
			},
			validate: func(t *testing.T, result map[string]*common.ConfigValue) {
				assert.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setup != nil {
				tt.setup()
			}

			// Test ToProto
			result := ToProto()

			// Validate
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestApplyProto tests the ApplyProto function
func TestApplyProto(t *testing.T) {
	tests := []struct {
		name      string
		configMap map[string]*common.ConfigValue
		validate  func(t *testing.T)
	}{
		{
			name: "apply all config values",
			configMap: func() map[string]*common.ConfigValue {
				configMap := make(map[string]*common.ConfigValue)

				configMap["db_path"] = &common.ConfigValue{}
				configMap["db_path"].SetStringValue("/applied/db")

				configMap["db_backend"] = &common.ConfigValue{}
				configMap["db_backend"].SetStringValue("sqlite")

				configMap["sqlite3_filename"] = &common.ConfigValue{}
				configMap["sqlite3_filename"].SetStringValue("applied.db")

				configMap["log_file"] = &common.ConfigValue{}
				configMap["log_file"].SetStringValue("/applied/logs/app.log")

				configMap["google_api_key"] = &common.ConfigValue{}
				configMap["google_api_key"].SetStringValue("applied-google-key")

				configMap["openai_api_key"] = &common.ConfigValue{}
				configMap["openai_api_key"].SetStringValue("applied-openai-key")

				return configMap
			}(),
			validate: func(t *testing.T) {
				assert.Equal(t, "/applied/db", viper.GetString("db_path"))
				assert.Equal(t, "sqlite", viper.GetString("db_backend"))
				assert.Equal(t, "applied.db", viper.GetString("sqlite3_filename"))
				assert.Equal(t, "/applied/logs/app.log", viper.GetString("log_file"))
				assert.Equal(t, "applied-google-key", viper.GetString("google_api_key"))
				assert.Equal(t, "applied-openai-key", viper.GetString("openai_api_key"))
			},
		},
		{
			name: "apply partial config values",
			configMap: func() map[string]*common.ConfigValue {
				configMap := make(map[string]*common.ConfigValue)

				configMap["db_path"] = &common.ConfigValue{}
				configMap["db_path"].SetStringValue("/partial/db")

				configMap["google_api_key"] = &common.ConfigValue{}
				configMap["google_api_key"].SetStringValue("partial-key")

				return configMap
			}(),
			validate: func(t *testing.T) {
				assert.Equal(t, "/partial/db", viper.GetString("db_path"))
				assert.Equal(t, "partial-key", viper.GetString("google_api_key"))

				// These should remain unchanged (empty in this test)
				assert.Empty(t, viper.GetString("db_backend"))
				assert.Empty(t, viper.GetString("openai_api_key"))
			},
		},
		{
			name:      "apply nil config map",
			configMap: nil,
			validate: func(t *testing.T) {
				// Should not panic and not change anything
				assert.Empty(t, viper.GetString("db_path"))
			},
		},
		{
			name: "apply config with empty values",
			configMap: func() map[string]*common.ConfigValue {
				configMap := make(map[string]*common.ConfigValue)

				configMap["db_path"] = &common.ConfigValue{}
				configMap["db_path"].SetStringValue("")

				configMap["google_api_key"] = &common.ConfigValue{}
				configMap["google_api_key"].SetStringValue("non-empty-key")

				return configMap
			}(),
			validate: func(t *testing.T) {
				// Empty values should not be set
				assert.Empty(t, viper.GetString("db_path"))
				// Non-empty values should be set
				assert.Equal(t, "non-empty-key", viper.GetString("google_api_key"))
			},
		},
		{
			name: "apply config with missing value field",
			configMap: func() map[string]*common.ConfigValue {
				configMap := make(map[string]*common.ConfigValue)

				// Create empty ConfigValue (no SetStringValue called)
				configMap["db_path"] = &common.ConfigValue{}

				configMap["google_api_key"] = &common.ConfigValue{}
				configMap["google_api_key"].SetStringValue("valid-key")

				return configMap
			}(),
			validate: func(t *testing.T) {
				// Missing/empty values should not be set
				assert.Empty(t, viper.GetString("db_path"))
				// Valid values should be set
				assert.Equal(t, "valid-key", viper.GetString("google_api_key"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper state
			viper.Reset()

			// Test ApplyProto
			ApplyProto(tt.configMap)

			// Validate
			if tt.validate != nil {
				tt.validate(t)
			}
		})
	}
}

// TestRoundTripConversion tests that ToProto and ApplyProto work together correctly
func TestRoundTripConversion(t *testing.T) {
	// Set up initial configuration
	viper.Reset()
	viper.Set("db_path", "/original/db")
	viper.Set("db_backend", "pebble")
	viper.Set("sqlite3_filename", "original.db")
	viper.Set("log_file", "/original/logs/app.log")
	viper.Set("google_api_key", "original-google-key")
	viper.Set("openai_api_key", "original-openai-key")

	// Convert to proto
	protoConfig := ToProto()

	// Reset viper and apply the proto config back
	viper.Reset()
	ApplyProto(protoConfig)

	// Verify all values are preserved
	assert.Equal(t, "/original/db", viper.GetString("db_path"))
	assert.Equal(t, "pebble", viper.GetString("db_backend"))
	assert.Equal(t, "original.db", viper.GetString("sqlite3_filename"))
	assert.Equal(t, "/original/logs/app.log", viper.GetString("log_file"))
	assert.Equal(t, "original-google-key", viper.GetString("google_api_key"))
	assert.Equal(t, "original-openai-key", viper.GetString("openai_api_key"))
}

// TestEnvVarMapping tests that environment variable mapping works correctly
func TestEnvVarMapping(t *testing.T) {
	// Reset viper completely before this test
	viper.Reset()

	defer func() {
		// Clean up environment variables
		os.Unsetenv("SM_DB_PATH")
		os.Unsetenv("SM_DB_BACKEND")
		os.Unsetenv("SM_SQLITE3_FILENAME")
		os.Unsetenv("SM_LOG_FILE")
		os.Unsetenv("SM_GOOGLE_API_KEY")
		os.Unsetenv("SM_OPENAI_API_KEY")
		// Reset viper after test
		viper.Reset()
	}()

	// Set environment variables
	os.Setenv("SM_DB_PATH", "/env/db")
	os.Setenv("SM_DB_BACKEND", "sqlite")
	os.Setenv("SM_SQLITE3_FILENAME", "env.db")
	os.Setenv("SM_LOG_FILE", "/env/logs/app.log")
	os.Setenv("SM_GOOGLE_API_KEY", "env-google-key")
	os.Setenv("SM_OPENAI_API_KEY", "env-openai-key")

	// Create command and load config
	cmd := &cobra.Command{}
	err := Load(cmd, "")
	require.NoError(t, err)

	// Verify environment variables are loaded
	assert.Equal(t, "/env/db", viper.GetString("db_path"))
	assert.Equal(t, "sqlite", viper.GetString("db_backend"))
	assert.Equal(t, "env.db", viper.GetString("sqlite3_filename"))
	assert.Equal(t, "/env/logs/app.log", viper.GetString("log_file"))
	assert.Equal(t, "env-google-key", viper.GetString("google_api_key"))
	assert.Equal(t, "env-openai-key", viper.GetString("openai_api_key"))
}

// TestDefaultPaths tests that default paths are set correctly
func TestDefaultPaths(t *testing.T) {
	viper.Reset()
	cmd := &cobra.Command{}

	err := Load(cmd, "")
	require.NoError(t, err)

	// Check that default paths are set
	dbPath := viper.GetString("db_path")
	assert.NotEmpty(t, dbPath)
	assert.Contains(t, dbPath, ".subtitle-manager")

	assert.Equal(t, "pebble", viper.GetString("db_backend"))
	assert.Equal(t, "subtitle-manager.db", viper.GetString("sqlite3_filename"))
	assert.Equal(t, "/config/logs/subtitle-manager.log", viper.GetString("log_file"))
}
