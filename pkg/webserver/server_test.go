//go:build !ci
// +build !ci

//nolint:errcheck // Test file - ignoring error checks for brevity
package webserver

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	auth "github.com/jdfalk/subtitle-manager/pkg/gcommonauth"
	"github.com/jdfalk/subtitle-manager/pkg/testutil"
	"github.com/jdfalk/subtitle-manager/pkg/translator"

	"github.com/spf13/viper"
)

// getTestDB returns a test database connection. Uses Pebble for pure Go builds.
func getTestDB(t *testing.T) (*database.PebbleStore, error) {
	return database.OpenPebble(t.TempDir())
}

// skipIfNoSQLite skips the test if SQLite support is not available (pure Go build).
func skipIfNoSQLite(t *testing.T) {
	if _, err := database.Open(":memory:"); err != nil {
		t.Skip("SQLite support not available - skipping test that requires SQLite")
	}
}

// TestHandler verifies that the handler serves index.html at root.
func TestHandler(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	testutil.MustNoError(t, "open db", err)
	defer db.Close()

	// create test user and api key
	testutil.MustNoError(t, "create user", auth.CreateUser(db, "test", "pass", "", "admin"))
	keyObj, err := auth.GenerateAPIKey(db, 1)
	testutil.MustNoError(t, "generate key", err)

	h, err := Handler(db)
	testutil.MustNoError(t, "create handler", err)
	srv := httptest.NewServer(h)
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/", nil)
	req.Header.Set("X-API-Key", keyObj.GetId())
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

// TestSPAIndexFallback verifies that unknown paths return index.html so client
// side routing works on refresh.
func TestSPAIndexFallback(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	testutil.MustNoError(t, "open db", err)
	defer db.Close()

	testutil.MustNoError(t, "create user", auth.CreateUser(db, "test", "pass", "", "admin"))
	keyObj, err := auth.GenerateAPIKey(db, 1)
	testutil.MustNoError(t, "generate key", err)

	h, err := Handler(db)
	testutil.MustNoError(t, "create handler", err)
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/settings", nil)
	req.Header.Set("X-API-Key", keyObj.GetId())
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

// TestBaseURL verifies that the handler respects the configured base URL.
func TestBaseURL(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	testutil.MustNoError(t, "open db", err)
	defer db.Close()

	testutil.MustNoError(t, "create user", auth.CreateUser(db, "test", "pass", "", "admin"))
	key := testutil.MustGet(t, "key", func() (string, error) {
		k, err := auth.GenerateAPIKey(db, 1)
		if err != nil {
			return "", err
		}
		return k.GetId(), nil
	})

	viper.Set("base_url", "sub")
	defer viper.Reset()

	h, err := Handler(db)
	testutil.MustNoError(t, "handler", err)
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/sub/", nil)
	req.Header.Set("X-API-Key", key)
	resp := testutil.MustGet(t, "request", func() (*http.Response, error) { return srv.Client().Do(req) })
	testutil.MustEqual(t, "status", http.StatusOK, resp.StatusCode)

	req2, _ := http.NewRequest("GET", srv.URL+"/", nil)
	req2.Header.Set("X-API-Key", key)
	resp2 := testutil.MustGet(t, "request2", func() (*http.Response, error) { return srv.Client().Do(req2) })
	if resp2.StatusCode == http.StatusOK {
		t.Fatalf("root path should not be served when base_url set")
	}
}

// TestRBAC verifies that permissions are enforced for protected routes.
func TestRBAC(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	testutil.MustNoError(t, "open db", err)
	defer db.Close()

	// viewer role should not access /api/config
	testutil.MustNoError(t, "create viewer", auth.CreateUser(db, "viewer", "p", "", "viewer"))
	vkeyObj, err := auth.GenerateAPIKey(db, 1)
	testutil.MustNoError(t, "viewer key", err)

	// admin role can access
	testutil.MustNoError(t, "create admin", auth.CreateUser(db, "admin", "p", "", "admin"))
	akeyObj, err := auth.GenerateAPIKey(db, 2)
	testutil.MustNoError(t, "admin key", err)

	h, err := Handler(db)
	testutil.MustNoError(t, "create handler", err)
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/config", nil)
	req.Header.Set("X-API-Key", vkeyObj.GetId())
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer status %d", resp.StatusCode)
	}

	req2, _ := http.NewRequest("GET", srv.URL+"/api/config", nil)
	req2.Header.Set("X-API-Key", akeyObj.GetId())
	resp2, err := srv.Client().Do(req2)
	if err != nil {
		t.Fatalf("admin get: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("admin status %d", resp2.StatusCode)
	}
}

// TestAuthMiddleware_Unauthorized ensures requests without credentials get 401.
func TestAuthMiddleware_Unauthorized(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Create a protected handler and start server
	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	// No X-API-Key and no session cookie
	req, _ := http.NewRequest("GET", srv.URL+"/api/config", nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// TestAuthMiddleware_Forbidden ensures wrong role gets 403.
func TestAuthMiddleware_Forbidden(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Create a viewer (insufficient permissions)
	if err := auth.CreateUser(db, "viewer", "p", "", "viewer"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("api key: %v", err)
	}

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/config", nil)
	req.Header.Set("X-API-Key", keyObj.GetId())
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

// TestAuthMiddleware_SessionCookie ensures session cookie authenticates.
func TestAuthMiddleware_SessionCookie(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Create admin and session
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	sess, err := auth.GenerateSession(db, 1, time.Hour)
	if err != nil {
		t.Fatalf("session: %v", err)
	}

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/config", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sess.GetId()})
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// TestConfigUpdate verifies that POST /api/config updates and persists values.
func TestConfigUpdate(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	testutil.MustNoError(t, "open db", err)
	defer db.Close()

	testutil.MustNoError(t, "create admin", auth.CreateUser(db, "admin", "p", "", "admin"))
	akeyObj, err := auth.GenerateAPIKey(db, 1)
	testutil.MustNoError(t, "admin key", err)

	tmp := filepath.Join(t.TempDir(), "cfg.yaml")
	viper.SetConfigFile(tmp)
	viper.Set("test_key", "old")
	testutil.MustNoError(t, "write config", viper.WriteConfig())
	defer viper.Reset()

	h, err := Handler(db)
	testutil.MustNoError(t, "create handler", err)
	srv := httptest.NewServer(h)
	defer srv.Close()

	body := strings.NewReader(`{"test_key":"new"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/api/config", body)
	req.Header.Set("X-API-Key", akeyObj.GetId())
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}

	if viper.GetString("test_key") != "new" {
		t.Fatalf("viper not updated")
	}
	data := testutil.MustGet(t, "read config", func() ([]byte, error) { return os.ReadFile(tmp) })
	if !strings.Contains(string(data), "test_key: new") {
		t.Fatalf("config not written")
	}
}

// TestScanHandlers verifies /api/scan and /api/scan/status.
func TestScanHandlers(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("api key: %v", err)
	}
	// fake subtitle server
	subSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("sub")) // nolint:errcheck
	}))
	defer subSrv.Close()
	viper.Set("providers.generic.api_url", subSrv.URL)
	defer viper.Reset()
	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	// create video file
	dir := t.TempDir()
	viper.Set("media_directory", dir)
	defer viper.Reset()
	vid := filepath.Join(dir, "file.mkv")
	_ = os.WriteFile(vid, []byte("x"), 0644) // nolint:errcheck
	// trigger scan
	body := strings.NewReader(`{"provider":"generic","directory":"` + dir + `","lang":"en"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/api/scan", body)
	req.Header.Set("X-API-Key", keyObj.GetId())
	resp, err := srv.Client().Do(req)
	if err != nil || resp.StatusCode != http.StatusAccepted {
		t.Fatalf("scan start: %v %d", err, resp.StatusCode)
	}
	// poll status until not running
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		req2, _ := http.NewRequest("GET", srv.URL+"/api/scan/status", nil)
		req2.Header.Set("X-API-Key", keyObj.GetId())
		r2, err := srv.Client().Do(req2)
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		var s struct {
			Running   bool `json:"running"`
			Completed int  `json:"completed"`
		}
		if err := json.NewDecoder(r2.Body).Decode(&s); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		r2.Body.Close()
		if !s.Running {
			if s.Completed != 1 {
				t.Fatalf("completed %d", s.Completed)
			}
			return
		}
	}
	t.Fatalf("scan did not finish")
}

// TestExtract verifies that POST /api/extract returns subtitle items.
func TestExtract(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("api key: %v", err)
	}

	dir := t.TempDir()
	viper.Set("media_directory", dir)
	defer viper.Reset()
	script := filepath.Join(dir, "ffmpeg")
	data := "#!/bin/sh\ncp ../../testdata/simple.srt \"$6\"\n"
	if err := os.WriteFile(script, []byte(data), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	h, err := Handler(db)
	h = testutil.Must(t, "handler creation", h, err)
	srv := httptest.NewServer(h)
	defer srv.Close()

	dummyPath := filepath.Join(dir, "dummy.mkv")
	body := strings.NewReader(fmt.Sprintf(`{"path":%q}`, dummyPath))
	req, _ := http.NewRequest("POST", srv.URL+"/api/extract", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", keyObj.GetId())
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var items []any
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("no items returned")
	}
}

// TestConvert verifies that POST /api/convert returns converted data.
func TestConvert(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	key := setupTestUser(t, db)

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	data, err := os.ReadFile("../../testdata/simple.srt")
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	fw, err := mw.CreateFormFile("file", "simple.srt")
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	fw.Write(data)
	mw.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/convert", buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-API-Key", key)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("status %d, body: %s", resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(body) == 0 {
		t.Fatalf("no data returned")
	}
}

// TestTranslate verifies the subtitle translation API.
func TestTranslate(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Mock the Google Translate API
	// One translation per input string, as the real API returns.
	//
	// This mock previously returned a single fixed translation regardless of
	// how many texts were sent. testdata/simple.srt has two cues, so the
	// translator rejected the response with "expected 2 translations, got 1"
	// and the handler answered 500 — with the error discarded, so nothing said
	// why.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		queries := r.Form["q"]
		translations := make([]map[string]string, 0, len(queries))
		for _, q := range queries {
			translations = append(translations, map[string]string{
				"translatedText": "hola " + q,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"translations": translations},
		}); err != nil {
			t.Errorf("encode mock response: %v", err)
		}
	}))
	defer ts.Close()
	translator.SetGoogleAPIURL(ts.URL)
	defer translator.SetGoogleAPIURL("https://translation.googleapis.com/language/translate/v2")

	viper.Set("google_api_key", "test-key")
	defer viper.Reset()

	key := setupTestUser(t, db)

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	b, _ := os.ReadFile("../../testdata/simple.srt")
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, _ := writer.CreateFormFile("file", "in.srt")
	fw.Write(b)
	writer.WriteField("lang", "es")
	writer.WriteField("service", "google")
	writer.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/translate", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", key)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	out, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !bytes.Contains(out, []byte("hola")) {
		t.Fatalf("unexpected output: %s", out)
	}
}

// TestSetup verifies the initial setup workflow.
func TestSetup(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	// status should report setup needed
	resp, err := http.Get(srv.URL + "/api/setup/status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	var st struct {
		Needed bool `json:"needed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	resp.Body.Close()
	if !st.Needed {
		t.Fatalf("expected setup needed")
	}

	body := strings.NewReader(`{"server_name":"Test","admin_user":"a","admin_pass":"p"}`)
	resp2, err := http.Post(srv.URL+"/api/setup", "application/json", body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp2.StatusCode)
	}

	// now status should be false
	resp3, err := http.Get(srv.URL + "/api/setup/status")
	if err != nil {
		t.Fatalf("status2: %v", err)
	}
	if err := json.NewDecoder(resp3.Body).Decode(&st); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	resp3.Body.Close()
	if st.Needed {
		t.Fatalf("setup still needed")
	}
}

// TestHistory verifies that /api/history returns history records and filters by language.
func TestHistory(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("api key: %v", err)
	}
	_ = database.InsertSubtitle(db, "a.srt", "a.mkv", "en", "google", "", false)
	_ = database.InsertDownload(db, "b.srt", "b.mkv", "os", "en")
	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/api/history", nil)
	req.Header.Set("X-API-Key", keyObj.GetId())
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out struct {
		Translations []database.SubtitleRecord `json:"translations"`
		Downloads    []database.DownloadRecord `json:"downloads"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	resp.Body.Close()
	if len(out.Translations) != 1 || len(out.Downloads) != 1 {
		t.Fatalf("unexpected counts %d %d", len(out.Translations), len(out.Downloads))
	}
	req2, _ := http.NewRequest("GET", srv.URL+"/api/history?lang=fr", nil)
	req2.Header.Set("X-API-Key", keyObj.GetId())
	resp2, _ := srv.Client().Do(req2)
	var out2 struct {
		Translations []database.SubtitleRecord `json:"translations"`
		Downloads    []database.DownloadRecord `json:"downloads"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&out2); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	resp2.Body.Close()
	if len(out2.Translations) != 0 || len(out2.Downloads) != 0 {
		t.Fatalf("filter failed")
	}
}

// TestHistoryVideoFilter verifies the video file filter works.
func TestHistoryVideoFilter(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	keyObj, _ := auth.GenerateAPIKey(db, 1)
	_ = database.InsertSubtitle(db, "a.srt", "a.mkv", "en", "google", "", false)
	_ = database.InsertSubtitle(db, "b.srt", "b.mkv", "en", "google", "", false)
	h, _ := Handler(db)
	srv := httptest.NewServer(h)
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/api/history?video=b.mkv", nil)
	req.Header.Set("X-API-Key", keyObj.GetId())
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	var out struct {
		Translations []database.SubtitleRecord `json:"translations"`
		Downloads    []database.DownloadRecord `json:"downloads"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	resp.Body.Close()
	if len(out.Translations) != 1 || out.Translations[0].VideoFile != "b.mkv" {
		t.Fatalf("unexpected filter result")
	}
}

// TestDownload verifies that POST /api/download fetches a subtitle and records history.
func TestDownload(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("api key: %v", err)
	}

	// fake subtitle provider
	subSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("sub"))
	}))
	defer subSrv.Close()
	viper.Set("providers.generic.api_url", subSrv.URL)
	defer viper.Reset()

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	dir := t.TempDir()
	viper.Set("media_directory", dir)
	defer viper.Reset()
	vid := filepath.Join(dir, "video.mkv")
	os.WriteFile(vid, []byte("x"), 0644)

	body := strings.NewReader(`{"provider":"generic","path":"` + vid + `","lang":"en"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/api/download", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", keyObj.GetId())
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var res struct {
		File string `json:"file"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	resp.Body.Close()

	out := strings.TrimSuffix(vid, filepath.Ext(vid)) + ".en.srt"
	if res.File != out {
		t.Fatalf("returned %s", res.File)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("subtitle not written: %v", err)
	}
	recs, err := database.ListDownloads(db)
	if err != nil || len(recs) != 1 {
		t.Fatalf("records %v %d", err, len(recs))
	}
	if recs[0].File != out {
		t.Fatalf("record file %s", recs[0].File)
	}
}

// TestWebhookSonarr verifies that Sonarr webhook triggers subtitle download.
func TestWebhookSonarr(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	subSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("sub")) }))
	defer subSrv.Close()
	viper.Set("providers.generic.api_url", subSrv.URL)
	defer viper.Reset()

	dir := t.TempDir()
	viper.Set("media_directory", dir)
	defer viper.Reset()
	vid := filepath.Join(dir, "file.mkv")
	os.WriteFile(vid, []byte("x"), 0644)

	body := strings.NewReader(`{"provider":"generic","path":"` + vid + `","lang":"en"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/api/webhooks/sonarr", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
	out := strings.TrimSuffix(vid, filepath.Ext(vid)) + ".en.srt"
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("subtitle not written")
	}
}

// TestConvertUpload verifies that /api/convert returns an SRT file.
func TestConvertUpload(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("api key: %v", err)
	}

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	fw, _ := mw.CreateFormFile("file", "in.srt")
	data, _ := os.ReadFile("../../testdata/simple.srt")
	fw.Write(data)
	mw.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/convert", buf)
	req.Header.Set("X-API-Key", keyObj.GetId())
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("status %d, body: %s", resp.StatusCode, string(body))
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if len(b) == 0 {
		t.Fatalf("no data returned")
	}
}

// TestTranslateUpload verifies that /api/translate performs translation using Google.
func TestTranslateUpload(t *testing.T) {
	skipIfNoSQLite(t)
	// Mock the Google Translate API
	srvTrans := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// One translation per input string, as the real API returns; a single
		// fixed translation fails the translator's count check for any
		// subtitle with more than one cue.
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		translations := make([]map[string]string, 0, len(r.Form["q"]))
		for _, q := range r.Form["q"] {
			translations = append(translations, map[string]string{
				"translatedText": "hola " + q,
			})
		}
		if err := json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"translations": translations},
		}); err != nil {
			t.Errorf("encode mock response: %v", err)
		}
	}))
	defer srvTrans.Close()
	translator.SetGoogleAPIURL(srvTrans.URL)
	defer translator.SetGoogleAPIURL("https://translation.googleapis.com/language/translate/v2")

	db, err := database.Open(":memory:")
	testutil.MustNoError(t, "open db", err)
	defer db.Close()
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("api key: %v", err)
	}

	viper.Set("translate_service", "google")
	viper.Set("google_api_key", "test-key")
	defer viper.Reset()

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	fw, _ := mw.CreateFormFile("file", "in.srt")
	data, _ := os.ReadFile("../../testdata/simple.srt")
	fw.Write(data)
	mw.WriteField("lang", "es")
	mw.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/translate", buf)
	req.Header.Set("X-API-Key", keyObj.GetId())
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("status %d, body: %s", resp.StatusCode, string(body))
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !bytes.Contains(b, []byte("hola")) {
		t.Fatalf("translation missing: %s", string(b))
	}
}

// TestProvidersDefault verifies that the providers endpoint returns
// all available providers for UI configuration, with only embedded enabled by default.
func TestProvidersDefault(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	key := setupTestUser(t, db)

	viper.Reset()
	viper.SetDefault("providers.embedded.enabled", true)

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/providers", nil)
	req.Header.Set("X-API-Key", key)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("get providers: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out []ProviderInfo
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	resp.Body.Close()
	if len(out) != 52 {
		t.Fatalf("expected 52 providers, got %d", len(out))
	}

	// Verify that embedded provider is included and enabled by default
	var embeddedProvider *ProviderInfo
	for i := range out {
		if out[i].Name == "embedded" {
			embeddedProvider = &out[i]
			break
		}
	}
	if embeddedProvider == nil {
		t.Fatalf("embedded provider not found in provider list")
	}
	if !embeddedProvider.Enabled {
		t.Fatalf("embedded provider should be enabled by default, got %+v", embeddedProvider)
	}
}

// TestProviderConfigUpdate verifies that POST /api/providers updates provider configuration
// and persists the change to the config file.
func TestProviderConfigUpdate(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	key := setupTestUser(t, db)

	tmp := filepath.Join(t.TempDir(), "cfg.yaml")
	viper.SetConfigFile(tmp)
	viper.Set("providers.generic.enabled", false)
	viper.Set("providers.generic.config", map[string]any{"api_url": "old"})
	if err := viper.WriteConfig(); err != nil {
		t.Fatalf("write config: %v", err)
	}
	defer viper.Reset()

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	body := strings.NewReader(`{"name":"generic","enabled":true,"config":{"api_url":"new"}}`)
	req, _ := http.NewRequest("POST", srv.URL+"/api/providers", body)
	req.Header.Set("X-API-Key", key)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
	resp.Body.Close()

	if !viper.GetBool("providers.generic.enabled") {
		t.Fatalf("config not updated: enabled")
	}
	cfg := viper.GetStringMapString("providers.generic.config")
	if cfg["api_url"] != "new" {
		t.Fatalf("config not updated: %v", cfg)
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("read cfg: %v", err)
	}
	if !strings.Contains(string(data), "api_url: new") {
		t.Fatalf("file not written")
	}
}

// TestSyncBatchEndpoint verifies that POST /api/sync/batch processes multiple
// subtitle sync requests and writes the outputs.
func TestSyncBatchEndpoint(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	key := setupTestUser(t, db)

	dir := t.TempDir()
	data, err := os.ReadFile("../../testdata/simple.srt")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	in1 := filepath.Join(dir, "a.srt")
	os.WriteFile(in1, data, 0644)
	in2 := filepath.Join(dir, "b.srt")
	os.WriteFile(in2, data, 0644)
	out1 := filepath.Join(dir, "out1.srt")
	out2 := filepath.Join(dir, "out2.srt")

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	body := strings.NewReader(fmt.Sprintf(`{"items":[{"media":"m1.mkv","subtitle":"%s","output":"%s"},{"media":"m2.mkv","subtitle":"%s","output":"%s"}],"options":{}}`, in1, out1, in2, out2))
	req, _ := http.NewRequest("POST", srv.URL+"/api/sync/batch", body)
	req.Header.Set("X-API-Key", key)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out struct {
		Results []struct {
			Output string `json:"output"`
			Error  string `json:"error"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	resp.Body.Close()
	if len(out.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(out.Results))
	}
	for i, p := range []string{out1, out2} {
		if out.Results[i].Error != "" {
			t.Fatalf("result %d error: %s", i, out.Results[i].Error)
		}
		if fi, err := os.Stat(p); err != nil || fi.Size() == 0 {
			t.Fatalf("output %d not written", i)
		}
	}
}

