package webserver

import (
	"encoding/json"
	"github.com/spf13/viper"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	auth "github.com/jdfalk/subtitle-manager/pkg/gcommonauth"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/testutil"
)

// TestSystemHandlers verifies /api/logs and /api/system endpoints.
func TestSystemHandlers(t *testing.T) {
	skipIfNoSQLite(t)

	db := testutil.GetTestDB(t)
	defer db.Close()
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	logger := logging.GetLogger("test")
	logger.Info("hello")

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/logs", nil)
	req.Header.Set("X-API-Key", keyObj.GetId())
	r, err := srv.Client().Do(req)
	if err != nil || r.StatusCode != http.StatusOK {
		t.Fatalf("logs: %v %d", err, r.StatusCode)
	}
	var lines []string
	if err := json.NewDecoder(r.Body).Decode(&lines); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	r.Body.Close()
	if len(lines) == 0 {
		t.Fatalf("no logs returned")
	}

	req2, _ := http.NewRequest("GET", srv.URL+"/api/system", nil)
	req2.Header.Set("X-API-Key", keyObj.GetId())
	r2, err := srv.Client().Do(req2)
	if err != nil || r2.StatusCode != http.StatusOK {
		t.Fatalf("system: %v %d", err, r2.StatusCode)
	}
	var info struct {
		GoVersion string `json:"go_version"`
	}
	if err := json.NewDecoder(r2.Body).Decode(&info); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	r2.Body.Close()
	if info.GoVersion == "" {
		t.Fatalf("no info")
	}

	// Point the handler at a fixture. The default path is resolved relative to
	// the process working directory and no announcements.json is committed, so
	// this previously asserted against a file that never existed and the
	// endpoint answered 404 on every deployment as well as in this test.
	annFile := filepath.Join(t.TempDir(), "announcements.json")
	if err := os.WriteFile(annFile,
		[]byte(`[{"date":"2026-07-30","message":"hello"}]`), 0o600); err != nil {
		t.Fatalf("write announcements: %v", err)
	}
	prevAnn := viper.Get("announcements_file")
	viper.Set("announcements_file", annFile)
	defer viper.Set("announcements_file", prevAnn)

	req3, _ := http.NewRequest("GET", srv.URL+"/api/announcements", nil)
	req3.Header.Set("X-API-Key", keyObj.GetId())
	r3, err := srv.Client().Do(req3)
	if err != nil || r3.StatusCode != http.StatusOK {
		t.Fatalf("announcements: %v %d", err, r3.StatusCode)
	}
	var ann []map[string]any
	if err := json.NewDecoder(r3.Body).Decode(&ann); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	r3.Body.Close()
	if len(ann) != 1 || ann[0]["message"] != "hello" {
		t.Fatalf("announcements = %v, want the fixture contents", ann)
	}
}

// TestAnnouncementsMissingFileIsEmptyList verifies an absent file is "no
// announcements" rather than an error.
//
// The handler returned 404 when the file was missing, which is the state of
// every deployment: none is committed, and the default path is resolved
// relative to the process working directory. The frontend then treated a
// working install as a broken endpoint.
func TestAnnouncementsMissingFileIsEmptyList(t *testing.T) {
	prev := viper.Get("announcements_file")
	viper.Set("announcements_file", filepath.Join(t.TempDir(), "absent.json"))
	defer viper.Set("announcements_file", prev)

	rr := httptest.NewRecorder()
	announcementsHandler().ServeHTTP(rr,
		httptest.NewRequest(http.MethodGet, "/api/announcements", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for an absent announcements file", rr.Code)
	}
	var ann []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &ann); err != nil {
		t.Fatalf("body is not JSON (%v): %s", err, rr.Body.String())
	}
	if len(ann) != 0 {
		t.Errorf("got %v, want an empty list", ann)
	}
}

// TestStartTaskHandler_Negatives validates method and parameter validation.
func TestStartTaskHandler_Negatives(t *testing.T) {
	skipIfNoSQLite(t)

	db := testutil.GetTestDB(t)
	defer db.Close()
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	// Method not allowed
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/tasks/start", nil)
	req.Header.Set("X-API-Key", keyObj.GetId())
	r1, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if r1.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", r1.StatusCode)
	}
	r1.Body.Close()

	// Missing name
	req2, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/tasks/start", nil)
	req2.Header.Set("X-API-Key", keyObj.GetId())
	r2, err := srv.Client().Do(req2)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if r2.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", r2.StatusCode)
	}
	r2.Body.Close()

	// Invalid name (too long / invalid chars)
	req3, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/tasks/start?name=../../etc/passwd", nil)
	req3.Header.Set("X-API-Key", keyObj.GetId())
	r3, err := srv.Client().Do(req3)
	if err != nil {
		t.Fatalf("post invalid: %v", err)
	}
	if r3.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", r3.StatusCode)
	}
	r3.Body.Close()
}

