// file: pkg/webhooks/plex_test.go
// version: 1.0.0
// guid: f2a9c7d1-5b83-4e60-9a14-7c0e2d8b6035

package webhooks

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

const plexLibraryNew = `{"event":"library.new","Metadata":{"librarySectionType":"movie","title":"Inception","Media":[{"Part":[{"file":"/media/Inception.mkv"}]}]}}`

func newPlexMultipart(t *testing.T, payload string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	if err := mw.WriteField("payload", payload); err != nil {
		t.Fatalf("write field: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/plex", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestPlexRawPayloadMultipart(t *testing.T) {
	req := newPlexMultipart(t, plexLibraryNew)
	if got := plexRawPayload(req); got != plexLibraryNew {
		t.Fatalf("expected extracted payload, got %q", got)
	}
}

func TestPlexPayloadFirstFile(t *testing.T) {
	req := newPlexMultipart(t, plexLibraryNew)
	raw := plexRawPayload(req)
	var p plexPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Event != "library.new" {
		t.Fatalf("event = %q", p.Event)
	}
	if got := p.firstFile(); got != "/media/Inception.mkv" {
		t.Fatalf("firstFile = %q", got)
	}
}

func TestPlexHandlerIgnoresNonLibraryNew(t *testing.T) {
	req := newPlexMultipart(t, `{"event":"media.play","Metadata":{"Media":[{"Part":[{"file":"/media/x.mkv"}]}]}}`)
	w := httptest.NewRecorder()
	PlexHandler().ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for ignored event, got %d", w.Code)
	}
}

func TestPlexHandlerBadBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/plex", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	PlexHandler().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad body, got %d", w.Code)
	}
}

func TestPlexHandlerMethod(t *testing.T) {
	w := httptest.NewRecorder()
	PlexHandler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET, got %d", w.Code)
	}
}
