// file: pkg/webserver/profiles_test.go
// version: 1.1.0
// guid: 1b0c9d8e-7f6e-2a3b-5c4d-8f7e9a0b1c2d
// last-edited: 2026-07-25

package webserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/profiles"
	"github.com/spf13/viper"
)

// useTempProfileStore points the profile handlers at a throwaway SQLite
// database for the duration of the test.
//
// The handlers open their own store via database.OpenStoreWithConfig rather
// than using the *sql.DB passed to them, so the backend has to be configured
// through viper. The previous tests worked around this by asserting the 500
// that a missing store produced — locking in a failure rather than testing
// behaviour, which is why they inverted and started failing once the store
// opened successfully.
func useTempProfileStore(t *testing.T) {
	t.Helper()
	prevBackend := viper.GetString("db_backend")
	prevPath := viper.GetString("db_path")
	t.Cleanup(func() {
		viper.Set("db_backend", prevBackend)
		viper.Set("db_path", prevPath)
	})
	viper.Set("db_backend", "sqlite")
	viper.Set("db_path", filepath.Join(t.TempDir(), "profiles.db"))
}

func TestProfilesHandlerList(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	handler := profilesHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/profiles", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d (body: %s)",
			http.StatusOK, rr.Code, rr.Body.String())
	}

	// An empty store must still yield valid JSON the UI can iterate over.
	var got []profiles.LanguageProfile
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not a JSON profile array: %v (body: %s)",
			err, rr.Body.String())
	}
}

func TestProfilesHandlerCreate(t *testing.T) {
	// Test profile creation payload validation
	profile := profiles.LanguageProfile{
		Name:        "Test Profile",
		Languages:   []profiles.LanguageConfig{{Language: "en", Priority: 1, Forced: false, HI: false}},
		CutoffScore: 75,
		IsDefault:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	body, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}

	skipIfNoSQLite(t)
	useTempProfileStore(t)

	handler := profilesHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/profiles", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		t.Fatalf("Expected 200 or 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// The created profile must come back from a subsequent list, otherwise the
	// write silently went nowhere.
	listReq := httptest.NewRequest(http.MethodGet, "/api/profiles", nil)
	listRR := httptest.NewRecorder()
	handler.ServeHTTP(listRR, listReq)

	var got []profiles.LanguageProfile
	if err := json.Unmarshal(listRR.Body.Bytes(), &got); err != nil {
		t.Fatalf("list response is not a JSON profile array: %v", err)
	}
	for _, p := range got {
		if p.Name == "Test Profile" {
			return
		}
	}
	t.Errorf("created profile %q not present in list (got %d profiles)",
		"Test Profile", len(got))
}

func TestProfilesHandlerInvalidJSON(t *testing.T) {
	handler := profilesHandler(nil)

	req, err := http.NewRequest("POST", "/api/profiles", bytes.NewBufferString("invalid json"))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should return bad request for invalid JSON
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestProfilesHandlerMethodNotAllowed(t *testing.T) {
	handler := profilesHandler(nil)

	req, err := http.NewRequest("PATCH", "/api/profiles", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should return method not allowed
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestProfilesHandlerValidation(t *testing.T) {
	// Test validation of profile with empty name
	profile := profiles.LanguageProfile{
		Name:        "", // Invalid: empty name
		Languages:   []profiles.LanguageConfig{{Language: "en", Priority: 1, Forced: false, HI: false}},
		CutoffScore: 75,
	}

	body, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}

	handler := profilesHandler(nil)

	req, err := http.NewRequest("POST", "/api/profiles", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should return bad request for validation error
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestMediaProfilesHandler(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	handler := mediaProfilesHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/media/profile/123", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Media 123 has no assignment. Any answer is acceptable except a 5xx: the
	// handler must resolve the store and report "no assignment" rather than
	// fall over. Asserting the exact code would pin down an unspecified
	// not-found representation.
	if rr.Code >= 500 {
		t.Errorf("unassigned media returned server error %d (body: %s)",
			rr.Code, rr.Body.String())
	}
}

func TestMediaProfilesHandlerBadPath(t *testing.T) {
	handler := mediaProfilesHandler(nil)

	req, err := http.NewRequest("GET", "/api/media/profile/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should return bad request for empty path
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
