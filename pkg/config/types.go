// file: pkg/config/types.go
// version: 1.1.1
// guid: a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6

package config

import (
	"fmt"
	"strings"

	common "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/commonpb/v2"
)

// LogLevel is now an alias for gcommon LogLevel to maintain compatibility
// All new code should use common.LogLevel directly
type LogLevel = common.LogLevel

const (
	LogLevel_UNSPECIFIED = common.LogLevel_LOG_LEVEL_UNSPECIFIED
	LogLevel_TRACE       = common.LogLevel_LOG_LEVEL_TRACE
	LogLevel_DEBUG       = common.LogLevel_LOG_LEVEL_DEBUG
	LogLevel_INFO        = common.LogLevel_LOG_LEVEL_INFO
	LogLevel_WARN        = common.LogLevel_LOG_LEVEL_WARN
	LogLevel_ERROR       = common.LogLevel_LOG_LEVEL_ERROR
	LogLevel_FATAL       = common.LogLevel_LOG_LEVEL_FATAL
)

var logLevelNames = map[LogLevel]string{
	LogLevel_UNSPECIFIED: "UNSPECIFIED",
	LogLevel_TRACE:       "TRACE",
	LogLevel_DEBUG:       "DEBUG",
	LogLevel_INFO:        "INFO",
	LogLevel_WARN:        "WARN",
	LogLevel_ERROR:       "ERROR",
	LogLevel_FATAL:       "FATAL",
}

var logLevelValues = map[string]LogLevel{
	"UNSPECIFIED": LogLevel_UNSPECIFIED,
	"TRACE":       LogLevel_TRACE,
	"DEBUG":       LogLevel_DEBUG,
	"INFO":        LogLevel_INFO,
	"WARN":        LogLevel_WARN,
	"ERROR":       LogLevel_ERROR,
	"FATAL":       LogLevel_FATAL,
}

// LogLevelString returns the string representation of the log level
func LogLevelString(l LogLevel) string {
	if name, ok := logLevelNames[l]; ok {
		return name
	}
	return fmt.Sprintf("LogLevel(%d)", l)
}

// ParseLogLevel parses a string into a LogLevel
func ParseLogLevel(s string) (LogLevel, error) {
	if level, ok := logLevelValues[strings.ToUpper(s)]; ok {
		return level, nil
	}
	return LogLevel_UNSPECIFIED, fmt.Errorf("unknown log level: %s", s)
}

// AppenderType represents the type of log appender
// TODO: Replace with gcommon protobuf AppenderType when available
type AppenderType int32

const (
	AppenderType_UNSPECIFIED AppenderType = 0
	AppenderType_CONSOLE     AppenderType = 1
	AppenderType_FILE        AppenderType = 2
	AppenderType_SYSLOG      AppenderType = 3
	AppenderType_NETWORK     AppenderType = 4
)

var appenderTypeNames = map[AppenderType]string{
	AppenderType_UNSPECIFIED: "UNSPECIFIED",
	AppenderType_CONSOLE:     "CONSOLE",
	AppenderType_FILE:        "FILE",
	AppenderType_SYSLOG:      "SYSLOG",
	AppenderType_NETWORK:     "NETWORK",
}

var appenderTypeValues = map[string]AppenderType{
	"UNSPECIFIED": AppenderType_UNSPECIFIED,
	"CONSOLE":     AppenderType_CONSOLE,
	"FILE":        AppenderType_FILE,
	"SYSLOG":      AppenderType_SYSLOG,
	"NETWORK":     AppenderType_NETWORK,
}

// String returns the string representation of the appender type
func (a AppenderType) String() string {
	if name, ok := appenderTypeNames[a]; ok {
		return name
	}
	return fmt.Sprintf("AppenderType(%d)", a)
}

// ParseAppenderType parses a string into an AppenderType
func ParseAppenderType(s string) (AppenderType, error) {
	if appender, ok := appenderTypeValues[strings.ToUpper(s)]; ok {
		return appender, nil
	}
	return AppenderType_UNSPECIFIED, fmt.Errorf("unknown appender type: %s", s)
}

// SubtitleManagerConfig represents the main configuration for the subtitle manager
// This replaces the protobuf-based configuration temporarily
// TODO: Replace with protobuf-based config when gcommon integration is complete
type SubtitleManagerConfig struct {
	DbPath          string `json:"db_path" yaml:"db_path"`
	DbBackend       string `json:"db_backend" yaml:"db_backend"`
	Sqlite3Filename string `json:"sqlite3_filename" yaml:"sqlite3_filename"`

	// Legacy string-based log file configuration (deprecated)
	LogFile string `json:"log_file,omitempty" yaml:"log_file,omitempty"`

	GoogleApiKey string `json:"google_api_key" yaml:"google_api_key"`
	OpenaiApiKey string `json:"openai_api_key" yaml:"openai_api_key"`

	// Enhanced logging configuration
	Logging LoggingConfig `json:"logging" yaml:"logging"`
}

// LoggingConfig defines comprehensive logging settings
type LoggingConfig struct {
	// Global log level
	GlobalLevel LogLevel `json:"global_level" yaml:"global_level"`

	// Primary appender type (console, file, etc.)
	PrimaryAppender AppenderType `json:"primary_appender" yaml:"primary_appender"`

	// Log file path (when using file appenders)
	LogFilePath string `json:"log_file_path" yaml:"log_file_path"`

	// Component-specific log levels
	ComponentLevels map[string]LogLevel `json:"component_levels" yaml:"component_levels"`

	// Enable structured JSON logging
	StructuredLogging bool `json:"structured_logging" yaml:"structured_logging"`

	// Log rotation settings
	Rotation LogRotationConfig `json:"rotation" yaml:"rotation"`
}

// LogRotationConfig defines log rotation settings
type LogRotationConfig struct {
	// Maximum size in megabytes before rotation
	MaxSizeMB int `json:"max_size_mb" yaml:"max_size_mb"`

	// Maximum number of backup files to keep
	MaxBackups int `json:"max_backups" yaml:"max_backups"`

	// Maximum age in days before deletion
	MaxAgeDays int `json:"max_age_days" yaml:"max_age_days"`

	// Whether to compress rotated files
	Compress bool `json:"compress" yaml:"compress"`
}
