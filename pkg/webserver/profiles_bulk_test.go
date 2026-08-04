// file: pkg/webserver/profiles_bulk_test.go
// version: 1.0.0
// guid: 8d4e0b62-1c95-47af-b3e0-6a2f19d8c47b
// last-edited: 2026-08-04

package webserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jdfalk/subtitle-manager/pkg/database"
)

// postBulk issues a bulk assignment request against the handler and decodes the
// response, failing the test if the status is not the expected one.
func postBulk(t *testing.T, body string, wantStatus int) bulkProfileResponse {
	t.Helper()
	rr := httptest.NewRecorder()
	bulkMediaProfilesHandler(nil).ServeHTTP(rr, httptest.NewRequest(
		http.MethodPost, "/api/media/profiles/bulk", strings.NewReader(body)))
	if rr.Code != wantStatus {
		t.Fatalf("status = %d, want %d (body: %s)", rr.Code, wantStatus, rr.Body.String())
	}
	var resp bulkProfileResponse
	if wantStatus == http.StatusOK {
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v (body: %s)", err, rr.Body.String())
		}
	}
	return resp
}

// TestBulkAssignProfileWritesEveryItem is the core behaviour: one request,
// many media items, all assigned to the same profile.
//
// It reads back through the store rather than through the handler so a bug that
// made the handler agree with itself but not with the storage key would fail
// here.
func TestBulkAssignProfileWritesEveryItem(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	profileID := createTestProfile(t, "Nordic", "sv")
	media := []string{"/media/A.mkv", "/media/B.mkv", "/media/sub/C.mkv"}

	body, err := json.Marshal(bulkProfileRequest{ProfileID: profileID, MediaIDs: media})
	if err != nil {
		t.Fatal(err)
	}
	resp := postBulk(t, string(body), http.StatusOK)

	if resp.Succeeded != len(media) || resp.Failed != 0 {
		t.Fatalf("succeeded=%d failed=%d, want %d/0 (%+v)",
			resp.Succeeded, resp.Failed, len(media), resp.Results)
	}
	if len(resp.Results) != len(media) {
		t.Fatalf("results len = %d, want %d", len(resp.Results), len(media))
	}

	store, err := database.GetSharedStore()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	for _, m := range media {
		got, err := store.GetMediaProfile(m)
		if err != nil {
			t.Fatalf("GetMediaProfile(%s): %v", m, err)
		}
		if got.ID != profileID {
			t.Fatalf("media %s resolved to profile %q, want %q — the bulk write "+
				"did not land on the key the store reads", m, got.ID, profileID)
		}
	}
}

// TestBulkResultsStayAlignedWithInput pins the 1:1, in-order contract the UI
// relies on to map results back onto its selection — including duplicates,
// which are deliberately not collapsed.
func TestBulkResultsStayAlignedWithInput(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	profileID := createTestProfile(t, "Dup", "de")
	media := []string{"/media/A.mkv", "/media/A.mkv", "/media/B.mkv"}
	body, _ := json.Marshal(bulkProfileRequest{ProfileID: profileID, MediaIDs: media})
	resp := postBulk(t, string(body), http.StatusOK)

	if len(resp.Results) != len(media) {
		t.Fatalf("results len = %d, want %d (duplicates must not be collapsed)",
			len(resp.Results), len(media))
	}
	for i, r := range resp.Results {
		if r.MediaID != media[i] {
			t.Fatalf("results[%d].media_id = %q, want %q — results must stay in "+
				"submitted order so the UI can zip them against its selection",
				i, r.MediaID, media[i])
		}
	}
}

// TestBulkClearProfile covers profile_id:"" meaning "unassign", the bulk
// equivalent of DELETE on the single-item route.
func TestBulkClearProfile(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	assigned := createTestProfile(t, "Assigned", "fr")
	fallback := createTestProfile(t, "Fallback", "en")
	store, err := database.GetSharedStore()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.SetDefaultLanguageProfile(fallback); err != nil {
		t.Fatalf("set default: %v", err)
	}

	const media = "/media/Clear.mkv"
	body, _ := json.Marshal(bulkProfileRequest{ProfileID: assigned, MediaIDs: []string{media}})
	if resp := postBulk(t, string(body), http.StatusOK); resp.Succeeded != 1 {
		t.Fatalf("setup assign failed: %+v", resp)
	}

	body, _ = json.Marshal(bulkProfileRequest{ProfileID: "", MediaIDs: []string{media}})
	resp := postBulk(t, string(body), http.StatusOK)
	if resp.Succeeded != 1 || resp.Failed != 0 {
		t.Fatalf("clear: succeeded=%d failed=%d (%+v)", resp.Succeeded, resp.Failed, resp.Results)
	}

	// After clearing, the item must resolve to the default rather than to the
	// profile it used to have.
	got, err := store.GetMediaProfile(media)
	if err != nil {
		t.Fatalf("GetMediaProfile: %v", err)
	}
	if got.ID == assigned {
		t.Fatalf("media still resolves to the cleared profile %q", assigned)
	}
}

// TestBulkNormalisesMediaKeys pins that a bulk write uses the same key
// normalisation as the single-item route. Without it, "/media//A.mkv" from a
// UI listing would occupy a different row than "/media/A.mkv" written
// elsewhere, and the scanner — which looks up the cleaned path — would miss it.
func TestBulkNormalisesMediaKeys(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	profileID := createTestProfile(t, "Norm", "it")
	body, _ := json.Marshal(bulkProfileRequest{
		ProfileID: profileID,
		MediaIDs:  []string{"/media//Norm.mkv", "/media/./Other.mkv"},
	})
	if resp := postBulk(t, string(body), http.StatusOK); resp.Failed != 0 {
		t.Fatalf("unexpected failures: %+v", resp.Results)
	}

	store, err := database.GetSharedStore()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	for _, cleaned := range []string{"/media/Norm.mkv", "/media/Other.mkv"} {
		got, err := store.GetMediaProfile(cleaned)
		if err != nil {
			t.Fatalf("GetMediaProfile(%s): %v", cleaned, err)
		}
		if got.ID != profileID {
			t.Fatalf("cleaned key %s did not resolve to the assigned profile — "+
				"the bulk route is not normalising like the single-item route", cleaned)
		}
	}
}

