// file: pkg/webserver/basepath_test.go
// version: 1.0.0
// guid: d58f16de-9d4f-4e6b-8a95-8a3d53f5dbcd
// last-edited: 2026-07-30

package webserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// useBaseURL sets base_url for one test and restores it afterwards.
func useBaseURL(t *testing.T, v string) {
	t.Helper()
	prev := viper.Get("base_url")
	viper.Set("base_url", v)
	t.Cleanup(func() { viper.Set("base_url", prev) })
}

// TestAPIPath covers prefix stripping, including the boundary cases that make
// a naive strings.TrimPrefix wrong.
func TestAPIPath(t *testing.T) {
	for name, tc := range map[string]struct {
		base, path, want string
	}{
		"no base url is a passthrough": {
			"", "/api/media/profile/42", "/api/media/profile/42"},
		"base url is stripped": {
			"/sm", "/sm/api/media/profile/42", "/api/media/profile/42"},
		"base url without slashes is normalised": {
			"sm", "/sm/api/tags/7", "/api/tags/7"},
		"trailing slash on the config value is tolerated": {
			"/sm/", "/sm/api/users/3", "/api/users/3"},
		// Only whole segments may match: "/sm" must not strip "/small".
		"a longer sibling segment is not stripped": {
			"/sm", "/small/api/tags/7", "/small/api/tags/7"},
		"bare prefix becomes root": {
			"/sm", "/sm", "/"},
		"unrelated path is untouched": {
			"/sm", "/api/tags/7", "/api/tags/7"},
		"nested base url": {
			"/a/b", "/a/b/api/tags/7", "/api/tags/7"},
	} {
		t.Run(name, func(t *testing.T) {
			useBaseURL(t, tc.base)
			r := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if got := apiPath(r); got != tc.want {
				t.Errorf("apiPath(%q) with base_url %q = %q, want %q",
					tc.path, tc.base, got, tc.want)
			}
		})
	}
}

// TestPathSegment covers ID extraction, which is where the base_url defect
// actually bit.
func TestPathSegment(t *testing.T) {
	for name, tc := range map[string]struct {
		base, path, route, want string
	}{
		"no base url": {
			"", "/api/media/profile/42", "/api/media/profile/", "42"},
		"under a base url": {
			"/sm", "/sm/api/media/profile/42", "/api/media/profile/", "42"},
		"extra segments are ignored": {
			"/sm", "/sm/api/media/profile/42/extra", "/api/media/profile/", "42"},
		"trailing slash means no id": {
			"/sm", "/sm/api/media/profile/", "/api/media/profile/", ""},
		"missing id": {
			"", "/api/media/profile/", "/api/media/profile/", ""},
		"path outside the route": {
			"/sm", "/sm/api/other/42", "/api/media/profile/", ""},
	} {
		t.Run(name, func(t *testing.T) {
			useBaseURL(t, tc.base)
			r := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if got := pathSegment(r, tc.route); got != tc.want {
				t.Errorf("pathSegment(%q) with base_url %q = %q, want %q",
					tc.path, tc.base, got, tc.want)
			}
		})
	}
}

// TestMediaProfileRejectsMissingIDUnderBaseURL is the end-to-end guard.
//
// It asserts the *rejection* path rather than the success path, because that
// is what can be observed without a database — and it is a genuine
// discriminator. The old code did
// strings.TrimPrefix(r.URL.Path, "/api/media/profile/") and checked the result
// for emptiness. Behind a base URL the path is "/sm/api/media/profile/", the
// literal does not match, TrimPrefix returns the whole path, the emptiness
// check passes, and the handler proceeds into a database lookup with an ID of
// "" instead of rejecting the request.
//
// So under base_url the broken version does NOT return 400 here, and the fixed
// one does. Verified by reverting the fix and watching this fail.
func TestMediaProfileRejectsMissingIDUnderBaseURL(t *testing.T) {
	useBaseURL(t, "/sm")

	rr := httptest.NewRecorder()
	mediaProfilesHandler(nil).ServeHTTP(rr,
		httptest.NewRequest(http.MethodGet, "/sm/api/media/profile/", nil))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d for a path carrying no media ID, want 400: the path was "+
			"not parsed relative to base_url, so an empty ID reached the lookup "+
			"(body %q)", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
}

// TestNoRawAPIPathParsing is a source-level guard against this defect
// reappearing.
//
// The bug is invisible in review and in tests: every test builds requests with
// an empty base_url, where the literal prefix happens to match, so a handler
// that strips r.URL.Path directly looks correct everywhere except on a real
// deployment behind a subpath. Seven handlers had drifted into it. Rather than
// rely on catching the eighth by eye, this fails the build for any new one.
//
// If a handler genuinely needs the raw path, it should say so explicitly
// rather than reaching for r.URL.Path with an /api literal.
func TestNoRawAPIPathParsing(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	// Matches TrimPrefix/HasPrefix/Split against r.URL.Path with an "/api"
	// literal — the shape that ignores base_url.
	bad := regexp.MustCompile(`(TrimPrefix|HasPrefix|TrimSuffix)\(\s*r\.URL\.Path\s*,\s*"/api`)

	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == "basepath.go" {
			continue // basepath.go documents the anti-pattern in a comment
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if bad.MatchString(line) {
				t.Errorf("%s:%d parses r.URL.Path against an /api literal, which "+
					"ignores base_url and makes the route dead behind a subpath. "+
					"Use apiPath/pathSegment/pathSegments instead:\n\t%s",
					f, i+1, strings.TrimSpace(line))
			}
		}
	}
}
