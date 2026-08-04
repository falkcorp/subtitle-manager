// file: pkg/monitoring/monitor.go
// version: 1.2.0
// last-edited: 2026-08-04
// guid: 12345678-1234-1234-1234-123456789012

package monitoring

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/radarr"
	"github.com/jdfalk/subtitle-manager/pkg/scanner"
	"github.com/jdfalk/subtitle-manager/pkg/sonarr"
)

// MonitorStatus represents the current monitoring state of a media item.
type MonitorStatus string

const (
	StatusPending     MonitorStatus = "pending"
	StatusMonitoring  MonitorStatus = "monitoring"
	StatusFound       MonitorStatus = "found"
	StatusFailed      MonitorStatus = "failed"
	StatusBlacklisted MonitorStatus = "blacklisted"
)

// MonitoredItem represents a media item that is being monitored for subtitle availability.
type MonitoredItem struct {
	ID          string        `json:"id"`
	MediaID     string        `json:"media_id"`
	Path        string        `json:"path"`
	Languages   []string      `json:"languages"`
	LastChecked time.Time     `json:"last_checked"`
	Status      MonitorStatus `json:"status"`
	RetryCount  int           `json:"retry_count"`
	MaxRetries  int           `json:"max_retries"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// EpisodeMonitor provides automatic subtitle monitoring for TV episodes and movies.
type EpisodeMonitor struct {
	interval     time.Duration
	sonarr       *sonarr.Client
	radarr       *radarr.Client
	store        database.SubtitleStore
	maxRetries   int
	qualityCheck bool
	logger       *logrus.Entry
}

// NewEpisodeMonitor creates a new episode monitoring instance.
func NewEpisodeMonitor(
	interval time.Duration,
	sonarrClient *sonarr.Client,
	radarrClient *radarr.Client,
	store database.SubtitleStore,
	maxRetries int,
	qualityCheck bool,
) *EpisodeMonitor {
	return &EpisodeMonitor{
		interval:     interval,
		sonarr:       sonarrClient,
		radarr:       radarrClient,
		store:        store,
		maxRetries:   maxRetries,
		qualityCheck: qualityCheck,
		logger:       logging.GetLogger("monitoring"),
	}
}

// Start begins the monitoring process, running until the context is cancelled.
func (m *EpisodeMonitor) Start(ctx context.Context) error {
	m.logger.Info("Starting episode monitoring")
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Run initial check
	if err := m.checkForSubtitles(ctx); err != nil {
		m.logger.Errorf("Initial monitoring check failed: %v", err)
		return err
	}

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Stopping episode monitoring")
			return ctx.Err()
		case <-ticker.C:
			if err := m.checkForSubtitles(ctx); err != nil {
				m.logger.Warnf("Monitoring check failed: %v", err)
			}
		}
	}
}

// checkForSubtitles performs a single monitoring cycle.
func (m *EpisodeMonitor) checkForSubtitles(ctx context.Context) error {
	// Get monitored items that need checking
	items, err := m.getItemsToCheck()
	if err != nil {
		return err
	}

	m.logger.Debugf("Checking %d monitored items", len(items))

	for _, item := range items {
		if err := m.processItem(ctx, item); err != nil {
			m.logger.Warnf("Failed to process item %s: %v", item.Path, err)
		}
	}

	return nil
}

// processItem processes a single monitored item.
func (m *EpisodeMonitor) processItem(ctx context.Context, item *MonitoredItem) error {
	// Skip if item is blacklisted
	if m.IsBlacklisted(item.ID, "") {
		m.logger.Debugf("Skipping blacklisted item: %s", item.Path)
		return nil
	}

	// Update last checked time
	item.LastChecked = time.Now()

	// A file with its own language profile is downloaded for the languages that
	// profile asks for, in priority order, instead of item.Languages.
	//
	// The profile wins because item.Languages is accumulated rather than
	// curated — sync.go unions in every language it is ever asked for and never
	// removes one — while an assignment is a deliberate per-file choice. See
	// scanner.ProcessWithProfileIfAssigned for the full reasoning.
	if handled, err := scanner.ProcessWithProfileIfAssigned(ctx, item.Path, "", nil, true, m.store); handled {
		if err != nil {
			m.logger.Debugf("assigned profile failed for %s: %v", item.Path, err)
		}
		return m.recordCheckOutcome(item, err == nil)
	}

	// Check each requested language
	foundAny := false
	for _, lang := range item.Languages {
		if err := m.checkLanguage(ctx, item, lang); err != nil {
			m.logger.Debugf("Language %s failed for %s: %v", lang, item.Path, err)
			continue
		}
		foundAny = true
	}

	return m.recordCheckOutcome(item, foundAny)
}

// recordCheckOutcome applies the retry/blacklist bookkeeping for one check and
// persists the item. It is shared by the profile-driven and per-language paths
// so a profile-assigned file still retries, fails and auto-blacklists exactly
// like any other monitored item.
func (m *EpisodeMonitor) recordCheckOutcome(item *MonitoredItem, foundAny bool) error {
	// Update retry count and status
	if !foundAny {
		item.RetryCount++
		if item.RetryCount >= item.MaxRetries {
			item.Status = StatusFailed
			m.logger.Warnf("Item %s failed after %d retries", item.Path, item.RetryCount)

			// Auto-blacklist on failure
			if err := m.AutoBlacklistOnFailure(item); err != nil {
				m.logger.Errorf("Failed to auto-blacklist item %s: %v", item.Path, err)
			}
		} else {
			item.Status = StatusMonitoring
		}
	} else {
		item.Status = StatusFound
		m.logger.Infof("Found subtitles for %s", item.Path)
	}

	// Update item status in database
	return m.updateMonitoredItem(item)
}

// checkLanguage attempts to download subtitles for a specific language via the
// full download pipeline (scanner.ProcessFile), so the automatic monitoring
// loop gets the same behaviour as manual downloads: score-gating, Whisper
// fallback, post-processing, single-language naming and score-based upgrade —
// instead of the previous bare fetch-and-write.
func (m *EpisodeMonitor) checkLanguage(ctx context.Context, item *MonitoredItem, lang string) error {
	return scanner.ProcessFile(ctx, item.Path, lang, "", nil, true, m.store)
}

// getItemsToCheck retrieves monitored items that need checking.
func (m *EpisodeMonitor) getItemsToCheck() ([]*MonitoredItem, error) {
	dbItems, err := m.store.GetMonitoredItemsToCheck(m.interval)
	if err != nil {
		return nil, err
	}

	var items []*MonitoredItem
	for _, dbItem := range dbItems {
		// Convert database.MonitoredItem to monitoring.MonitoredItem
		var languages []string
		if err := json.Unmarshal([]byte(dbItem.Languages), &languages); err != nil {
			m.logger.Warnf("Failed to parse languages for item %s: %v", dbItem.Path, err)
			continue
		}

		item := &MonitoredItem{
			ID:          dbItem.ID,
			MediaID:     dbItem.MediaID,
			Path:        dbItem.Path,
			Languages:   languages,
			LastChecked: dbItem.LastChecked,
			Status:      MonitorStatus(dbItem.Status),
			RetryCount:  dbItem.RetryCount,
			MaxRetries:  dbItem.MaxRetries,
			CreatedAt:   dbItem.CreatedAt,
			UpdatedAt:   dbItem.UpdatedAt,
		}
		items = append(items, item)
	}

	return items, nil
}

// updateMonitoredItem updates the item status in the database.
func (m *EpisodeMonitor) updateMonitoredItem(item *MonitoredItem) error {
	// Convert back to database.MonitoredItem
	languagesJSON, err := json.Marshal(item.Languages)
	if err != nil {
		return err
	}

	dbItem := &database.MonitoredItem{
		ID:          item.ID,
		MediaID:     item.MediaID,
		Path:        item.Path,
		Languages:   string(languagesJSON),
		LastChecked: item.LastChecked,
		Status:      string(item.Status),
		RetryCount:  item.RetryCount,
		MaxRetries:  item.MaxRetries,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   time.Now(),
	}

	return m.store.UpdateMonitoredItem(dbItem)
}
