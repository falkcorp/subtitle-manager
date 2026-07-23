package webserver

import (
	"errors"
	"io/fs"
	"net/http"
	"strings"
)

// spaFileServer serves static files from the given filesystem and falls back to
// index.html for unknown paths when it exists. This enables client-side routing
// for the React application.
func spaFileServer(fsys fs.FS) http.Handler {
	fsHandler := http.FileServer(http.FS(fsys))
	// Read index.html up front so unknown paths can be served the SPA shell
	// directly. (Detecting its presence also avoids breaking tests that embed no
	// frontend.)
	indexData, err := fs.ReadFile(fsys, "index.html")
	hasIndex := err == nil

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hasIndex {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path != "" {
				if _, statErr := fs.Stat(fsys, path); statErr != nil && errors.Is(statErr, fs.ErrNotExist) {
					// Unknown path → serve the SPA shell so client-side routing
					// (e.g. /library, /tools/verify) works on hard navigation.
					// We write index.html's bytes directly rather than rewriting
					// r.URL.Path to "/index.html": http.FileServer 301-redirects
					// requests for ".../index.html" to "./", which would bounce
					// every deep link back to the app root.
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.Header().Set("Cache-Control", "no-cache")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(indexData)
					return
				}
			}
		}
		fsHandler.ServeHTTP(w, r)
	})
}
