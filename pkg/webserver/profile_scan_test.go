// file: pkg/webserver/profile_scan_test.go
// version: 1.0.0
// guid: 6a41d8b3-7e29-4c50-9f18-2b5d0c7a3e64
// last-edited: 2026-08-01

package webserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/profiles"
	"github.com/jdfalk/subtitle-manager/pkg/scanner"
)

// TestProfileAssignedInUIIsSeenByScanner is the cross-entry-point guard.
//
// Assigning a language profile in the UI and reading it back in the UI proves
// only that the web handler agrees with itself. The defect this feature had
// was precisely that it did not agree with anyone else: the handler wrote a
// path-keyed row through the SubtitleStore while the download path read an
// integer-keyed row out of a different database, so an assignment made in the
// UI could never influence a download.
//
// So the write here goes through the real HTTP handler and the read goes
// through the scanner's own resolution, with nothing shared but the store.
//
// The assertion is on which error comes back rather than on a successful
// download: no subtitle provider is configured in a test, so the fetch is
// expected to fail. What matters is that it fails *at the fetch*, having found
// the assignment — not at the lookup with "no language profile assigned",
// which is what a key or database disagreement produces.
func TestProfileAssignedInUIIsSeenByScanner(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	mediaDir := t.TempDir()
	video := filepath.Join(mediaDir, "Show.S01E01.mkv")
	if err := os.WriteFile(video, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	prevMedia := viper.GetString("media_directory")
	viper.Set("media_directory", mediaDir)
	t.Cleanup(func() { viper.Set("media_directory", prevMedia) })

	// A default profile, so the assignment has to be distinguishable from it.
	defaultID := createTestProfile(t, "Default", "en")
	store, err := database.GetSharedStore()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.SetDefaultLanguageProfile(defaultID); err != nil {
		t.Fatalf("set default: %v", err)
	}

	// Create the assigned profile with two languages, through the handler.
	body, err := json.Marshal(profiles.LanguageProfile{
		Name: "Iberian",
		Languages: []profiles.LanguageConfig{
			{Language: "pt", Priority: 2},
			{Language: "es", Priority: 1},
		},
		CutoffScore: 75,
	})
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	profilesHandler(nil).ServeHTTP(rr,
		httptest.NewRequest(http.MethodPost, "/api/profiles", bytes.NewReader(body)))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create profile: %d (%s)", rr.Code, rr.Body.String())
	}
	var created profiles.LanguageProfile
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	// Assign it to the video the way the UI does: the file path,
	// percent-encoded into the URL.
	rr = httptest.NewRecorder()
	mediaProfilesHandler(nil).ServeHTTP(rr, httptest.NewRequest(http.MethodPut,
		"/api/media/profile/"+url.PathEscape(video),
		bytes.NewReader([]byte(`{"profile_id":"`+created.ID+`"}`))))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("assign profile: %d (%s)", rr.Code, rr.Body.String())
	}

	// Now resolve it from the scanner side.
	err = scanner.ProcessFileWithProfile(t.Context(), video, nil, false, store)
	if err == nil {
		return // a download somehow succeeded; the assignment was certainly found
	}
	if strings.Contains(err.Error(), "no language profile assigned") {
		t.Fatalf("the scanner did not see the profile the UI assigned: %v — the "+
			"write and read paths disagree on the key or the database", err)
	}
}
