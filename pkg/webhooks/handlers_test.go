// file: pkg/webhooks/handlers_test.go
// version: 1.0.0
// guid: 2ad5bf1d-6db9-45a1-b132-6f9b82b529b0

package webhooks

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookHandlerValidateSignature(t *testing.T) {
	payload := []byte("payload")
	secret := "top-secret"
	validSignature := generateHMACSignature(payload, secret)

	cases := []struct {
		name      string
		validator func(signature string) bool
		signature string
		expected  bool
	}{
		{
			name: "sonarr valid signature",
			validator: func(signature string) bool {
				return NewSonarrHandler(secret).ValidateSignature(payload, signature)
			},
			signature: "sha256=" + validSignature,
			expected:  true,
		},
		{
			name: "radarr invalid signature",
			validator: func(signature string) bool {
				return NewRadarrHandler(secret).ValidateSignature(payload, signature)
			},
			signature: "deadbeef",
			expected:  false,
		},
		{
			name: "custom empty secret",
			validator: func(signature string) bool {
				return NewCustomHandler("").ValidateSignature(payload, signature)
			},
			signature: "anything",
			expected:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.validator(tc.signature))
		})
	}
}

func TestWebhookHandlerNames(t *testing.T) {
	assert.Equal(t, "sonarr", NewSonarrHandler("secret").GetName())
	assert.Equal(t, "radarr", NewRadarrHandler("secret").GetName())
	assert.Equal(t, "custom", NewCustomHandler("secret").GetName())
}

func TestSonarrHandlerHandleErrors(t *testing.T) {
	payload := map[string]any{
		"eventType": "Download",
		"episodeFile": map[string]any{
			"path": "",
		},
	}
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)

	handler := NewSonarrHandler("secret")

	cases := []struct {
		name    string
		payload []byte
		setup   func(t *testing.T)
		wantErr string
	}{
		{
			name:    "invalid payload",
			payload: []byte("{"),
			wantErr: "invalid payload format",
		},
		{
			name:    "non download event",
			payload: []byte(`{"eventType":"Test"}`),
			wantErr: "",
		},
		{
			name:    "missing file path",
			payload: encoded,
			wantErr: "missing file path",
		},
		{
			name:    "invalid file path",
			payload: []byte(`{"eventType":"Download","episodeFile":{"path":"/etc/passwd"}}`),
			setup: func(t *testing.T) {
				viper.Reset()
				t.Cleanup(viper.Reset)
				tempDir := t.TempDir()
				viper.Set("media_directory", tempDir)
			},
			wantErr: "invalid file path",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}
			err := handler.Handle(context.Background(), tc.payload, http.Header{})
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestRadarrHandlerHandleErrors(t *testing.T) {
	payload := map[string]any{
		"eventType": "Download",
		"movieFile": map[string]any{
			"path": "",
		},
	}
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)

	handler := NewRadarrHandler("secret")

	cases := []struct {
		name    string
		payload []byte
		setup   func(t *testing.T)
		wantErr string
	}{
		{
			name:    "invalid payload",
			payload: []byte("{"),
			wantErr: "invalid payload format",
		},
		{
			name:    "non download event",
			payload: []byte(`{"eventType":"Test"}`),
			wantErr: "",
		},
		{
			name:    "missing file path",
			payload: encoded,
			wantErr: "missing file path",
		},
		{
			name:    "invalid file path",
			payload: []byte(`{"eventType":"Download","movieFile":{"path":"/etc/passwd"}}`),
			setup: func(t *testing.T) {
				viper.Reset()
				t.Cleanup(viper.Reset)
				tempDir := t.TempDir()
				viper.Set("media_directory", tempDir)
			},
			wantErr: "invalid file path",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}
			err := handler.Handle(context.Background(), tc.payload, http.Header{})
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestCustomHandlerHandleErrors(t *testing.T) {
	validDir := t.TempDir()
	viper.Set("media_directory", validDir)
	t.Cleanup(viper.Reset)

	validPath := filepath.Join(validDir, "movie.mkv")
	if err := os.WriteFile(validPath, []byte("test"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cases := []struct {
		name    string
		payload []byte
		wantErr string
	}{
		{
			name:    "invalid payload",
			payload: []byte("{"),
			wantErr: "invalid payload format",
		},
		{
			name:    "missing fields",
			payload: []byte(`{"path":""}`),
			wantErr: "missing required fields",
		},
		{
			name:    "invalid path",
			payload: []byte(`{"path":"/etc/passwd","lang":"en"}`),
			wantErr: "invalid file path",
		},
		{
			name:    "invalid language",
			payload: []byte(`{"path":"` + validPath + `","lang":"en-us"}`),
			wantErr: "invalid language code",
		},
		{
			name:    "invalid provider name",
			payload: []byte(`{"path":"` + validPath + `","lang":"en","provider":"bad/provider"}`),
			wantErr: "invalid provider name",
		},
		{
			name:    "unknown provider",
			payload: []byte(`{"path":"` + validPath + `","lang":"en","provider":"unknown"}`),
			wantErr: "provider not available",
		},
	}

	handler := NewCustomHandler("secret")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := handler.Handle(context.Background(), tc.payload, http.Header{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// TestNoHardcodedLanguage guards against a hardcoded language literal
// returning to the webhook handlers.
//
// A behavioural test here would be vacuous: with no provider configured
// nothing is downloaded whatever language is requested, so an assertion on the
// handler's result passes with the bug present. The property that actually
// matters is that these call sites ask for the configured default rather than
// a literal, and that is visible in the source.
//
// pkg/profiles carries the real behavioural tests for how the default is
// resolved.
func TestNoHardcodedLanguage(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	// ProcessFile(ctx, path, <literal>, ...) — the shape that pinned every
	// import to English regardless of configuration.
	bad := regexp.MustCompile(`ProcessFile\([^)]*,\s*"[a-z]{2}(-[A-Za-z]{2,4})?"`)

	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if bad.MatchString(line) {
				t.Errorf("%s:%d passes a hardcoded language to ProcessFile; a "+
					"non-English library would silently get English subtitles. "+
					"Use profiles.DefaultLanguage():\n\t%s",
					f, i+1, strings.TrimSpace(line))
			}
		}
	}
}
