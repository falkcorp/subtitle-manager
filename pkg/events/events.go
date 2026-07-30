// file: pkg/events/events.go
// version: 1.1.0
// guid: 123e4567-e89b-12d3-a456-426614174006
// last-edited: 2026-07-30

package events

import (
	"context"
	"time"
)

// EventPublisher publishes events to registered handlers.
type EventPublisher interface {
	PublishSubtitleDownloaded(ctx context.Context, data SubtitleDownloadedData)
	PublishSubtitleUpgraded(ctx context.Context, data SubtitleUpgradedData)
	PublishSubtitleFailed(ctx context.Context, data SubtitleFailedData)
	PublishSearchFailed(ctx context.Context, data SearchFailedData)
}

// SubtitleDownloadedData represents data for subtitle download events.
type SubtitleDownloadedData struct {
	FilePath     string    `json:"file_path"`
	SubtitlePath string    `json:"subtitle_path"`
	Language     string    `json:"language"`
	Provider     string    `json:"provider"`
	Score        float64   `json:"score"`
	Size         int64     `json:"size"`
	Timestamp    time.Time `json:"timestamp"`
}

// SubtitleUpgradedData represents data for subtitle upgrade events.
type SubtitleUpgradedData struct {
	FilePath        string    `json:"file_path"`
	OldSubtitlePath string    `json:"old_subtitle_path"`
	NewSubtitlePath string    `json:"new_subtitle_path"`
	Language        string    `json:"language"`
	OldProvider     string    `json:"old_provider"`
	NewProvider     string    `json:"new_provider"`
	OldScore        float64   `json:"old_score"`
	NewScore        float64   `json:"new_score"`
	Timestamp       time.Time `json:"timestamp"`
}

// SubtitleFailedData represents data for subtitle failure events.
type SubtitleFailedData struct {
	FilePath  string    `json:"file_path"`
	Language  string    `json:"language"`
	Provider  string    `json:"provider,omitempty"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

// SearchFailedData represents data for search failure events.
type SearchFailedData struct {
	Query     string    `json:"query"`
	Language  string    `json:"language"`
	Provider  string    `json:"provider,omitempty"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

// globalPublisher holds the global event publisher instance.
var globalPublisher EventPublisher

// SetGlobalPublisher sets the global event publisher.
func SetGlobalPublisher(publisher EventPublisher) {
	globalPublisher = publisher
}

// GetGlobalPublisher returns the current global event publisher, which may be
// nil. It exists so a caller that swaps the publisher — a test asserting which
// code paths emit events, in practice — can put the previous one back rather
// than leaving the global cleared for everything that runs afterwards.
func GetGlobalPublisher() EventPublisher {
	return globalPublisher
}

// PublishSubtitleDownloaded publishes a subtitle downloaded event.
func PublishSubtitleDownloaded(ctx context.Context, data SubtitleDownloadedData) {
	if globalPublisher != nil {
		globalPublisher.PublishSubtitleDownloaded(ctx, data)
	}
}

// PublishSubtitleUpgraded publishes a subtitle upgraded event.
func PublishSubtitleUpgraded(ctx context.Context, data SubtitleUpgradedData) {
	if globalPublisher != nil {
		globalPublisher.PublishSubtitleUpgraded(ctx, data)
	}
}

// PublishSubtitleFailed publishes a subtitle failed event.
func PublishSubtitleFailed(ctx context.Context, data SubtitleFailedData) {
	if globalPublisher != nil {
		globalPublisher.PublishSubtitleFailed(ctx, data)
	}
}

// PublishSearchFailed publishes a search failed event.
func PublishSearchFailed(ctx context.Context, data SearchFailedData) {
	if globalPublisher != nil {
		globalPublisher.PublishSearchFailed(ctx, data)
	}
}
