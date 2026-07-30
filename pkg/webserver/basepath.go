// file: pkg/webserver/basepath.go
// version: 1.0.0
// guid: eb46ab64-f28e-4adb-9677-ac932a044fdd
// last-edited: 2026-07-30

package webserver

import (
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

// basePrefix returns the configured base-URL prefix, normalised the same way
// newMux normalises it when mounting routes: either empty, or a leading slash
// with no trailing one.
//
// It is derived from configuration on each call rather than captured at mount
// time because the handlers that need it are constructed without access to the
// mux. base_url is not reloadable at runtime, so re-reading it is a lookup, not
// a correctness risk.
func basePrefix() string {
	prefix := strings.Trim(viper.GetString("base_url"), "/")
	if prefix == "" {
		return ""
	}
	return "/" + prefix
}

// apiPath returns the request path with the base-URL prefix removed, so a
// handler can match against the literal route it was mounted for.
//
// # Why this exists
//
// Routes are mounted at prefix+"/api/...", where prefix comes from base_url.
// Handlers that pull an ID out of the path were written as if the prefix were
// always empty:
//
//	strings.TrimPrefix(r.URL.Path, "/api/media/profile/")
//
// Behind a base URL the incoming path is "/sm/api/media/profile/42", which
// that literal does not match. TrimPrefix returns the string unchanged, the
// subsequent Split yields "" as the first segment, and the handler proceeds
// with an empty ID — so it answers "not found" or "bad request" for every
// request. The feature is not degraded behind a subpath, it is dead.
//
// Nothing caught this because every test constructs requests with an empty
// prefix, where the literal happens to match. Seven handlers had the same
// defect: media profiles, profile lookup, users, tags, tag-to-user and
// tag-to-media assignment.
func apiPath(r *http.Request) string {
	p := r.URL.Path
	prefix := basePrefix()
	if prefix == "" {
		return p
	}
	// Only strip a prefix that is a whole path segment: "/sm" must match
	// "/sm/api/..." but not "/small/api/...".
	if p == prefix {
		return "/"
	}
	if strings.HasPrefix(p, prefix+"/") {
		return strings.TrimPrefix(p, prefix)
	}
	return p
}

// pathSegments returns the path segments following route, with the base-URL
// prefix already removed. It returns nil when the request path does not lie
// under route.
//
// Handlers used to inline strings.Split(strings.TrimPrefix(r.URL.Path, ...)),
// which silently produced a slice whose first element was "" whenever the
// literal did not match — an empty ID that then flowed into a database lookup
// instead of being rejected. Returning nil makes "this path is not what I
// expected" a case the handler has to handle, rather than something it
// discovers as a mysterious miss further down.
func pathSegments(r *http.Request, route string) []string {
	p := apiPath(r)
	if !strings.HasPrefix(p, route) {
		return nil
	}
	rest := strings.Trim(strings.TrimPrefix(p, route), "/")
	if rest == "" {
		return nil
	}
	return strings.Split(rest, "/")
}

// pathSegment returns the first path segment following route, or "" when the
// path does not lie under route or carries no further segment.
func pathSegment(r *http.Request, route string) string {
	segs := pathSegments(r, route)
	if len(segs) == 0 {
		return ""
	}
	return segs[0]
}
