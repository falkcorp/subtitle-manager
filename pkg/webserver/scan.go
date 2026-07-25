// file: pkg/webserver/scan.go
// version: 1.1.0
// guid: c025a40c-894d-401d-a68d-a27b7f66c3b6
// last-edited: 2026-07-25

package webserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/metadata"
	"github.com/jdfalk/subtitle-manager/pkg/providers"
	"github.com/jdfalk/subtitle-manager/pkg/radarr"
	"github.com/jdfalk/subtitle-manager/pkg/scanner"
	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/jdfalk/subtitle-manager/pkg/sonarr"
	"github.com/jdfalk/subtitle-manager/pkg/tasks"
)

// scanStatus tracks progress for an active scan.
type scanStatus struct {
	Running   bool     `json:"running"`
	Completed int      `json:"completed"`
	Files     []string `json:"files"`
}

var (
	scanMu sync.Mutex
	status = scanStatus{Files: []string{}}
)

// libScanStatus tracks progress for library scanning.
type libScanStatus struct {
	Running   bool     `json:"running"`
	Completed int      `json:"completed"`
	Files     []string `json:"files"`
}

var (
	libMu     sync.Mutex
	libStatus = libScanStatus{Files: []string{}}
)

func scanHandler() http.Handler {
	type req struct {
		Provider  string `json:"provider"`
		Directory string `json:"directory"`
		Lang      string `json:"lang"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var q req
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Validate directory path to prevent path injection
		if q.Directory != "" {
			if _, err := security.ValidateAndSanitizePath(q.Directory); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		// Validate language code
		if q.Lang != "" {
			if err := security.ValidateLanguageCode(q.Lang); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		// Validate provider name
		if q.Provider != "" {
			if err := security.ValidateProviderName(q.Provider); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		scanMu.Lock()
		if status.Running {
			scanMu.Unlock()
			w.WriteHeader(http.StatusConflict)
			return
		}
		status = scanStatus{Running: true, Files: []string{}}
		scanMu.Unlock()

		// Start task with proper integration
		taskID := fmt.Sprintf("scan-%s-%s", q.Directory, q.Lang)
		task := tasks.Start(r.Context(), taskID, func(ctx context.Context) error {
			var p providers.Provider
			var err error
			if q.Provider != "" {
				p, err = providers.Get(q.Provider, "")
				if err != nil {
					return err
				}
			}

			// Count files for progress tracking
			var fileCount int
			videoExtensions := []string{".mkv", ".mp4", ".avi", ".mov"}
			isVideoFile := func(path string) bool {
				ext := strings.ToLower(filepath.Ext(path))
				for _, e := range videoExtensions {
					if ext == e {
						return true
					}
				}
				return false
			}

			err = filepath.Walk(q.Directory, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && isVideoFile(path) {
					fileCount++
				}
				return nil
			})
			if err != nil {
				return err
			}

			processed := 0
			cb := func(f string) {
				scanMu.Lock()
				status.Completed++
				status.Files = append(status.Files, f)
				scanMu.Unlock()

				processed++
				if fileCount > 0 {
					tasks.Update(taskID, (processed*100)/fileCount)
				}
			}

			// Open database store for download tracking
			var store database.SubtitleStore
			if dbPath := viper.GetString("db_path"); dbPath != "" {
				backend := viper.GetString("db_backend")
				if s, err := database.OpenStore(dbPath, backend); err == nil {
					store = s
					defer s.Close()
				}
			}

			return scanner.ScanDirectoryProgress(ctx, q.Directory, q.Lang, q.Provider, p, false, 2, store, cb)
		})

		go func() {
			// Wait for task completion and clean up local status
			<-time.After(100 * time.Millisecond)
			for task.GetStatus() == "running" {
				<-time.After(1 * time.Second)
			}
			scanMu.Lock()
			status.Running = false
			scanMu.Unlock()
		}()

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"task_id": taskID})
	})
}

func scanStatusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		scanMu.Lock()
		defer scanMu.Unlock()
		_ = json.NewEncoder(w).Encode(status)
	})
}

func libraryScanHandler(db *sql.DB) http.Handler {
	type req struct {
		Path string `json:"path"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var q req
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil || q.Path == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Validate and sanitize the path to prevent path injection
		if _, err := security.ValidateAndSanitizePath(q.Path); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		libMu.Lock()
		if libStatus.Running {
			libMu.Unlock()
			w.WriteHeader(http.StatusConflict)
			return
		}
		libStatus = libScanStatus{Running: true, Files: []string{}}
		libMu.Unlock()
		go func() {
			// Shared, so not closed here: see database.GetSharedStore.
			store, err := database.GetSharedStore()
			if err != nil {
				libMu.Lock()
				libStatus.Running = false
				libMu.Unlock()
				return
			}
			cb := func(f string) {
				libMu.Lock()
				libStatus.Completed++
				libStatus.Files = append(libStatus.Files, f)
				libMu.Unlock()
			}
			_ = metadata.ScanLibraryProgress(context.Background(), q.Path, store, cb)

			if viper.GetBool("integrations.radarr.enabled") {
				host := viper.GetString("integrations.radarr.host")
				port := viper.GetString("integrations.radarr.port")
				key := viper.GetString("integrations.radarr.api_key")
				ssl := viper.GetBool("integrations.radarr.ssl")
				base := strings.Trim(viper.GetString("integrations.radarr.base_url"), "/")
				scheme := "http"
				if ssl {
					scheme = "https"
				}
				url := fmt.Sprintf("%s://%s:%s/%s", scheme, host, port, base)
				c := radarr.NewClient(url, key)
				_ = radarr.Sync(context.Background(), c, store)
			}
			if viper.GetBool("integrations.sonarr.enabled") {
				host := viper.GetString("integrations.sonarr.host")
				port := viper.GetString("integrations.sonarr.port")
				key := viper.GetString("integrations.sonarr.api_key")
				ssl := viper.GetBool("integrations.sonarr.ssl")
				base := strings.Trim(viper.GetString("integrations.sonarr.base_url"), "/")
				scheme := "http"
				if ssl {
					scheme = "https"
				}
				url := fmt.Sprintf("%s://%s:%s/%s", scheme, host, port, base)
				c := sonarr.NewClient(url, key)
				_ = sonarr.Sync(context.Background(), c, store)
			}
			libMu.Lock()
			libStatus.Running = false
			libMu.Unlock()
		}()
		w.WriteHeader(http.StatusAccepted)
	})
}

func libraryScanStatusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		libMu.Lock()
		defer libMu.Unlock()
		_ = json.NewEncoder(w).Encode(libStatus)
	})
}