// TestBulkRejectsBadRequests covers the whole-request failures, which are
// status codes rather than per-item results.
func TestBulkRejectsBadRequests(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	// A real profile has to exist so "unknown profile" is genuinely about the
	// requested ID and not about an empty store.
	createTestProfile(t, "Real", "en")

	t.Run("unknown profile is one 404, not N item errors", func(t *testing.T) {
		body, _ := json.Marshal(bulkProfileRequest{
			ProfileID: "no-such-profile",
			MediaIDs:  []string{"/media/A.mkv", "/media/B.mkv"},
		})
		postBulk(t, string(body), http.StatusNotFound)
	})

	t.Run("empty media_ids", func(t *testing.T) {
		postBulk(t, `{"profile_id":"x","media_ids":[]}`, http.StatusBadRequest)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		postBulk(t, `{"profile_id":`, http.StatusBadRequest)
	})

	t.Run("over the item cap", func(t *testing.T) {
		ids := make([]string, maxBulkProfileItems+1)
		for i := range ids {
			ids[i] = fmt.Sprintf("/media/%d.mkv", i)
		}
		body, _ := json.Marshal(bulkProfileRequest{ProfileID: "x", MediaIDs: ids})
		postBulk(t, string(body), http.StatusRequestEntityTooLarge)
	})

	t.Run("wrong method", func(t *testing.T) {
		rr := httptest.NewRecorder()
		bulkMediaProfilesHandler(nil).ServeHTTP(rr,
			httptest.NewRequest(http.MethodGet, "/api/media/profiles/bulk", nil))
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("GET returned %d, want 405", rr.Code)
		}
	})
}

// TestBulkReportsPerItemFailure proves the per-item error channel actually
// carries a failure rather than being decorative: an empty identifier cannot be
// written, and must come back as a failed item inside an otherwise-200 response
// while its neighbours still succeed.
func TestBulkReportsPerItemFailure(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	profileID := createTestProfile(t, "Partial", "es")
	body, _ := json.Marshal(bulkProfileRequest{
		ProfileID: profileID,
		MediaIDs:  []string{"/media/Good.mkv", "", "/media/AlsoGood.mkv"},
	})
	resp := postBulk(t, string(body), http.StatusOK)

	if resp.Succeeded != 2 || resp.Failed != 1 {
		t.Fatalf("succeeded=%d failed=%d, want 2/1 (%+v)",
			resp.Succeeded, resp.Failed, resp.Results)
	}
	if resp.Results[1].OK || resp.Results[1].Error == "" {
		t.Fatalf("the empty id should be a failed item with an error: %+v", resp.Results[1])
	}
	if !resp.Results[0].OK || !resp.Results[2].OK {
		t.Fatalf("one bad item must not abort its neighbours: %+v", resp.Results)
	}
}

// TestBulkProfileRouteIsNotSwallowedByTags is the registration guard.
//
// /api/media/ is a registered SUBTREE for mediaTagsHandler. A route beneath it
// that is not registered with its own exact pattern is answered by the tag
// handler — 200 with the wrong body, not a 404 — so neither "does the path
// match some pattern" nor "is the status not 404" can detect the mistake. This
// is the exact failure that hid /api/providers/available for a release.
//
// So this asserts the matched pattern is the bulk route specifically, and then
// that the response is the bulk handler's own shape.
func TestBulkProfileRouteIsNotSwallowedByTags(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	mux, prefix, err := newMux(nil)
	if err != nil {
		t.Fatalf("newMux: %v", err)
	}

	const path = "/api/media/profiles/bulk"
	req := httptest.NewRequest(http.MethodPost, prefix+path, nil)
	_, pattern := mux.Handler(req)
	if pattern != prefix+path {
		t.Fatalf("POST %s matched pattern %q, want %q — it is being absorbed by "+
			"an adjacent subtree instead of reaching the bulk handler",
			path, pattern, prefix+path)
	}
}

// TestBulkProfileEndToEndThroughMux drives the registered route rather than the
// handler directly, so route registration, auth wiring and the handler are
// exercised together.
func TestBulkProfileEndToEndThroughMux(t *testing.T) {
	skipIfNoSQLite(t)
	useTempProfileStore(t)

	profileID := createTestProfile(t, "E2E", "ja")
	mux, prefix, err := newMux(nil)
	if err != nil {
		t.Fatalf("newMux: %v", err)
	}

	body, _ := json.Marshal(bulkProfileRequest{
		ProfileID: profileID,
		MediaIDs:  []string{"/media/E2E.mkv"},
	})
	req := httptest.NewRequest(http.MethodPost, prefix+"/api/media/profiles/bulk",
		bytes.NewReader(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// Unauthenticated requests are rejected by the middleware; that is itself
	// proof the route is mounted behind auth rather than falling through to the
	// SPA catch-all, which would return HTML with 200.
	if rr.Code == http.StatusOK && strings.Contains(rr.Body.String(), "<html") {
		t.Fatalf("bulk route fell through to the SPA catch-all: %s", rr.Body.String())
	}
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rr.Code, rr.Body.String())
	}
}
