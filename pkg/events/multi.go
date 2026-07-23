// file: pkg/events/multi.go
// version: 1.0.0
// guid: 3e7b1c9a-2d4f-4a68-b0c5-8f1e6d3a2b74
// last-edited: 2026-07-23

package events

import "context"

// MultiPublisher fans out every event to each of the wrapped publishers, in
// order. A nil publisher in the list is skipped. This lets several independent
// subscribers (webhooks, notifications, ...) receive the same events without
// any one of them owning the global publisher slot.
type MultiPublisher struct {
	publishers []EventPublisher
}

// NewMultiPublisher returns a MultiPublisher wrapping the given publishers.
// nil entries are ignored.
func NewMultiPublisher(publishers ...EventPublisher) *MultiPublisher {
	filtered := make([]EventPublisher, 0, len(publishers))
	for _, p := range publishers {
		if p != nil {
			filtered = append(filtered, p)
		}
	}
	return &MultiPublisher{publishers: filtered}
}

// PublishSubtitleDownloaded forwards the event to every wrapped publisher.
func (m *MultiPublisher) PublishSubtitleDownloaded(ctx context.Context, data SubtitleDownloadedData) {
	for _, p := range m.publishers {
		p.PublishSubtitleDownloaded(ctx, data)
	}
}

// PublishSubtitleUpgraded forwards the event to every wrapped publisher.
func (m *MultiPublisher) PublishSubtitleUpgraded(ctx context.Context, data SubtitleUpgradedData) {
	for _, p := range m.publishers {
		p.PublishSubtitleUpgraded(ctx, data)
	}
}

// PublishSubtitleFailed forwards the event to every wrapped publisher.
func (m *MultiPublisher) PublishSubtitleFailed(ctx context.Context, data SubtitleFailedData) {
	for _, p := range m.publishers {
		p.PublishSubtitleFailed(ctx, data)
	}
}

// PublishSearchFailed forwards the event to every wrapped publisher.
func (m *MultiPublisher) PublishSearchFailed(ctx context.Context, data SearchFailedData) {
	for _, p := range m.publishers {
		p.PublishSearchFailed(ctx, data)
	}
}
