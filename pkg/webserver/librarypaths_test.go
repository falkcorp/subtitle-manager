// file: pkg/webserver/librarypaths_test.go
// version: 1.0.0
// guid: 1b2c62c8-236d-4c33-bdb5-23d18ff8bcdd

package webserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// TestLibraryPathsHandler verifies GET/POST/DELETE of configured library paths,
// which the Media Library UI needs (previously the route was missing, so the
// library was always empty and "Add Library Path" failed).
func TestLibraryPathsHandler(t *testing.T) {
	dir := t.TempDir()
	viper.Set("db_path", dir)
	defer viper.Reset()

	h := libraryPathsHandler()

	get := func() []string {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/library/paths", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET expected 200, got %d", rec.Code)
		}
		var out struct {
			Paths []string `json:"paths"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("bad json: %v (%s)", err, rec.Body.String())
		}
		return out.Paths
	}

	if p := get(); len(p) != 0 {
		t.Fatalf("expected no paths initially, got %v", p)
	}

	// POST a valid path (the temp dir is allowed by ValidateAndSanitizePath).
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/library/paths", strings.NewReader(`{"path":"`+dir+`"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if p := get(); len(p) != 1 || p[0] != dir {
		t.Fatalf("expected [%s], got %v", dir, p)
	}

	// DELETE it.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/library/paths", strings.NewReader(`{"path":"`+dir+`"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE expected 200, got %d", rec.Code)
	}
	if p := get(); len(p) != 0 {
		t.Fatalf("expected empty after delete, got %v", p)
	}
}

// TestLibraryPathsBadRequest verifies invalid input is rejected.
func TestLibraryPathsBadRequest(t *testing.T) {
	viper.Set("db_path", t.TempDir())
	defer viper.Reset()
	rec := httptest.NewRecorder()
	libraryPathsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/library/paths", strings.NewReader(`{}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty path, got %d", rec.Code)
	}
}
