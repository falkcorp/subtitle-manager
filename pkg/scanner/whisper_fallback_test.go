// file: pkg/scanner/whisper_fallback_test.go
// version: 1.0.0
// guid: b9f2c650-1a83-4e07-8d24-6c5f0a9e3b71

package scanner

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// failingProvider always fails to fetch, forcing the fallback path.
type failingProvider struct{}

func (failingProvider) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	return nil, fmt.Errorf("no subtitle")
}

// TestWhisperFallbackWritesSubtitle verifies that when the provider fails and
// whisper fallback is enabled, ProcessFile transcribes via the configured ASR
// server and writes the resulting subtitle.
func TestWhisperFallbackWritesSubtitle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "1\n00:00:00,000 --> 00:00:01,000\ntranscribed\n")
	}))
	defer srv.Close()

	dir := t.TempDir()
	viper.Reset()
	viper.Set("media_directory", dir)
	viper.Set("whisper.fallback_enabled", true)
	viper.Set("whisper.transcribe_url", srv.URL)
	defer viper.Reset()

	vid := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(vid, []byte("media"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	if err := ProcessFile(context.Background(), vid, "en", "test", failingProvider{}, false, nil); err != nil {
		t.Fatalf("process: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "movie.en.srt"))
	if err != nil {
		t.Fatalf("expected fallback subtitle written: %v", err)
	}
	if string(data) == "" {
		t.Fatal("expected non-empty fallback subtitle")
	}
}

// TestWhisperFallbackDisabled verifies the provider error is returned when the
// fallback is off.
func TestWhisperFallbackDisabled(t *testing.T) {
	dir := t.TempDir()
	viper.Reset()
	viper.Set("media_directory", dir)
	defer viper.Reset()

	vid := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(vid, []byte("media"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	if err := ProcessFile(context.Background(), vid, "en", "test", failingProvider{}, false, nil); err == nil {
		t.Fatal("expected error when fallback disabled and provider fails")
	}
	if _, err := os.Stat(filepath.Join(dir, "movie.en.srt")); !os.IsNotExist(err) {
		t.Fatal("no subtitle should be written when fallback is disabled")
	}
}
