// file: pkg/webserver/profiles_test.go
// version: 1.3.0
// guid: 1b0c9d8e-7f6e-2a3b-5c4d-8f7e9a0b1c2d
// last-edited: 2026-08-04

package webserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/database"
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

// createTestProfile posts a profile through the collection handler and returns
// its server-assigned ID.
func createTestProfile(t *testing.T, name, lang string) string {
	t.Helper()
	body, err := json.Marshal(profiles.LanguageProfile{
		Name:        name,
		Languages:   []profiles.LanguageConfig{{Language: lang, Priority: 1}},
		CutoffScore: 75,
	})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	profilesHandler(nil).ServeHTTP(rr,
		httptest.NewRequest(http.MethodPost, "/api/profiles", bytes.NewReader(body)))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create profile %q: got %d (body: %s)", name, rr.Code, rr.Body.String())
	}
	var created profiles.LanguageProfile
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("create profile %q: %v", name, err)
	}
	if created.ID == "" {
		t.Fatalf("create profile %q: server returned no ID", name)
	}
	return created.ID
}

// mediaProfileURL builds the URL the web UI builds: the media file path,
// percent-encoded whole, appended to the route.
func mediaProfileURL(mediaPath string) string {
	return "/api/media/profile/" + url.PathEscape(mediaPath)
}

// TestMediaProfilesHandlerDistinctPaths pins that two media files receive two
// independent profile assignments.
//
// They did not. The UI sends the media file path as the identifier
// (MediaDetails.jsx percent-encodes it), net/http decodes it before the
// handler runs, and the handler took only the first path segment — so every
// file under /media keyed as "media" and each assignment overwrote the last.
// Assigning a profile to one episode silently reassigned the entire library.
//
// The bug survived because every other test here uses an identifier like "123"
// or "some-id", which has no slash to split on. Only a slash-bearing path
// exercises it.
func TestMediaProfilesHandlerDistinctPaths(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	english := createTestProfile(t, "English", "en")
	spanish := createTestProfile(t, "Spanish", "es")

	const (
		ep1 = "/media/Show/Show.S01E01.mkv"
		ep2 = "/media/Show/Show.S01E02.mkv"
	)

	handler := mediaProfilesHandler(nil)

	assign := func(mediaPath, profileID string) {
		t.Helper()
		body := bytes.NewReader([]byte(`{"profile_id":"` + profileID + `"}`))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPut, mediaProfileURL(mediaPath), body))
		if rr.Code != http.StatusNoContent {
			t.Fatalf("assign %s: got %d (body: %s)", mediaPath, rr.Code, rr.Body.String())
		}
	}

	assigned := func(mediaPath string) profiles.LanguageProfile {
		t.Helper()
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, mediaProfileURL(mediaPath), nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("get %s: got %d (body: %s)", mediaPath, rr.Code, rr.Body.String())
		}
		var got profiles.LanguageProfile
		if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
			t.Fatalf("get %s: %v (body: %s)", mediaPath, err, rr.Body.String())
		}
		return got
	}

	assign(ep1, english)
	assign(ep2, spanish)

	if got := assigned(ep1); got.ID != english {
		t.Errorf("%s resolved to profile %q (%s), want English (%s) — the second "+
			"assignment overwrote the first, so both files share one key",
			ep1, got.Name, got.ID, english)
	}
	if got := assigned(ep2); got.ID != spanish {
		t.Errorf("%s resolved to profile %q (%s), want Spanish (%s)",
			ep2, got.Name, got.ID, spanish)
	}
}