// TestSyncBatchEndpointInvalidJSON verifies that invalid JSON input
// results in a 400 Bad Request response when hitting POST /api/sync/batch.
func TestSyncBatchEndpointInvalidJSON(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	key := setupTestUser(t, db)

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/sync/batch", strings.NewReader("{invalid"))
	req.Header.Set("X-API-Key", key)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

// TestProvidersUpdateInvalid verifies that invalid JSON results in 400 Bad Request.
func TestProvidersUpdateInvalid(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	key := setupTestUser(t, db)

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/api/providers", strings.NewReader("{invalid"))
	req.Header.Set("X-API-Key", key)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

// TestProvidersListAfterUpdate ensures that provider changes persist
// when fetching the provider list via GET /api/providers.
func TestProvidersListAfterUpdate(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	key := setupTestUser(t, db)

	tmp := filepath.Join(t.TempDir(), "cfg.yaml")
	viper.SetConfigFile(tmp)
	viper.Set("providers.generic.enabled", false)
	viper.Set("providers.generic.config", map[string]interface{}{"api_url": ""})
	testutil.MustNoError(t, "write config", viper.WriteConfig())
	defer viper.Reset()

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	body := strings.NewReader(`{"name":"generic","enabled":true,"config":{"api_url":"http://example.com"}}`)
	req, _ := http.NewRequest("POST", srv.URL+"/api/providers", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", key)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}

	req2, _ := http.NewRequest("GET", srv.URL+"/api/providers", nil)
	req2.Header.Set("X-API-Key", key)
	resp2, err := srv.Client().Do(req2)
	if err != nil {
		t.Fatalf("get providers: %v", err)
	}
	defer resp2.Body.Close()
	var list []struct {
		Name    string                 `json:"name"`
		Enabled bool                   `json:"enabled"`
		Config  map[string]interface{} `json:"config"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, p := range list {
		if p.Name == "generic" {
			found = true
			if !p.Enabled {
				t.Fatalf("provider should be enabled")
			}
			if p.Config["api_url"] != "http://example.com" {
				t.Fatalf("config not returned: %v", p.Config["api_url"])
			}
		}
	}
	if !found {
		t.Fatalf("generic provider missing in list")
	}
}

// TestExtractLanguageFromFilename verifies default detection and pattern overrides.
func TestExtractLanguageFromFilename(t *testing.T) {
	ResetLanguagePatterns()

	cases := map[string]string{
		"movie.es.srt":      "Spanish",
		"movie.zh-cn.srt":   "Chinese (Simplified)",
		"movie.zh-hk.srt":   "Chinese (Hong Kong)",
		"movie.zh-hant.srt": "Chinese (Traditional)",
		"movie.zh-tw.srt":   "Chinese (Taiwan)",
	}
	for name, want := range cases {
		if lang := extractLanguageFromFilename(name); lang != want {
			t.Fatalf("%s expected %s, got %s", name, want, lang)
		}
	}

	SetLanguagePatterns(map[string]string{"xx": "TestLang"})
	defer ResetLanguagePatterns()
	if lang := extractLanguageFromFilename("movie.xx.srt"); lang != "TestLang" {
		t.Fatalf("override failed: %s", lang)
	}
}

// setupTestUser creates a test user with an API key and returns the key.
func setupTestUser(t *testing.T, db *sql.DB) string {
	testutil.MustNoError(t, "create admin", auth.CreateUser(db, "admin", "p", "", "admin"))
	return testutil.MustGet(t, "api key", func() (string, error) {
		k, err := auth.GenerateAPIKey(db, 1)
		if err != nil {
			return "", err
		}
		return k.GetId(), nil
	})
}

// TestSecurityHeaders ensures that the server sets common security headers.
func TestSecurityHeaders(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	key := setupTestUser(t, db)

	h, err := Handler(db)
	testutil.MustNoError(t, "handler", err)
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/history", nil)
	req.Header.Set("X-API-Key", key)
	resp := testutil.MustGet(t, "request", func() (*http.Response, error) { return srv.Client().Do(req) })

	headers := resp.Header
	for _, name := range []string{"X-Frame-Options", "X-Content-Type-Options", "Content-Security-Policy", "Strict-Transport-Security", "Referrer-Policy"} {
		if headers.Get(name) == "" {
			t.Fatalf("%s header missing", name)
		}
	}
}

// TestBrowseDirectoryPathTraversalPrevention tests that browseDirectory prevents path traversal attacks
func TestBrowseDirectoryPathTraversalPrevention(t *testing.T) {
	// Create a temporary directory to serve as the allowed base directory
	tempDir := t.TempDir()

	// Set up viper config to use our temp directory as media directory
	originalMediaDir := viper.GetString("media_directory")
	viper.Set("media_directory", tempDir)
	defer viper.Set("media_directory", originalMediaDir)

	// Create some valid subdirectories and files for testing
	validSubdir := filepath.Join(tempDir, "movies")
	if err := os.Mkdir(validSubdir, 0755); err != nil {
		t.Fatalf("failed to create test subdirectory: %v", err)
	}

	validFile := filepath.Join(validSubdir, "movie.mkv")
	if err := os.WriteFile(validFile, []byte("test movie"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid path within allowed directory",
			path:        validSubdir,
			expectError: false,
		},
		{
			name:        "root path should show allowed directories",
			path:        "/",
			expectError: false,
		},
		{
			name:        "path traversal attack with ../ should fail",
			path:        filepath.Join(tempDir, "../../../etc/passwd"),
			expectError: true,
			errorMsg:    "path not in allowed directories",
		},
		{
			name:        "absolute path outside allowed directories should fail",
			path:        "/etc/passwd",
			expectError: true,
			errorMsg:    "path not in allowed directories",
		},
		{
			name:        "relative path should fail",
			path:        "relative/path",
			expectError: true,
			errorMsg:    "path must be absolute",
		},
		{
			name:        "path with .. components should fail",
			path:        validSubdir + "/../../../../usr/bin/passwd",
			expectError: true,
			errorMsg:    "path not in allowed directories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := browseDirectory(tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for path %q, but got none", tt.path)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for path %q: %v", tt.path, err)
				}
				if items == nil {
					t.Errorf("expected items to be non-nil for valid path %q", tt.path)
				}

				// For valid paths, verify we get meaningful results
				if tt.path == "/" {
					// Root path should return allowed base directories
					found := false
					for _, item := range items {
						if item.Path == tempDir && item.IsDirectory {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("root listing should include temp directory %q", tempDir)
					}
				} else if tt.path == validSubdir {
					// Valid subdirectory should contain our test file
					found := false
					for _, item := range items {
						if strings.Contains(item.Name, "movie.mkv") && !item.IsDirectory {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("subdirectory listing should include test movie file")
					}
				}
			}
		})
	}
}

// TestLibraryBrowsePathTraversal verifies that path traversal attacks are prevented
// in the library browse endpoint.
func TestLibraryBrowsePathTraversal(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	testutil.MustNoError(t, "create user", auth.CreateUser(db, "test", "pass", "", "admin"))
	keyObj, err := auth.GenerateAPIKey(db, 1)
	testutil.MustNoError(t, "generate key", err)

	h, err := Handler(db)
	testutil.MustNoError(t, "handler", err)
	srv := httptest.NewServer(h)
	defer srv.Close()

	// Test various path traversal attempts
	pathTraversalAttempts := []string{
		"../../../etc/passwd",
		"../../../../etc/shadow",
		"..\\..\\..\\windows\\system32",
		"/etc/passwd",
		"/etc/shadow",
		"../../../../../root/.ssh/id_rsa",
	}

	for _, maliciousPath := range pathTraversalAttempts {
		sanitizedPath := sanitizeSubtestName(maliciousPath)
		t.Run("path_traversal_"+sanitizedPath, func(t *testing.T) {
			escapedPath := url.QueryEscape(maliciousPath)
			req, _ := http.NewRequest("GET", srv.URL+"/api/library/browse?path="+escapedPath, nil)
			req.Header.Set("X-API-Key", keyObj.GetId())
			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			// Path traversal attempts should return 400 Bad Request
			if resp.StatusCode != http.StatusBadRequest {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 400 Bad Request for path traversal attempt %q, got %d: %s",
					maliciousPath, resp.StatusCode, string(body))
			}
		})
	}
}

// TestLibraryBrowseJSONStructure verifies the handler returns items in a wrapper object.
func TestLibraryBrowseJSONStructure(t *testing.T) {
	skipIfNoSQLite(t)
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	testutil.MustNoError(t, "create user", auth.CreateUser(db, "test", "pass", "", "admin"))
	keyObj, err := auth.GenerateAPIKey(db, 1)
	testutil.MustNoError(t, "generate key", err)

	h, err := Handler(db)
	testutil.MustNoError(t, "handler", err)
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/library/browse?path=/", nil)
	req.Header.Set("X-API-Key", keyObj.GetId())
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out struct {
		Items []MediaItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Items == nil {
		t.Fatalf("items missing")
	}
}

// sanitizeSubtestName removes problematic characters from test names to make them valid.
func sanitizeSubtestName(name string) string {
	// Replace path separators and other problematic characters with underscores
	sanitized := strings.ReplaceAll(name, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	return sanitized
}
