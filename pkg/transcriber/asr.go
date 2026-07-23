// file: pkg/transcriber/asr.go
// version: 1.0.0
// guid: c35500d2-dcef-4ad9-884d-9d91de10459a

package transcriber

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ASROptions configures a call to a self-hosted whisper-asr-webservice /asr
// endpoint (the onerahmet/openai-whisper-asr-webservice image).
type ASROptions struct {
	// Task is "transcribe" (default) or "translate" (translate-to-English).
	Task string
	// Language is the spoken-language ISO code; empty means auto-detect.
	Language string
	// Output is the subtitle format: "srt" (default), "vtt", "txt", "tsv", "json".
	Output string
	// WordTimestamps requests per-word timings (json output only).
	WordTimestamps bool
}

// asrHTTPClient is the default client for ASR calls. Transcription of a full
// media file can take many minutes, so the timeout is generous.
var asrHTTPClient = &http.Client{Timeout: 30 * time.Minute}

// ASRTranscribe posts filePath to a self-hosted whisper-asr-webservice instance
// at baseURL (for example http://localhost:9000) and returns the subtitle bytes
// in the requested output format.
//
// This speaks the webservice's NATIVE protocol — a multipart POST to /asr with
// the media in the `audio_file` field — not the OpenAI /audio/transcriptions
// API. It is the client the self-hosted ("start our own") Whisper path needs;
// the previous container code incorrectly pointed the OpenAI SDK at the
// container with a dummy key. If httpClient is nil, a long-timeout default is
// used.
func ASRTranscribe(ctx context.Context, httpClient *http.Client, baseURL, filePath string, opts ASROptions) ([]byte, error) {
	if opts.Task == "" {
		opts.Task = "transcribe"
	}
	if opts.Output == "" {
		opts.Output = "srt"
	}
	if httpClient == nil {
		httpClient = asrHTTPClient
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open media: %w", err)
	}
	defer f.Close()

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("audio_file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("build multipart: %w", err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		return nil, fmt.Errorf("copy media: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/asr")
	if err != nil {
		return nil, fmt.Errorf("invalid whisper base URL: %w", err)
	}
	q := u.Query()
	q.Set("task", opts.Task)
	q.Set("output", opts.Output)
	q.Set("encode", "true")
	if opts.Language != "" {
		q.Set("language", opts.Language)
	}
	if opts.WordTimestamps {
		q.Set("word_timestamps", "true")
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("asr request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read asr response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("whisper asr webservice returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("whisper asr webservice returned an empty subtitle")
	}
	return data, nil
}
