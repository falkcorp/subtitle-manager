// file: pkg/webserver/wanted.go
// version: 1.0.0
// guid: 7a3c5e19-4b28-4d06-9f71-2c8e0a45b3d7
// last-edited: 2026-07-25

package webserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/arr"
	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/monitoring"
	"github.com/jdfalk/subtitle-manager/pkg/radarr"
	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/jdfalk/subtitle-manager/pkg/sonarr"
)

// Defaults for the monitor configuration. Interval is in minutes.
const (
	defaultMonitorInterval   = 60
	defaultMonitorMaxRetries = 3
)

// monitorInterval returns the configured monitoring interval.
//
// The floor is not cosmetic: time.NewTicker panics on a non-positive duration,
// so an unset or zero monitor.interval would crash the web server at startup
// rather than fall back to a default.
func monitorInterval() time.Duration {
	minutes := viper.GetInt("monitor.interval")
	if minutes < 1 {
		minutes = defaultMonitorInterval
	}
	return time.Duration(minutes) * time.Minute
}

// monitorLanguages returns the languages a newly wanted item is monitored for
// when the caller does not specify any.
func monitorLanguages() []string {
	langs := viper.GetStringSlice("monitor.languages")
	if len(langs) == 0 {
		return []string{"en"}
	}
	return langs
}

// monitorMaxRetries returns how many times an item is retried before it is
// marked failed and auto-blacklisted.
func monitorMaxRetries() int {
	n := viper.GetInt("monitor.max_retries")
	if n < 1 {
		return defaultMonitorMaxRetries
	}
	return n
}

// startMonitoring starts the automatic subtitle monitoring loop, but only when
// the operator has explicitly enabled it.
//
// It is OFF BY DEFAULT, and that is a deliberate safety decision rather than
// conservatism. The loop calls scanner.ProcessFile with upgrade enabled for
// every monitored item, which contacts subtitle providers and writes subtitle
// files into the operator's media directories, unattended, on a timer. Turning
// that on as a side effect of upgrading the server would mean an installation
// starts modifying a media library nobody asked it to touch. Bazarr enables its
// equivalent by default, but Bazarr asks during setup; there is no such prompt
// here yet, so the safe default wins until there is.
//
// Set monitor.enabled to true to turn it on. Related keys: monitor.interval
// (minutes), monitor.languages, monitor.max_retries, monitor.quality_check.
func startMonitoring(ctx context.Context) {
	logger := logging.GetLogger("monitor")

	if !viper.GetBool("monitor.enabled") {
		logger.Info("automatic subtitle monitoring is disabled; " +
			"set monitor.enabled to true to have subtitles downloaded on a schedule")
		return
	}

	store, err := database.GetSharedStore()
	if err != nil {
		logger.Warnf("automatic monitoring not started, cannot open store: %v", err)
		return
	}

	// nil clients when the integration is not configured — the monitor treats
	// a nil client as "skip this source" rather than an error.
	var sonarrClient *sonarr.Client
	if u := arr.BaseURL("integrations.sonarr"); u != "" {
		sonarrClient = sonarr.NewClient(u, viper.GetString("integrations.sonarr.api_key"))
	}
	var radarrClient *radarr.Client
	if u := arr.BaseURL("integrations.radarr"); u != "" {
		radarrClient = radarr.NewClient(u, viper.GetString("integrations.radarr.api_key"))
	}

	interval := monitorInterval()
	mon := monitoring.NewEpisodeMonitor(interval, sonarrClient, radarrClient, store,
		monitorMaxRetries(), viper.GetBool("monitor.quality_check"))

	logger.Infof("starting automatic subtitle monitoring every %s", interval)

	go func() {
		// Populate the wanted list from Sonarr/Radarr before the first pass, so
		// an operator with an *arr configured does not have to add paths by
		// hand. Deliberately *not* derived from the local media library: that
		// would enrol every scanned file at once and turn a single opt-in into
		// a library-wide download. Items can still be added one at a time via
		// POST /api/wanted.
		opts := monitoring.SyncOptions{
			Languages:  monitorLanguages(),
			MaxRetries: monitorMaxRetries(),
		}
		if sonarrClient != nil {
			if err := mon.SyncFromSonarr(ctx, opts); err != nil {
				logger.Warnf("sync wanted list from Sonarr: %v", err)
			}
		}
		if radarrClient != nil {
			if err := mon.SyncFromRadarr(ctx, opts); err != nil {
				logger.Warnf("sync wanted list from Radarr: %v", err)
			}
		}

		// Start blocks until ctx is cancelled, and returns early if the very
		// first check fails.
		if err := mon.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Warnf("automatic subtitle monitoring stopped: %v", err)
		}
	}()
}

