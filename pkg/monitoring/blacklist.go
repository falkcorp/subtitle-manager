// file: pkg/monitoring/blacklist.go
// version: 1.1.0
// guid: 12345678-1234-1234-1234-123456789017

package monitoring

import (
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/database"
)

// BlacklistReason represents why an item was blacklisted.
type BlacklistReason string

const (
	ReasonMaxRetriesExceeded BlacklistReason = "max_retries_exceeded"
	ReasonManualBlacklist    BlacklistReason = "manual_blacklist"
	ReasonNoSubtitlesFound   BlacklistReason = "no_subtitles_found"
	ReasonQualityIssues      BlacklistReason = "quality_issues"
	ReasonProviderError      BlacklistReason = "provider_error"
)

// BlacklistEntry represents a blacklisted item with metadata.
type BlacklistEntry struct {
	ItemID        string          `json:"item_id"`
	Path          string          `json:"path"`
	Language      string          `json:"language"`
	Reason        BlacklistReason `json:"reason"`
	Details       string          `json:"details"`
	BlacklistedAt time.Time       `json:"blacklisted_at"`
	ExpiresAt     *time.Time      `json:"expires_at,omitempty"`
}

// BlacklistManager manages blacklisted items and providers.
type BlacklistManager struct {
	store database.SubtitleStore
}

// NewBlacklistManager creates a new blacklist manager.
func NewBlacklistManager(store database.SubtitleStore) *BlacklistManager {
	return &BlacklistManager{
		store: store,
	}
}

// AddToBlacklist adds an item to the blacklist.
func (m *EpisodeMonitor) AddToBlacklist(itemID, path, language string, reason BlacklistReason, details string, duration *time.Duration) error {
	var expiresAt *time.Time
	if duration != nil {
		exp := time.Now().Add(*duration)
		expiresAt = &exp
	}

	entry := BlacklistEntry{
		ItemID:        itemID,
		Path:          path,
		Language:      language,
		Reason:        reason,
		Details:       details,
		BlacklistedAt: time.Now(),
		ExpiresAt:     expiresAt,
	}

	// Persist the full entry (reason + expiry) when the store supports it, so
	// the blacklist — including expiry — survives restarts. Otherwise the
	// entry's metadata is not durable and only the item status is kept.
	if bs, ok := m.store.(database.BlacklistStore); ok {
		if err := bs.InsertBlacklist(&database.BlacklistItem{
			ItemID:    entry.ItemID,
			Path:      entry.Path,
			Language:  entry.Language,
			Reason:    string(entry.Reason),
			Details:   entry.Details,
			CreatedAt: entry.BlacklistedAt,
			ExpiresAt: entry.ExpiresAt,
		}); err != nil {
			return err
		}
	}

	// Update the monitored item status to blacklisted
	items, err := m.store.ListMonitoredItems()
	if err != nil {
		return err
	}

	for _, item := range items {
		if item.ID == itemID {
			item.Status = "blacklisted"
			item.UpdatedAt = time.Now()
			return m.store.UpdateMonitoredItem(&item)
		}
	}

	m.logger.Infof("Blacklisted item %s (language: %s) - reason: %s", path, language, reason)
	return nil
}

// RemoveFromBlacklist removes an item from the blacklist.
func (m *EpisodeMonitor) RemoveFromBlacklist(itemID string) error {
	// Update the monitored item status from blacklisted
	items, err := m.store.ListMonitoredItems()
	if err != nil {
		return err
	}

	// Drop any persisted blacklist entries for this item.
	if bs, ok := m.store.(database.BlacklistStore); ok {
		if _, err := bs.DeleteBlacklistByItem(itemID); err != nil {
			return err
		}
	}

	for _, item := range items {
		if item.ID == itemID && item.Status == "blacklisted" {
			item.Status = "pending"
			item.RetryCount = 0 // Reset retry count
			item.UpdatedAt = time.Now()
			err := m.store.UpdateMonitoredItem(&item)
			if err == nil {
				m.logger.Infof("Removed item %s from blacklist", item.Path)
			}
			return err
		}
	}

	return nil
}

// IsBlacklisted checks if an item is currently blacklisted. When the store
// persists blacklist entries, a matching non-expired entry (for the same
// language, or an all-language entry) counts; expired entries are ignored.
// Otherwise it falls back to the monitored item's status.
func (m *EpisodeMonitor) IsBlacklisted(itemID, language string) bool {
	if bs, ok := m.store.(database.BlacklistStore); ok {
		entries, err := bs.ListBlacklist()
		if err == nil {
			now := time.Now()
			for i := range entries {
				e := &entries[i]
				if e.ItemID != itemID || e.Expired(now) {
					continue
				}
				if e.Language == "" || language == "" || e.Language == language {
					return true
				}
			}
			return false
		}
	}

	items, err := m.store.ListMonitoredItems()
	if err != nil {
		return false
	}

	for _, item := range items {
		if item.ID == itemID && item.Status == "blacklisted" {
			return true
		}
	}

	return false
}

// CleanupExpiredBlacklist removes expired blacklist entries from persistent
// storage. It is a no-op when the store does not persist the blacklist.
func (m *EpisodeMonitor) CleanupExpiredBlacklist() error {
	bs, ok := m.store.(database.BlacklistStore)
	if !ok {
		m.logger.Debug("blacklist store does not persist entries; nothing to expire")
		return nil
	}
	n, err := bs.DeleteExpiredBlacklist(time.Now())
	if err != nil {
		return err
	}
	if n > 0 {
		m.logger.Infof("removed %d expired blacklist entries", n)
	}
	return nil
}

// GetBlacklistedItems returns all currently blacklisted items.
func (m *EpisodeMonitor) GetBlacklistedItems() ([]database.MonitoredItem, error) {
	items, err := m.store.ListMonitoredItems()
	if err != nil {
		return nil, err
	}

	var blacklisted []database.MonitoredItem
	for _, item := range items {
		if item.Status == "blacklisted" {
			blacklisted = append(blacklisted, item)
		}
	}

	return blacklisted, nil
}

// AutoBlacklistOnFailure automatically blacklists items that have exceeded retry limits.
func (m *EpisodeMonitor) AutoBlacklistOnFailure(item *MonitoredItem) error {
	if item.RetryCount >= item.MaxRetries {
		duration := time.Hour * 24 * 7 // Blacklist for a week
		return m.AddToBlacklist(
			item.ID,
			item.Path,
			"", // All languages
			ReasonMaxRetriesExceeded,
			"Maximum retry attempts exceeded",
			&duration,
		)
	}
	return nil
}
