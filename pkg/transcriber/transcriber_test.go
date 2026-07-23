package transcriber

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// TestWhisperTranscribe verifies that the request is made to the correct
// endpoint and the subtitle text is returned.
func TestWhisperTranscribe(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		fmt.Fprint(w, "1\n00:00:00,000 --> 00:00:01,000\ntext\n")
	}))
	defer srv.Close()

	SetBaseURL(srv.URL + "/v1")
	defer SetBaseURL("https://api.openai.com/v1")
	SetWhisperModel("whisper-1")

	dir := t.TempDir()
	file := filepath.Join(dir, "a.wav")
	if err := os.WriteFile(file, []byte("data"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	b, err := WhisperTranscribe(file, "en", "k")
	if err != nil {
		t.Fatalf("transcribe: %v", err)
	}
	if gotPath != "/v1/audio/transcriptions" {
		t.Fatalf("unexpected path %s", gotPath)
	}
	if len(b) == 0 {
		t.Fatalf("empty result")
	}
}

// TestWhisperTranscribeRoutesToASR verifies that when whisper.transcribe_url is
// configured, WhisperTranscribe talks to the self-hosted /asr webservice (no API
// key required) instead of the OpenAI-compatible endpoint.
func TestWhisperTranscribeRoutesToASR(t *testing.T) {
	var gotPath, gotLang string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotLang = r.URL.Query().Get("language")
		fmt.Fprint(w, "1\n00:00:00,000 --> 00:00:01,000\nhola\n")
	}))
	defer srv.Close()

	viper.Reset()
	viper.Set("whisper.transcribe_url", srv.URL)
	defer viper.Reset()

	dir := t.TempDir()
	file := filepath.Join(dir, "a.wav")
	if err := os.WriteFile(file, []byte("data"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Empty API key on purpose: the ASR path must not require one.
	b, err := WhisperTranscribe(file, "es", "")
	if err != nil {
		t.Fatalf("transcribe: %v", err)
	}
	if gotPath != "/asr" {
		t.Fatalf("expected /asr, got %s", gotPath)
	}
	if gotLang != "es" {
		t.Fatalf("expected language es, got %q", gotLang)
	}
	if len(b) == 0 {
		t.Fatalf("empty result")
	}
}
