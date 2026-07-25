// file: pkg/monitoring/sync.go
// version: 1.1.0
// guid: 12345678-1234-1234-1234-123456789013
// last-edited: 2026-07-25

package monitoring

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/database"
)

// SyncOptions contains configuration for media library synchronization.
type SyncOptions struct {
	Languages    []string // Languages to monitor for
	MaxRetries   int      // Maximum retry attempts per item
	ForceRefresh bool     // Whether to refresh existing items
}

// SyncFromSonarr synchronizes TV episodes from Sonarr and adds them to monitoring.
func (m *EpisodeMonitor) SyncFromSonarr(ctx context.Context, opts SyncOptions) error {
	if m.sonarr == nil {
		return nil // Skip if Sonarr is not configured
	}

	m.logger.Info("Syncing episodes from Sonarr")

	// Get episodes from Sonarr
	episodes, err := m.sonarr.Episodes(ctx)
	if err != nil {
		return err
	}

	m.logger.Infof("Found %d episodes from Sonarr", len(episodes))

	// Process each episode
	for _, episode := range episodes {
		if err := m.addToMonitoring(episode, opts); err != nil {
			m.logger.Warnf("Failed to add episode %s to monitoring: %v", episode.Path, err)
		}
	}

	return nil
}

// SyncFromRadarr synchronizes movies from Radarr and adds them to monitoring.
func (m *EpisodeMonitor) SyncFromRadarr(ctx context.Context, opts SyncOptions) error {
	if m.radarr == nil {
		return nil // Skip if Radarr is not configured
	}

	m.logger.Info("Syncing movies from Radarr")

	// Get movies from Radarr
	movies, err := m.radarr.Movies(ctx)
	if err != nil {
		return err
	}

	m.logger.Infof("Found %d movies from Radarr", len(movies))

	// Process each movie
	for _, movie := range movies {
		if err := m.addToMonitoring(movie, opts); err != nil {
			m.logger.Warnf("Failed to add movie %s to monitoring: %v", movie.Path, err)
		}
	}

	return nil
}

// AddToMonitoring adds a single media item to the monitoring ("wanted") list,
// merging languages if the path is already monitored.
//
// Exported for callers outside this package — the web server's /api/wanted
// handler, which adds a path the operator picked by hand rather than one
// discovered by a Sonarr/Radarr sync. For a manual add, only Path need be set;
// MediaID stays empty, as there is no media_items row to link to.
func (m *EpisodeMonitor) AddToMonitoring(mediaItem database.MediaItem, opts SyncOptions) error {
	return m.addToMonitoring(mediaItem, opts)
}

// addToMonitoring adds a media item to the monitoring system.
func (m *EpisodeMonitor) addToMonitoring(mediaItem database.MediaItem, opts SyncOptions) error {
	// Check if file exists
	if _, err := os.Stat(mediaItem.Path); os.IsNotExist(err) {
		m.logger.Debugf("Skipping non-existent file: %s", mediaItem.Path)
		return nil
	}

	// Check if already being monitored
	existing, err := m.store.ListMonitoredItems()
	if err != nil {
		return err
	}
	for _, item := range existing {
		if item.Path == mediaItem.Path {
			// Merge languages if needed
			var langs []string
			if err := json.Unmarshal([]byte(item.Languages), &langs); err == nil {
				updated := false
				for _, lang := range opts.Languages {
					if !containsString(langs, lang) {
						langs = append(langs, lang)
						updated = true
					}
				}
				if updated || item.MaxRetries != opts.MaxRetries || opts.ForceRefresh {
					b, _ := json.Marshal(langs)
					item.Languages = string(b)
					item.MaxRetries = opts.MaxRetries
					if err := m.store.UpdateMonitoredItem(&item); err != nil {
						return err
					}
					m.logger.Debugf("Updated monitored item: %s", item.Path)
				} else {
					m.logger.Debugf("Already monitoring: %s", mediaItem.Path)
				}
			}
			return nil
		}
	}

	// Serialize languages
	languagesJSON, err := json.Marshal(opts.Languages)
	if err != nil {
		return err
	}

	// Create monitored item
	monitoredItem := &database.MonitoredItem{
		MediaID:     mediaItem.ID,
		Path:        mediaItem.Path,
		Languages:   string(languagesJSON),
		LastChecked: time.Time{}, // Will be checked on first monitoring cycle
		Status:      "pending",
		RetryCount:  0,
		MaxRetries:  opts.MaxRetries,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Insert into database
	if err := m.store.InsertMonitoredItem(monitoredItem); err != nil {
		return err
	}

	m.logger.Debugf("Added to monitoring: %s (languages: %v)", mediaItem.Path, opts.Languages)
	return nil
}

// RemoveFromMonitoring removes a media item from monitoring by path.
func (m *EpisodeMonitor) RemoveFromMonitoring(path string) error {
	items, err := m.store.ListMonitoredItems()
	if err != nil {
		return err
	}

	for _, item := range items {
		if item.Path == path {
			if err := m.store.DeleteMonitoredItem(item.ID); err != nil {
				return err
			}
			m.logger.Infof("Removed from monitoring: %s", path)
			return nil
		}
	}

	return nil // Not found, no error
}

// GetMonitoringStats returns statistics about the monitoring system.
func (m *EpisodeMonitor) GetMonitoringStats() (*MonitoringStats, error) {
	items, err := m.store.ListMonitoredItems()
	if err != nil {
		return nil, err
	}

	stats := &MonitoringStats{
		Total:       len(items),
		Pending:     0,
		Monitoring:  0,
		Found:       0,
		Failed:      0,
		Blacklisted: 0,
	}

	for _, item := range items {
		switch item.Status {
		case "pending":
			stats.Pending++
		case "monitoring":
			stats.Monitoring++
		case "found":
			stats.Found++
		case "failed":
			stats.Failed++
		case "blacklisted":
			stats.Blacklisted++
		}
	}

	return stats, nil
}

// MonitoringStats contains statistics about the monitoring system.
type MonitoringStats struct {
	Total       int `json:"total"`
	Pending     int `json:"pending"`
	Monitoring  int `json:"monitoring"`
	Found       int `json:"found"`
	Failed      int `json:"failed"`
	Blacklisted int `json:"blacklisted"`
}

// containsString checks if a slice contains a string.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
