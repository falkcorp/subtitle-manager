// file: pkg/transcriber/asr_test.go
// version: 1.0.0
// guid: 9a7483a9-a895-401e-951d-2f0abea7746c

package transcriber

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const sampleSRT = "1\n00:00:01,000 --> 00:00:02,000\nHello\n"

// TestASRTranscribe verifies the client speaks the whisper-asr-webservice
// native /asr protocol: multipart audio_file + query params, SRT body back.
func TestASRTranscribe(t *testing.T) {
	dir := t.TempDir()
	media := filepath.Join(dir, "clip.wav")
	if err := os.WriteFile(media, []byte("AUDIODATA"), 0644); err != nil {
		t.Fatalf("write media: %v", err)
	}

	var sawFile bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/asr" {
			http.Error(w, "bad route", http.StatusNotFound)
			return
		}
		q := r.URL.Query()
		if q.Get("task") != "transcribe" || q.Get("output") != "srt" || q.Get("language") != "en" {
			http.Error(w, "bad query: "+r.URL.RawQuery, http.StatusBadRequest)
			return
		}
		f, hdr, err := r.FormFile("audio_file")
		if err != nil {
			http.Error(w, "no audio_file: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer f.Close()
		if hdr.Filename != "clip.wav" {
			http.Error(w, "bad filename", http.StatusBadRequest)
			return
		}
		if b, _ := io.ReadAll(f); string(b) != "AUDIODATA" {
			http.Error(w, "bad audio content", http.StatusBadRequest)
			return
		}
		sawFile = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleSRT))
	}))
	defer srv.Close()

	out, err := ASRTranscribe(context.Background(), srv.Client(), srv.URL, media, ASROptions{Language: "en"})
	if err != nil {
		t.Fatalf("asr: %v", err)
	}
	if !sawFile {
		t.Fatal("server did not receive audio_file")
	}
	if string(out) != sampleSRT {
		t.Fatalf("unexpected subtitle: %q", out)
	}
}

// TestASRTranscribeErrorStatus verifies a non-200 is surfaced as an error.
func TestASRTranscribeErrorStatus(t *testing.T) {
	dir := t.TempDir()
	media := filepath.Join(dir, "clip.wav")
	_ = os.WriteFile(media, []byte("x"), 0644)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not loaded", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	if _, err := ASRTranscribe(context.Background(), srv.Client(), srv.URL, media, ASROptions{}); err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

// TestASRTranscribeMissingFile verifies a clear error when the media is absent.
func TestASRTranscribeMissingFile(t *testing.T) {
	if _, err := ASRTranscribe(context.Background(), nil, "http://127.0.0.1:0", "/no/such/file.wav", ASROptions{}); err == nil {
		t.Fatal("expected error for missing media file")
	}
}
