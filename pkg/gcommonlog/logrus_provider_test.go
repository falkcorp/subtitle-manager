// file: pkg/gcommonlog/logrus_provider_test.go
// version: 2.0.0
// guid: 3456b7c8-9d0e-4f12-8b34-5a6c7d8e9f01
package gcommonlog

import (
	"bytes"
	"strings"
	"testing"

	common "github.com/falkcorp/gcommon/v2/pkg/commonpb/v2"
)

// TestLogrusProvider verifies that the provider writes log messages and fields.
func TestLogrusProvider(t *testing.T) {
	buf := &bytes.Buffer{}
	p := NewLogrusProviderWithLevel(common.LogLevel_LOG_LEVEL_DEBUG)
	p.SetOutput(buf)
	fields := map[string]interface{}{"k": "v"}
	p.Info("hello", fields)

	out := buf.String()
	if !strings.Contains(out, "hello") || !strings.Contains(out, "k=v") {
		t.Errorf("unexpected output: %s", out)
	}
}

// TestWithField ensures With adds structured fields.
func TestWithField(t *testing.T) {
	buf := &bytes.Buffer{}
	p := NewLogrusProviderWithLevel(common.LogLevel_LOG_LEVEL_INFO)
	p.SetOutput(buf)

	fields := map[string]interface{}{"module": "test"}
	logger := p.With(fields)
	logger.Info("msg", nil)

	out := buf.String()
	if !strings.Contains(out, "module=test") {
		t.Errorf("expected field not found: %s", out)
	}
}
