// file: pkg/webserver/librarypaths.go
// version: 1.0.0
// guid: 17206a5a-f1bc-4f44-acfa-aa3d7f72f59c

package webserver

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/metadata"
	"github.com/jdfalk/subtitle-manager/pkg/security"
)

// libraryPathsMu guards the on-disk library-paths list.
var libraryPathsMu sync.Mutex

// libraryPathsFile returns the JSON file used to persist the configured library
// root paths. It lives alongside the database so it survives restarts.
func libraryPathsFile() string {
	dbPath := viper.GetString("db_path")
	dir := dbPath
	if fi, err := os.Stat(dbPath); err == nil && !fi.IsDir() {
		dir = filepath.Dir(dbPath)
	}
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, "library-paths.json")
}

func readLibraryPaths() []string {
	data, err := os.ReadFile(libraryPathsFile())
	if err != nil {
		return []string{}
	}
	var paths []string
	if err := json.Unmarshal(data, &paths); err != nil || paths == nil {
		return []string{}
	}
	return paths
}

func writeLibraryPaths(paths []string) error {
	data, err := json.MarshalIndent(paths, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(libraryPathsFile()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(libraryPathsFile(), data, 0o644)
}

// scanPathAsync scans dir into the media library in the background so newly
// added / rescanned paths populate MediaItems without blocking the request.
func scanPathAsync(dir string) {
	go func() {
		logger := logging.GetLogger("library-paths")
		store, err := database.OpenStoreWithConfig()
		if err != nil {
			logger.Warnf("scan %s: open store: %v", dir, err)
			return
		}
		defer store.Close()
		if err := metadata.ScanLibraryProgress(context.Background(), dir, store, nil); err != nil {
			logger.Warnf("scan %s: %v", dir, err)
		}
	}()
}

// libraryPathsHandler manages the configured library root paths that the Media
// Library UI browses. The previous UI called /api/library/paths but no backend
// route existed, so the library always appeared empty and "Add Library Path"
// silently failed. GET returns {"paths": [...]}, POST {"path": "..."} adds and
// scans it, DELETE {"path": "..."} removes it.
func libraryPathsHandler() http.Handler {
	type body struct {
		Path string `json:"path"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			libraryPathsMu.Lock()
			paths := readLibraryPaths()
			libraryPathsMu.Unlock()
			writeJSON(w, map[string]any{"paths": paths})

		case http.MethodPost:
			var q body
			if err := json.NewDecoder(r.Body).Decode(&q); err != nil || q.Path == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			clean, err := security.ValidateAndSanitizePath(q.Path)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			libraryPathsMu.Lock()
			paths := readLibraryPaths()
			if !slices.Contains(paths, clean) {
				paths = append(paths, clean)
				sort.Strings(paths)
				if err := writeLibraryPaths(paths); err != nil {
					libraryPathsMu.Unlock()
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			libraryPathsMu.Unlock()
			scanPathAsync(clean)
			writeJSON(w, map[string]any{"paths": paths})

		case http.MethodDelete:
			var q body
			if err := json.NewDecoder(r.Body).Decode(&q); err != nil || q.Path == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			libraryPathsMu.Lock()
			paths := readLibraryPaths()
			paths = slices.DeleteFunc(paths, func(p string) bool { return p == q.Path })
			err := writeLibraryPaths(paths)
			libraryPathsMu.Unlock()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{"paths": paths})

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// libraryRescanHandler rescans every configured library path from disk.
func libraryRescanHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		libraryPathsMu.Lock()
		paths := readLibraryPaths()
		libraryPathsMu.Unlock()
		for _, p := range paths {
			scanPathAsync(p)
		}
		writeJSON(w, map[string]any{"status": "started", "paths": len(paths)})
	})
}

// writeJSON writes v as a JSON response.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