// wantedItem is the API representation of a monitored item.
//
// It is a distinct type from database.MonitoredItem rather than a re-encoding
// of it because Languages is stored as a JSON string in the database; handing
// that to the UI would force it to parse JSON out of a JSON field.
type wantedItem struct {
	ID          string    `json:"id"`
	MediaID     string    `json:"media_id,omitempty"`
	Path        string    `json:"path"`
	Languages   []string  `json:"languages"`
	Status      string    `json:"status"`
	RetryCount  int       `json:"retry_count"`
	MaxRetries  int       `json:"max_retries"`
	LastChecked time.Time `json:"last_checked"`
	CreatedAt   time.Time `json:"created_at"`
}

// wantedHandler serves the "wanted" list: media that is being monitored
// because it is missing subtitles in one or more desired languages.
//
// This is Bazarr's central concept, and it is backed by the existing
// monitored_items table, which already carries status, retry count and
// blacklist integration. No schema change was needed.
//
//	GET    /api/wanted                       list every monitored item
//	POST   /api/wanted {path, languages}     start monitoring a path
//	DELETE /api/wanted {path}                stop monitoring a path
//
// Note that adding an item here does not by itself download anything. The
// monitoring loop that acts on this list is opt-in and off by default; see
// startMonitoring in server.go for why.
func wantedHandler() http.Handler {
	type body struct {
		Path      string   `json:"path"`
		Languages []string `json:"languages"`
	}
	logger := logging.GetLogger("wanted")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		store, err := database.GetSharedStore()
		if err != nil {
			logger.Warnf("open store: %v", err)
			http.Error(w, "database unavailable", http.StatusInternalServerError)
			return
		}

		switch r.Method {
		case http.MethodGet:
			items, err := store.ListMonitoredItems()
			if err != nil {
				logger.Warnf("list monitored items: %v", err)
				http.Error(w, "failed to list wanted items", http.StatusInternalServerError)
				return
			}
			// Always a slice, never nil: the UI iterates the response, and a
			// JSON null would force every caller to guard against it.
			out := make([]wantedItem, 0, len(items))
			for _, it := range items {
				var langs []string
				if err := json.Unmarshal([]byte(it.Languages), &langs); err != nil {
					// A row with unparseable languages is reported with none
					// rather than dropped, so it stays visible and fixable.
					logger.Warnf("item %s has unparseable languages %q: %v", it.Path, it.Languages, err)
					langs = nil
				}
				out = append(out, wantedItem{
					ID:          it.ID,
					MediaID:     it.MediaID,
					Path:        it.Path,
					Languages:   langs,
					Status:      it.Status,
					RetryCount:  it.RetryCount,
					MaxRetries:  it.MaxRetries,
					LastChecked: it.LastChecked,
					CreatedAt:   it.CreatedAt,
				})
			}
			writeJSON(w, out)

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
			langs := q.Languages
			if len(langs) == 0 {
				langs = monitorLanguages()
			}
			for _, l := range langs {
				if err := security.ValidateLanguageCode(l); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			}

			// nil Sonarr/Radarr clients: this path adds one item the operator
			// named, so no *arr lookup is involved.
			mon := monitoring.NewEpisodeMonitor(monitorInterval(), nil, nil, store,
				monitorMaxRetries(), false)
			opts := monitoring.SyncOptions{Languages: langs, MaxRetries: monitorMaxRetries()}
			if err := mon.AddToMonitoring(database.MediaItem{Path: clean}, opts); err != nil {
				logger.Warnf("add %s: %v", clean, err)
				http.Error(w, "failed to add wanted item", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)

		case http.MethodDelete:
			var q body
			if err := json.NewDecoder(r.Body).Decode(&q); err != nil || q.Path == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mon := monitoring.NewEpisodeMonitor(monitorInterval(), nil, nil, store,
				monitorMaxRetries(), false)
			if err := mon.RemoveFromMonitoring(q.Path); err != nil {
				logger.Warnf("remove %s: %v", q.Path, err)
				http.Error(w, "failed to remove wanted item", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}
