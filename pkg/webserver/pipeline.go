// file: pkg/webserver/pipeline.go
// version: 1.0.0
// guid: 6e57ab8d-480a-41c5-8d2f-3eab3a966511

package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/scanner"
	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/jdfalk/subtitle-manager/pkg/subsync"
	"github.com/jdfalk/subtitle-manager/pkg/subtitles"
	"github.com/jdfalk/subtitle-manager/pkg/video"
)

// librarySearchState tracks the async library-search sweep for the web UI.
type librarySearchState struct {
	Running bool   `json:"running"`
	Lang    string `json:"lang,omitempty"`
	Error   string `json:"error,omitempty"`
}

var (
	librarySearchMu     sync.Mutex
	librarySearchStatus librarySearchState
)

// librarySearchHandler downloads subtitles for every item in the persisted media
// library (populated by the library scan and the Sonarr/Radarr sync). This is the
// web surface of the W1 library→search bridge. It runs asynchronously; poll
// /api/library/search/status for progress.
func librarySearchHandler() http.Handler {
	type req struct {
		Lang    string `json:"lang"`
		Upgrade bool   `json:"upgrade"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var q req
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil || q.Lang == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := security.ValidateLanguageCode(q.Lang); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		librarySearchMu.Lock()
		if librarySearchStatus.Running {
			librarySearchMu.Unlock()
			w.WriteHeader(http.StatusConflict)
			return
		}
		librarySearchStatus = librarySearchState{Running: true, Lang: q.Lang}
		librarySearchMu.Unlock()

		go func() {
			logger := logging.GetLogger("library-search")
			store, err := database.OpenStoreWithConfig()
			if err != nil {
				finishLibrarySearch(err)
				return
			}
			defer store.Close()
			workers := viper.GetInt("scan_workers")
			if workers < 1 {
				workers = 4
			}
			err = scanner.ProcessLibrary(context.Background(), q.Lang, "", nil, q.Upgrade, workers, store)
			if err != nil {
				logger.Warnf("library search: %v", err)
			}
			finishLibrarySearch(err)
		}()

		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	})
}

func finishLibrarySearch(err error) {
	librarySearchMu.Lock()
	librarySearchStatus.Running = false
	if err != nil {
		librarySearchStatus.Error = err.Error()
	}
	librarySearchMu.Unlock()
}

// librarySearchStatusHandler reports the current library-search sweep status.
func librarySearchStatusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		librarySearchMu.Lock()
		s := librarySearchStatus
		librarySearchMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s)
	})
}

// dualSubHandler generates bilingual "double subs" (W5) from an uploaded subtitle
// file and returns the sentinel-tagged SRT. Form fields: file (subtitle), lang
// (target, default zh), service, grpc.
func dualSubHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		file, hdr, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()

		target := r.FormValue("lang")
		if target == "" {
			target = "zh"
		}
		if err := security.ValidateLanguageCode(target); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		service := r.FormValue("service")
		if service == "" {
			service = viper.GetString("translate_service")
			if service == "" {
				service = "google"
			}
		}
		grpcAddr := r.FormValue("grpc")
		if grpcAddr == "" {
			grpcAddr = viper.GetString("grpc_addr")
		}

		in, err := os.CreateTemp("", "dualin-*"+filepath.Ext(hdr.Filename))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer os.Remove(in.Name())
		if _, err := io.Copy(in, file); err != nil {
			_ = in.Close()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = in.Close()

		out, err := os.CreateTemp("", "dualout-*.srt")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = out.Close()
		defer os.Remove(out.Name())

		if err := subtitles.GenerateDualSubtitles(in.Name(), out.Name(), target,
			service, viper.GetString("google_api_key"), viper.GetString("openai_api_key"), grpcAddr); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		data, err := os.ReadFile(out.Name())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/x-subrip")
		w.Header().Set("Content-Disposition", "attachment; filename=dual.eo.srt")
		_, _ = w.Write(data)
	})
}

// verifyHandler runs subtitle drift verification (W4) for a server-side media +
// subtitle path and returns the DriftReport as JSON. It is synchronous and can
// take a while (it transcribes several audio windows with Whisper).
func verifyHandler() http.Handler {
	type req struct {
		Media    string `json:"media"`
		Subtitle string `json:"subtitle"`
		Lang     string `json:"lang"`
		Anchors  int    `json:"anchors"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var q req
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil || q.Media == "" || q.Subtitle == "" || q.Lang == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := security.ValidateAndSanitizePath(q.Media); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := security.ValidateAndSanitizePath(q.Subtitle); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := security.ValidateLanguageCode(q.Lang); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		info, err := video.AnalyzeVideo(q.Media)
		if err != nil || info.Duration <= 0 {
			http.Error(w, fmt.Sprintf("probe media: %v", err), http.StatusBadRequest)
			return
		}
		measure, err := subsync.NewWhisperMeasurer(q.Media, q.Subtitle, q.Lang, viper.GetString("openai_api_key"), 0, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rep, err := subsync.Verify(r.Context(), subsync.VerifyOptions{
			MediaDuration: info.Duration,
			Anchors:       q.Anchors,
		}, measure)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rep)
	})
}