// TestMediaProfilesHandlerKeyMatchesCLI pins that the key the handler stores is
// the media path itself, normalised the way the CLI normalises it, so a UI
// assignment is visible to `subtitle-manager profiles show <path>` and vice
// versa. Storing a raw, uncleaned path would be invisible to the CLI, which
// looks up security.ValidateAndSanitizePath's cleaned output.
func TestMediaProfilesHandlerKeyMatchesCLI(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	english := createTestProfile(t, "English", "en")
	handler := mediaProfilesHandler(nil)

	// Assign through an uncleaned spelling of the path...
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPut,
		mediaProfileURL("/media/Show/../Show/Show.S01E01.mkv"),
		bytes.NewReader([]byte(`{"profile_id":"`+english+`"}`))))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("assign: got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// ...and read it back through the cleaned one the CLI would use.
	store, err := database.GetSharedStore()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	got, err := store.GetMediaProfile("/media/Show/Show.S01E01.mkv")
	if err != nil {
		t.Fatalf("lookup by cleaned path: %v", err)
	}
	if got.ID != english {
		t.Errorf("cleaned path resolved to profile %q (%s), want English (%s) — "+
			"the handler stored an unnormalised key the CLI cannot find",
			got.Name, got.ID, english)
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

// deleteProfile issues DELETE through the item handler and returns the status.
func deleteProfile(t *testing.T, id string) (int, string) {
	t.Helper()
	rr := httptest.NewRecorder()
	profilesHandler(nil).ServeHTTP(rr,
		httptest.NewRequest(http.MethodDelete, "/api/profiles/"+id, nil))
	return rr.Code, rr.Body.String()
}

// profileIDs lists the IDs currently stored, through the collection handler.
func profileIDs(t *testing.T) []string {
	t.Helper()
	rr := httptest.NewRecorder()
	profilesHandler(nil).ServeHTTP(rr,
		httptest.NewRequest(http.MethodGet, "/api/profiles", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("list profiles: got %d (body: %s)", rr.Code, rr.Body.String())
	}
	var list []profiles.LanguageProfile
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	ids := make([]string, 0, len(list))
	for _, p := range list {
		ids = append(ids, p.ID)
	}
	return ids
}

// TestLastProfileIsDeletable pins that the profile list can be emptied.
//
// The delete guard asked GetDefaultLanguageProfile whether the target was the
// default. That helper answers "which profile governs by default" and falls
// back to the *first* profile when none is flagged — so the last remaining
// profile always came back as the default and was always refused. There was no
// sequence of requests that emptied the list.
func TestLastProfileIsDeletable(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	for _, id := range profileIDs(t) {
		if code, body := deleteProfile(t, id); code != http.StatusNoContent {
			t.Fatalf("delete seeded profile %s: got %d (body: %s)", id, code, body)
		}
	}

	if ids := profileIDs(t); len(ids) != 0 {
		t.Errorf("profiles remain after deleting every one: %v", ids)
	}
}

// TestNonDefaultProfileIsDeletableWhenNothingIsFlagged is the other half of the
// same bug. With two profiles and no is_default set, GetDefaultLanguageProfile
// returned whichever came first, making that arbitrary profile permanently
// undeletable even though the user never chose it as the default.
func TestNonDefaultProfileIsDeletableWhenNothingIsFlagged(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	for _, id := range profileIDs(t) {
		if code, body := deleteProfile(t, id); code != http.StatusNoContent {
			t.Fatalf("clear seeded profile %s: got %d (body: %s)", id, code, body)
		}
	}
	first := createTestProfile(t, "first", "en")
	createTestProfile(t, "second", "fr")

	if code, body := deleteProfile(t, first); code != http.StatusNoContent {
		t.Fatalf("delete unflagged profile: got %d (body: %s)", code, body)
	}
}

// TestDefaultProfileStillProtectedWhenOthersExist pins what the guard is
// actually for: while alternatives exist, removing the default would leave
// scans without a fallback, so it stays refused.
func TestDefaultProfileStillProtectedWhenOthersExist(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	for _, id := range profileIDs(t) {
		if code, body := deleteProfile(t, id); code != http.StatusNoContent {
			t.Fatalf("clear seeded profile %s: got %d (body: %s)", id, code, body)
		}
	}
	keep := createTestProfile(t, "keep", "en")
	createTestProfile(t, "other", "fr")

	rr := httptest.NewRecorder()
	profilesHandler(nil).ServeHTTP(rr,
		httptest.NewRequest(http.MethodPost, "/api/profiles/"+keep+"/default", nil))
	if rr.Code != http.StatusOK && rr.Code != http.StatusNoContent {
		t.Fatalf("set default: got %d (body: %s)", rr.Code, rr.Body.String())
	}

	if code, _ := deleteProfile(t, keep); code != http.StatusBadRequest {
		t.Errorf("deleting the default with another profile present: got %d, want 400", code)
	}
}