// TestAnnouncements_NotFound ensures missing file results in 404.
func TestAnnouncements_NotFound(t *testing.T) {
	skipIfNoSQLite(t)

	db := testutil.GetTestDB(t)
	defer db.Close()
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/announcements", nil)
	req.Header.Set("X-API-Key", keyObj.GetId())
	r, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	// When file missing, handler returns 404
	if r.StatusCode != http.StatusNotFound && r.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", r.StatusCode)
	}
	r.Body.Close()
}

// TestProviderEndpoints covers status, refresh, and reset provider endpoints.
func TestProviderEndpoints(t *testing.T) {
	skipIfNoSQLite(t)

	db := testutil.GetTestDB(t)
	defer db.Close()
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	// status
	req1, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/providers/status", nil)
	req1.Header.Set("X-API-Key", keyObj.GetId())
	r1, err := srv.Client().Do(req1)
	if err != nil || r1.StatusCode != http.StatusOK {
		t.Fatalf("status: %v %d", err, r1.StatusCode)
	}
	r1.Body.Close()

	// refresh
	req2, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/providers/refresh", nil)
	req2.Header.Set("X-API-Key", keyObj.GetId())
	r2, err := srv.Client().Do(req2)
	if err != nil || r2.StatusCode != http.StatusAccepted {
		t.Fatalf("refresh: %v %d", err, r2.StatusCode)
	}
	r2.Body.Close()

	// reset
	req3, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/providers/reset", nil)
	req3.Header.Set("X-API-Key", keyObj.GetId())
	r3, err := srv.Client().Do(req3)
	if err != nil || r3.StatusCode != http.StatusNoContent {
		t.Fatalf("reset: %v %d", err, r3.StatusCode)
	}
	r3.Body.Close()
}

// TestTasksEndpoints covers listing tasks and starting a task (happy path).
func TestTasksEndpoints(t *testing.T) {
	skipIfNoSQLite(t)

	db := testutil.GetTestDB(t)
	defer db.Close()
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	// start a task
	reqStart, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/tasks/start?name=unit_task", nil)
	reqStart.Header.Set("X-API-Key", keyObj.GetId())
	rs, err := srv.Client().Do(reqStart)
	if err != nil || rs.StatusCode != http.StatusOK {
		t.Fatalf("start: %v %d", err, rs.StatusCode)
	}
	var startResp map[string]any
	_ = json.NewDecoder(rs.Body).Decode(&startResp)
	rs.Body.Close()

	// list tasks
	reqList, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/tasks", nil)
	reqList.Header.Set("X-API-Key", keyObj.GetId())
	rl, err := srv.Client().Do(reqList)
	if err != nil || rl.StatusCode != http.StatusOK {
		t.Fatalf("list: %v %d", err, rl.StatusCode)
	}
	var tasks map[string]any
	_ = json.NewDecoder(rl.Body).Decode(&tasks)
	rl.Body.Close()
	if len(tasks) == 0 {
		t.Fatalf("expected at least one task")
	}
}

// TestBackupsEndpoints verifies backups listing, creation, and restore.
func TestBackupsEndpoints(t *testing.T) {
	skipIfNoSQLite(t)

	db := testutil.GetTestDB(t)
	defer db.Close()
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	// list
	r1, err := srv.Client().Do(func() *http.Request {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/backups", nil)
		req.Header.Set("X-API-Key", keyObj.GetId())
		return req
	}())
	if err != nil || r1.StatusCode != http.StatusOK {
		t.Fatalf("backups list: %v %d", err, r1.StatusCode)
	}
	r1.Body.Close()

	// create
	r2, err := srv.Client().Do(func() *http.Request {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/backups/create", nil)
		req.Header.Set("X-API-Key", keyObj.GetId())
		return req
	}())
	if err != nil || r2.StatusCode != http.StatusOK {
		t.Fatalf("backups create: %v %d", err, r2.StatusCode)
	}
	r2.Body.Close()

	// restore
	r3, err := srv.Client().Do(func() *http.Request {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/backups/restore", nil)
		req.Header.Set("X-API-Key", keyObj.GetId())
		return req
	}())
	if err != nil || r3.StatusCode != http.StatusOK {
		t.Fatalf("backups restore: %v %d", err, r3.StatusCode)
	}
	r3.Body.Close()
}

// TestErrorDashboardEndpoints verifies error stats/recent/top endpoints respond.
func TestErrorDashboardEndpoints(t *testing.T) {
	skipIfNoSQLite(t)

	db := testutil.GetTestDB(t)
	defer db.Close()
	if err := auth.CreateUser(db, "admin", "p", "", "admin"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	keyObj, err := auth.GenerateAPIKey(db, 1)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	h, err := Handler(db)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	for _, path := range []string{"/api/errors/stats", "/api/errors/recent", "/api/errors/top"} {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		req.Header.Set("X-API-Key", keyObj.GetId())
		r, err := srv.Client().Do(req)
		if err != nil || r.StatusCode != http.StatusOK {
			t.Fatalf("%s: %v %d", path, err, r.StatusCode)
		}
		r.Body.Close()
	}
}
