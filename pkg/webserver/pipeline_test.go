// file: pkg/webserver/pipeline_test.go
// version: 1.0.0
// guid: b3718542-ed3a-4f7b-af10-5e1cf63c3000

package webserver

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/translator"
)

// TestDualSubHandler verifies the endpoint returns a bilingual SRT keeping the
// original text and appending the translation.
func TestDualSubHandler(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		qs := r.URL.Query()["q"]
		if len(qs) == 0 {
			qs = []string{""}
		}
		parts := make([]string, len(qs))
		for i := range qs {
			parts[i] = `{"translatedText":"你好"}`
		}
		fmt.Fprintf(w, `{"data":{"translations":[%s]}}`, strings.Join(parts, ","))
	}))
	defer srv.Close()
	translator.SetGoogleAPIURL(srv.URL)
	defer translator.SetGoogleAPIURL("https://translation.googleapis.com/language/translate/v2")
	viper.Set("google_api_key", "k")
	defer viper.Reset()

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, _ := mw.CreateFormFile("file", "in.srt")
	fmt.Fprint(fw, "1\n00:00:01,000 --> 00:00:02,000\nHello world\n")
	_ = mw.WriteField("lang", "zh")
	_ = mw.WriteField("service", "google")
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/dualsub", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	dualSubHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	out := rec.Body.String()
	if !strings.Contains(out, "Hello world") || !strings.Contains(out, "你好") {
		t.Fatalf("expected original + translation, got:\n%s", out)
	}
}

// TestPipelineHandlersMethod verifies non-POST is rejected.
func TestPipelineHandlersMethod(t *testing.T) {
	for name, h := range map[string]http.Handler{
		"library-search": librarySearchHandler(),
		"dualsub":        dualSubHandler(),
		"verify":         verifyHandler(),
	} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s: expected 405 for GET, got %d", name, rec.Code)
		}
	}
}

// TestLibrarySearchBadRequest verifies missing/invalid lang is rejected.
func TestLibrarySearchBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/library/search", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	librarySearchHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing lang, got %d", rec.Code)
	}
}

// TestVerifyBadRequest verifies missing fields are rejected before any heavy work.
func TestVerifyBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/verify", strings.NewReader(`{"media":"x"}`))
	rec := httptest.NewRecorder()
	verifyHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for incomplete request, got %d", rec.Code)
	}
}

// TestLibrarySearchStatus verifies the status endpoint returns JSON.
func TestLibrarySearchStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/library/search/status", nil)
	rec := httptest.NewRecorder()
	librarySearchStatusHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected json content-type, got %q", ct)
	}
}
