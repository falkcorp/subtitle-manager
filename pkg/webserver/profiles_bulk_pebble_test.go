// file: pkg/webserver/profiles_bulk_pebble_test.go
// version: 1.0.0
// guid: e5c81b40-93a7-4d62-a018-27f6b4c5309e
// last-edited: 2026-08-04

package webserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// usePebbleProfileStore points the profile handlers at a throwaway Pebble store.
//
// The existing profile tests configure SQLite, which is exactly why the bug this
// file covers survived them: SQLStore reports a missing profile with
// sql.ErrNoRows, so an "err != nil means not found" check happens to work there.
// PebbleStore returns (nil, nil) instead — and Pebble is the default backend.
func usePebbleProfileStore(t *testing.T) {
	t.Helper()
	prevBackend := viper.GetString("db_backend")
	prevPath := viper.GetString("db_path")
	t.Cleanup(func() {
		viper.Set("db_backend", prevBackend)
		viper.Set("db_path", prevPath)
	})
	viper.Set("db_backend", "pebble")
	viper.Set("db_path", filepath.Join(t.TempDir(), "db"))
}

// TestBulkRejectsUnknownProfileOnPebble is the regression.
//
// Verified live against a running server on a Pebble store: POST with
// profile_id "does-not-exist" returned 200 and {"succeeded":1,"failed":0},
// writing an assignment that pointed at no profile. The validation was there
// and had a test — the test just ran on the one backend where the check works.
func TestBulkRejectsUnknownProfileOnPebble(t *testing.T) {
	usePebbleProfileStore(t)

	body, err := json.Marshal(bulkProfileRequest{
		ProfileID: "does-not-exist",
		MediaIDs:  []string{"/media/Show/ep1.mkv"},
	})
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	bulkMediaProfilesHandler(nil).ServeHTTP(rr,
		httptest.NewRequest(http.MethodPost, "/api/media/profiles/bulk", bytes.NewReader(body)))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d (body: %s), want 404; an unknown profile must never be assigned",
			rr.Code, rr.Body.String())
	}
}

// TestBulkAcceptsKnownProfileOnPebble is the control: without it, a fix that
// rejected everything would look correct.
func TestBulkAcceptsKnownProfileOnPebble(t *testing.T) {
	usePebbleProfileStore(t)

	id := createTestProfile(t, "pebble-profile", "en")

	body, err := json.Marshal(bulkProfileRequest{
		ProfileID: id,
		MediaIDs:  []string{"/media/Show/ep1.mkv", "/media/Show/ep2.mkv"},
	})
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	bulkMediaProfilesHandler(nil).ServeHTTP(rr,
		httptest.NewRequest(http.MethodPost, "/api/media/profiles/bulk", bytes.NewReader(body)))

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d (body: %s), want 200", rr.Code, rr.Body.String())
	}
	var resp bulkProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Succeeded != 2 || resp.Failed != 0 {
		t.Errorf("succeeded=%d failed=%d, want 2/0", resp.Succeeded, resp.Failed)
	}
}

// TestSingleAssignRejectsUnknownProfileOnPebble covers the same divergence on
// the single-item route, which had the identical check.
func TestSingleAssignRejectsUnknownProfileOnPebble(t *testing.T) {
	usePebbleProfileStore(t)

	body := []byte(`{"profile_id":"does-not-exist"}`)
	mux, _, err := newMux(nil)
	if err != nil {
		t.Fatalf("newMux: %v", err)
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr,
		httptest.NewRequest(http.MethodPut, mediaProfileURL("/media/Show/ep1.mkv"), bytes.NewReader(body)))

	if rr.Code == http.StatusOK || rr.Code == http.StatusNoContent {
		t.Fatalf("got %d; an unknown profile must not be assigned", rr.Code)
	}
}
