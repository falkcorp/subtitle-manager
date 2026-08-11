// file: pkg/webserver/stacksubs_test.go
// version: 1.0.0
// guid: 2f7a6c05-4b91-4de8-83c2-90e5b71fa634
// last-edited: 2026-08-11

package webserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const enSRT = `1
00:00:01,000 --> 00:00:04,000
My name is Walter Hartwell White.

2
00:00:05,500 --> 00:00:09,000
I live at 308 Negra Arroyo Lane.
`

const esSRT = `1
00:00:01,000 --> 00:00:04,000
Me llamo Walter Hartwell White.

2
00:00:05,500 --> 00:00:09,000
Vivo en 308 Negra Arroyo Lane.
`

// writeSubs drops an English and Spanish sidecar into a temp dir and returns
// their paths.
func writeSubs(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	en := filepath.Join(dir, "Episode.en.srt")
	es := filepath.Join(dir, "Episode.es.srt")
	if err := os.WriteFile(en, []byte(enSRT), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(es, []byte(esSRT), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, en, es
}

func postStack(t *testing.T, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/subtitles/stack", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	stackSubtitlesHandler().ServeHTTP(rec, req)
	return rec
}

// The UI needs to build a bilingual file from two subtitles that already exist
// in the library, so this endpoint takes paths rather than uploads — unlike
// /api/dualsub, which uploads a single file and machine-translates it.
func TestStackSubtitlesHandlerWritesBilingualFile(t *testing.T) {
	dir, en, es := writeSubs(t)
	out := filepath.Join(dir, "Episode.eo.srt")

	rec := postStack(t, map[string]string{"primary": en, "secondary": es, "output": out})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output not written: %v", err)
	}
	got := string(data)
	// One cue must carry both languages — that is the whole point. If the two
	// tracks were interleaved instead, these would sit in separate cues.
	if !strings.Contains(got, "My name is Walter Hartwell White.\nMe llamo Walter Hartwell White.") {
		t.Errorf("languages are not stacked within one cue:\n%s", got)
	}
	if n := strings.Count(got, "-->"); n != 2 {
		t.Errorf("expected 2 cues, got %d:\n%s", n, got)
	}
}

// Without an explicit output the endpoint must choose the sentinel-language
// name, so media servers see a distinct track rather than an overwritten one.
func TestStackSubtitlesHandlerDefaultsToSentinelOutput(t *testing.T) {
	dir, en, es := writeSubs(t)

	rec := postStack(t, map[string]string{"primary": en, "secondary": es})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Output string `json:"output"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	want := filepath.Join(dir, "Episode.eo.srt")
	if resp.Output != want {
		t.Errorf("output = %q, want %q", resp.Output, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("default output not written: %v", err)
	}
}

// A generated bilingual file used to be reported as English, because `eo` was
// absent from the language table and extractLanguageFromFilename defaults to
// English for anything it does not recognise. In the Media Library that showed
// two identical "English" rows for one episode, so the user could not tell the
// bilingual track from the real English one — and anything keyed on language
// saw two English subtitles.
func TestSentinelLanguageIsNotReportedAsEnglish(t *testing.T) {
	got := extractLanguageFromFilename("Breaking Bad - S01E01 - Pilot.eo.srt")
	if got == "English" {
		t.Fatalf("sentinel .eo subtitle reported as English")
	}
	if !strings.Contains(strings.ToLower(got), "bilingual") {
		t.Errorf("language = %q, want it to identify the file as bilingual", got)
	}
}

func TestStackSubtitlesHandlerRejectsBadRequests(t *testing.T) {
	_, en, _ := writeSubs(t)

	t.Run("GET is not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/subtitles/stack", nil)
		rec := httptest.NewRecorder()
		stackSubtitlesHandler().ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", rec.Code)
		}
	})

	t.Run("missing secondary", func(t *testing.T) {
		if rec := postStack(t, map[string]string{"primary": en}); rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	// Path traversal must be refused before any file is read, the same as every
	// other path-taking endpoint here.
	t.Run("traversal is refused", func(t *testing.T) {
		rec := postStack(t, map[string]string{
			"primary":   "../../../../etc/passwd",
			"secondary": en,
		})
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}
