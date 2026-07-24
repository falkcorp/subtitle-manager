// file: pkg/config/config_test.go
// version: 1.0.1
// guid: b1c2d3e4-f5g6-h7i8-j9k0-l1m2n3o4p5q6

package config

import (
	"testing"

	common "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/commonpb/v2"
)

func TestLogLevelMigration(t *testing.T) {
	// Test that our LogLevel alias works with gcommon types
	var level LogLevel = common.LogLevel_LOG_LEVEL_INFO

	if level != LogLevel_INFO {
		t.Errorf("Expected LogLevel_INFO, got %v", level)
	}

	// Test parsing
	parsed, err := ParseLogLevel("DEBUG")
	if err != nil {
		t.Errorf("Failed to parse log level: %v", err)
	}

	if parsed != LogLevel_DEBUG {
		t.Errorf("Expected LogLevel_DEBUG, got %v", parsed)
	}

	// Test string conversion
	str := LogLevelString(LogLevel_WARN)
	if str != "WARN" {
		t.Errorf("Expected 'WARN', got %s", str)
	}

	// Test that gcommon LogLevel values work
	if common.LogLevel_LOG_LEVEL_ERROR != LogLevel_ERROR {
		t.Error("gcommon LogLevel constants don't match our aliases")
	}
}

func TestLogLevelCompatibility(t *testing.T) {
	// Test that all our constants match gcommon constants
	tests := []struct {
		local   LogLevel
		gcommon common.LogLevel
		name    string
	}{
		{LogLevel_UNSPECIFIED, common.LogLevel_LOG_LEVEL_UNSPECIFIED, "UNSPECIFIED"},
		{LogLevel_TRACE, common.LogLevel_LOG_LEVEL_TRACE, "TRACE"},
		{LogLevel_DEBUG, common.LogLevel_LOG_LEVEL_DEBUG, "DEBUG"},
		{LogLevel_INFO, common.LogLevel_LOG_LEVEL_INFO, "INFO"},
		{LogLevel_WARN, common.LogLevel_LOG_LEVEL_WARN, "WARN"},
		{LogLevel_ERROR, common.LogLevel_LOG_LEVEL_ERROR, "ERROR"},
		{LogLevel_FATAL, common.LogLevel_LOG_LEVEL_FATAL, "FATAL"},
	}

	for _, test := range tests {
		if test.local != test.gcommon {
			t.Errorf("LogLevel mismatch for %s: local=%d, gcommon=%d", test.name, test.local, test.gcommon)
		}
	}
}
