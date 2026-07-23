// file: pkg/webserver/spa_test.go
// version: 1.0.0
// guid: 98a47e42-ba7f-4823-962f-09688de989bd

package webserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// TestSpaFileServerServesShellForClientRoutes verifies unknown paths (client
// routes like /library, /tools/verify) get index.html with a 200 — not a
// redirect. Regression test: the previous implementation rewrote unknown paths
// to /index.html, which http.FileServer 301-redirects to "./", bouncing every
// deep link back to the app root so no page but the dashboard ever rendered.
func TestSpaFileServerServesShellForClientRoutes(t *testing.T) {
	const shell = "<!doctype html><title>app</title>"
	fsys := fstest.MapFS{
		"index.html":    {Data: []byte(shell)},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
	h := spaFileServer(fsys)

	for _, route := range []string{"/library", "/tools/verify", "/settings/auth"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, route, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d (Location=%q)", route, rec.Code, rec.Header().Get("Location"))
		}
		if !strings.Contains(rec.Body.String(), "<title>app</title>") {
			t.Fatalf("%s: expected SPA shell, got %q", route, rec.Body.String())
		}
	}
}

// TestSpaFileServerServesRealAssets verifies real files are still served
// normally (not replaced by the shell).
func TestSpaFileServerServesRealAssets(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
	h := spaFileServer(fsys)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "console.log(1)") {
		t.Fatalf("asset not served: code=%d body=%q", rec.Code, rec.Body.String())
	}
}
