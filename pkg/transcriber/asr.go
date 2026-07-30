// file: pkg/transcriber/asr.go
// version: 1.1.0
// guid: c35500d2-dcef-4ad9-884d-9d91de10459a
// last-edited: 2026-07-30

package transcriber

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
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

const (
	// defaultASRTotalTimeout bounds a whole transcription. Transcribing a full
	// media file legitimately takes many minutes, so it is generous.
	defaultASRTotalTimeout = 30 * time.Minute
	// defaultASRConnectTimeout bounds establishing the connection only.
	defaultASRConnectTimeout = 10 * time.Second
)

// asrConnectTimeout returns how long to wait for the connection to the Whisper
// server to be established.
func asrConnectTimeout() time.Duration {
	if secs := viper.GetInt("whisper.connect_timeout"); secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return defaultASRConnectTimeout
}

// asrTotalTimeout returns the ceiling on a whole transcription request.
func asrTotalTimeout() time.Duration {
	if secs := viper.GetInt("whisper.transcribe_timeout"); secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return defaultASRTotalTimeout
}

// newASRClient builds the HTTP client for ASR calls, separating the time
// allowed to *reach* the Whisper server from the time allowed to transcribe.
//
// # Why these are two different numbers
//
// A single timeout cannot express this. Transcription takes minutes, so the
// overall budget has to be large — but that same large number was also what
// bounded connecting. With the Whisper server down, misconfigured, or behind a
// black-holing firewall, every transcription attempt hung for the full budget
// (30 minutes by default) before failing. A library scan that hit that path
// once appeared to stall completely.
//
// Splitting them means an unreachable server fails in seconds while a running
// transcription still gets its full budget.
//
// # Why the transport is cloned rather than constructed
//
// pkg/proxy implements the proxy_url setting by setting Proxy on
// http.DefaultTransport. Building a fresh http.Transport here would silently
// opt Whisper traffic out of the operator's configured proxy — the sort of gap
// that only shows up as "why does everything except transcription go through
// the proxy?". Cloning inherits Proxy and the rest of the standard tuning; only
// the dial deadlines are overridden.
func newASRClient() *http.Client {
	connect := asrConnectTimeout()

	tr, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		// Something has replaced the default transport with a non-standard
		// type. Honour the connect timeout rather than silently ignoring it,
		// and accept that whatever that transport added is not inherited.
		return &http.Client{
			Timeout: asrTotalTimeout(),
			Transport: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				DialContext:         (&net.Dialer{Timeout: connect}).DialContext,
				TLSHandshakeTimeout: connect,
			},
		}
	}

	clone := tr.Clone()
	clone.DialContext = (&net.Dialer{
		Timeout:   connect,
		KeepAlive: 30 * time.Second,
	}).DialContext
	clone.TLSHandshakeTimeout = connect
	// Deliberately NOT setting ResponseHeaderTimeout: the whisper-asr-webservice
	// sends response headers only once transcription has finished, so any value
	// short enough to be useful as a liveness check would abort valid work.
	// The overall Timeout is what bounds that phase.
	return &http.Client{Timeout: asrTotalTimeout(), Transport: clone}
}

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
		c := newASRClient()
		httpClient = c
		// One request per client, so drop the idle connection rather than
		// leaving a transport behind on every transcription.
		if tr, ok := c.Transport.(*http.Transport); ok {
			defer tr.CloseIdleConnections()
		}
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
