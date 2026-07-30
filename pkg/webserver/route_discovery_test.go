// file: pkg/webserver/route_discovery_test.go
// version: 1.0.0
// guid: 9c4b7e21-58f3-4a06-b9d7-2e150c8a4f63
// last-edited: 2026-07-30

package webserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestDiscoveredFrontendPathsAreMounted scans the web UI source for the API
// paths it calls and asserts each one resolves to a real handler.
//
// # Why this exists alongside route_coverage_test.go
//
// That file checks a hand-maintained list, which only ever catches paths
// somebody remembered to add. This one derives the list from the frontend, so a
// new call to an endpoint that was never mounted fails here without anyone
// updating anything.
//
// # What this cannot catch, and why that matters
//
// It asserts a path matches *some* pattern, not the *right* one. That is a real
// gap, and /api/providers/available is the example: "/api/providers/" is
// registered as a subtree for universalTagsHandler, so before that endpoint
// existed the path resolved happily — to the tag handler. Both this guard and
// the curated one would have called it mounted.
//
// A path swallowed by a neighbouring subtree therefore still needs to be caught
// by a test of the endpoint's actual behaviour. What this guard does cover is
// the more common case: a path with no matching pattern at all, which the SPA
// catch-all answers with index.html and a 200.
func TestDiscoveredFrontendPathsAreMounted(t *testing.T) {
	root := filepath.Join("..", "..", "webui", "src")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("web UI source not present: %v", err)
	}

	paths, err := discoverAPIPaths(root)
	if err != nil {
		t.Fatalf("scan web UI: %v", err)
	}
	if len(paths) < 10 {
		t.Fatalf("only discovered %d API paths (%v); the scanner is probably "+
			"broken and this guard would pass vacuously", len(paths), paths)
	}

	mux, prefix, err := newMux(nil)
	if err != nil {
		t.Fatalf("newMux: %v", err)
	}
	catchAll := prefix + "/"

	for _, p := range paths {
		if _, skip := knownMissingAPIPaths[p]; skip {
			continue
		}
		req := httptest.NewRequest(http.MethodGet, prefix+p, nil)
		_, pattern := mux.Handler(req)
		if pattern == catchAll || pattern == "" {
			t.Errorf("%s is called by the web UI but matches no route (pattern %q). "+
				"It will return index.html with a 200 from the SPA catch-all, so the "+
				"caller sees a success and then fails parsing HTML as JSON. Mount it, "+
				"or record it in knownMissingAPIPaths with the reason.", p, pattern)
		}
	}
}

// apiPathPattern matches a string literal beginning with /api/ in the web UI
// source, whether passed to fetch, apiFetch or apiService.
var apiPathPattern = regexp.MustCompile(`['"` + "`" + `](/api/[a-zA-Z0-9._/-]*)`)

// discoverAPIPaths collects the distinct /api/ paths referenced under root.
//
// Template literals are truncated at the first interpolation — "/api/tags/${id}"
// yields "/api/tags/", which is enough to tell a mounted subtree from an
// unmounted one, and avoids inventing IDs the handler might reject.
func discoverAPIPaths(root string) ([]string, error) {
	seen := map[string]bool{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Tests describe paths they mock, which are not necessarily paths
			// the application calls.
			if info.Name() == "__tests__" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if ext := filepath.Ext(path); ext != ".js" && ext != ".jsx" {
			return nil
		}
		// mockApi.js is a development double that prefix-matches paths
		// (`urlPath.includes('/api/library')`). Those are not endpoints the
		// application calls, and treating them as such reports routes that do
		// not need to exist.
		if strings.Contains(filepath.Base(path), "mockApi") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range apiPathPattern.FindAllStringSubmatch(string(src), -1) {
			p := strings.TrimSuffix(m[1], "/")
			if p == "" || p == "/api" {
				continue
			}
			seen[p] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

// TestAvailableProvidersReturnsAList asserts the endpoint's behaviour, not just
// that it routes somewhere.
//
// Routing alone cannot express this. Before the endpoint existed,
// /api/providers/available resolved to the "/api/providers/" subtree — the
// universal tag handler — so every pattern-based check called it mounted while
// the UI received a shape it could not use and crashed on
// "availableProviders.map is not a function". The property that actually
// matters is that a JSON array comes back.
func TestAvailableProvidersReturnsAList(t *testing.T) {
	rr := httptest.NewRecorder()
	availableProvidersHandler().ServeHTTP(rr,
		httptest.NewRequest(http.MethodGet, "/api/providers/available", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var got []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not a JSON array (%v); the dialog does "+
			"availableProviders.map over this and would crash: %s",
			err, rr.Body.String())
	}
	if len(got) == 0 {
		t.Error("no providers returned; the add-provider dialog would be empty")
	}
	if _, ok := got[0]["name"]; !ok {
		t.Errorf("entries lack a name field: %v", got[0])
	}
}
