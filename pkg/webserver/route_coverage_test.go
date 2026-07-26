// file: pkg/webserver/route_coverage_test.go
// version: 1.2.0
// guid: 6b2c9d41-8f37-4a52-9c60-1d4e7a8b3f05
// last-edited: 2026-07-26

package webserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// frontendAPIPaths lists API paths the web UI calls. Each must be served by a
// real handler.
//
// This guards a whole class of bug rather than one endpoint. The mux registers
// a catch-all at "/" that serves the SPA's index.html, so an unmounted API
// route does not 404 — it returns 200 with an HTML body. The frontend's
// `if (res.ok)` check passes, the following res.json() throws into a catch
// block that only console.errors, and the page renders empty with no
// server-side trace. Language profiles shipped that way: profilesHandler was
// written and unit-tested, but never mounted, so the entire Languages settings
// page was inert.
//
// Unit-testing a handler in isolation cannot catch this. The assertion has to
// run through the real mux.
var frontendAPIPaths = []string{
	"/api/profiles",
	"/api/profiles/some-id",
	"/api/language-profiles",
	"/api/language-profiles/some-id",
	"/api/media/profile/some-id",
	"/api/config",
	"/api/history",
	"/api/system",
	"/api/library/browse",
	"/api/providers/status",
	"/api/wanted",
	"/api/library/rescan-all",
}

// knownMissingAPIPaths are paths the web UI calls that have no backend handler
// at all — not merely unmounted, unwritten.
//
// They are listed rather than omitted so this file records the real state of
// the contract: frontendAPIPaths above is a curated list of routes that should
// pass, and without this second list a green run would read as "the frontend's
// API surface is covered" when it means "these ten are". Each entry is skipped
// with its caller; delete the entry when the handler lands.
var knownMissingAPIPaths = map[string]string{
	// Not merely unmounted — there is no caller either. MediaLibrary.jsx defines
	// handleBulkOperation and toggleFileSelection, but nothing in the component
	// references them: there is no selection checkbox and no bulk toolbar. So
	// this is dead frontend code rather than a working feature missing its
	// backend, and writing a handler would mean inventing a request contract no
	// UI actually sends. Build the selection UI and the endpoint together, or
	// delete the dead functions.
	"/api/bulk-operation": "MediaLibrary.jsx — bulk media operations; no UI invokes it",
	// NotificationSettings.jsx calls /api/notifications/test/{type}. Held back
	// deliberately: the settings page saves through POST /api/config, which
	// viper.Set()s flat keys ("discord_webhook"), while the runtime reads
	// namespaced ones ("notifications.discord_webhook"). A test button on top of
	// that would report "not configured" straight after a successful save, or
	// worse, report success for a channel that never fires in production. The
	// key bridge and the endpoint have to land together.
	"/api/notifications/test": "NotificationSettings.jsx — blocked on the config key namespace mismatch",
}

// TestFrontendAPIPathsAreMounted verifies no frontend-called API path falls
// through to the SPA catch-all.
//
// The assertion is on the mux pattern rather than the response. A mounted
// handler may legitimately answer 400/404/405/500 depending on the ID and
// storage backend, so no status code distinguishes "missing route" from
// "handler said no" — but the matched pattern does, exactly.
// A nil *sql.DB is deliberate, and load-bearing: route registration does not
// touch the database, so the test runs without the sqlite build tag. The rest
// of this package's tests call skipIfNoSQLite and therefore skip entirely in
// CI, which builds without that tag — a guard that skips where it is needed
// most is not a guard.
func TestFrontendAPIPathsAreMounted(t *testing.T) {
	mux, prefix, err := newMux(nil)
	if err != nil {
		t.Fatalf("newMux: %v", err)
	}
	catchAll := prefix + "/"

	assertMounted := func(t *testing.T, p string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, prefix+p, nil)
		if _, pattern := mux.Handler(req); pattern == catchAll {
			t.Errorf("%s matched the SPA catch-all %q instead of an API route; "+
				"the web UI calls this path, so it returns index.html with a 200 "+
				"and the page silently renders empty", p, catchAll)
		}
	}

	for _, p := range frontendAPIPaths {
		t.Run(p, func(t *testing.T) { assertMounted(t, p) })
	}

	for p, why := range knownMissingAPIPaths {
		t.Run("missing"+p, func(t *testing.T) {
			t.Skipf("no handler exists: %s", why)
		})
	}
}

// TestRewritePrefix verifies the language-profiles alias maps onto the
// /api/profiles paths the handlers parse, and leaves other paths untouched.
func TestRewritePrefix(t *testing.T) {
	var got string
	h := rewritePrefix("/api/language-profiles", "/api/profiles",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = r.URL.Path
		}))

	for _, tc := range []struct{ in, want string }{
		{"/api/language-profiles", "/api/profiles"},
		{"/api/language-profiles/", "/api/profiles/"},
		{"/api/language-profiles/abc", "/api/profiles/abc"},
		{"/api/language-profiles/abc/default", "/api/profiles/abc/default"},
		// Non-matching paths pass through unchanged.
		{"/api/profiles/abc", "/api/profiles/abc"},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.in, nil)
		h.ServeHTTP(httptest.NewRecorder(), req)
		if got != tc.want {
			t.Errorf("rewritePrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
		// The original request must not be mutated; middleware up the chain
		// still sees the path the client actually requested.
		if req.URL.Path != tc.in {
			t.Errorf("rewritePrefix mutated the original request: %q became %q",
				tc.in, req.URL.Path)
		}
	}
}
