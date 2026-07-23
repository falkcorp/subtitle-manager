// file: pkg/notifications/event_publisher.go
// version: 1.0.0
// guid: a1f4c8e2-5b90-4d37-8e6a-2c9f1b0d7e53
// last-edited: 2026-07-23

package notifications

import (
	"context"
	"fmt"

	"github.com/jdfalk/subtitle-manager/pkg/events"
)

// EventPublisher adapts a notification Service to the events.EventPublisher
// interface so that subtitle download/upgrade/failure events are delivered to
// the operator's configured notification channels (Discord, Telegram, email,
// Apprise). Each event kind can be suppressed via the NotifyOn* flags.
type EventPublisher struct {
	svc *Service
	// NotifyOnDownload, NotifyOnUpgrade and NotifyOnFailure gate which event
	// kinds produce a notification.
	NotifyOnDownload bool
	NotifyOnUpgrade  bool
	NotifyOnFailure  bool
}

// NewEventPublisher returns an events.EventPublisher backed by svc. By default
// download, upgrade and failure events are all delivered.
func NewEventPublisher(svc *Service) *EventPublisher {
	return &EventPublisher{
		svc:              svc,
		NotifyOnDownload: true,
		NotifyOnUpgrade:  true,
		NotifyOnFailure:  true,
	}
}

// send delivers msg through the service, ignoring delivery errors: a failing
// notification channel must never break the subtitle pipeline that produced the
// event.
func (p *EventPublisher) send(ctx context.Context, msg string) {
	if p == nil || p.svc == nil {
		return
	}
	_ = p.svc.Send(ctx, msg)
}

// PublishSubtitleDownloaded notifies that a subtitle was downloaded.
func (p *EventPublisher) PublishSubtitleDownloaded(ctx context.Context, data events.SubtitleDownloadedData) {
	if !p.NotifyOnDownload {
		return
	}
	p.send(ctx, fmt.Sprintf("✅ Downloaded %s subtitle for %s (provider %s)",
		data.Language, data.FilePath, data.Provider))
}

// PublishSubtitleUpgraded notifies that a subtitle was upgraded.
func (p *EventPublisher) PublishSubtitleUpgraded(ctx context.Context, data events.SubtitleUpgradedData) {
	if !p.NotifyOnUpgrade {
		return
	}
	p.send(ctx, fmt.Sprintf("⬆️ Upgraded %s subtitle for %s (provider %s)",
		data.Language, data.FilePath, data.NewProvider))
}

// PublishSubtitleFailed notifies that a subtitle download failed.
func (p *EventPublisher) PublishSubtitleFailed(ctx context.Context, data events.SubtitleFailedData) {
	if !p.NotifyOnFailure {
		return
	}
	p.send(ctx, fmt.Sprintf("❌ Subtitle download failed for %s (%s): %s",
		data.FilePath, data.Language, data.Error))
}

// PublishSearchFailed is intentionally a no-op: search failures are noisy and
// usually transient, so they are not surfaced as notifications.
func (p *EventPublisher) PublishSearchFailed(ctx context.Context, data events.SearchFailedData) {}
