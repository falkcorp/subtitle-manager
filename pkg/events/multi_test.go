// file: pkg/events/multi_test.go
// version: 1.0.0
// guid: 7c2e5a91-4f38-4b06-9d1a-3e8c0f2b6a45

package events

import (
	"context"
	"testing"
)

type countingPublisher struct {
	downloaded, upgraded, failed, searchFailed int
}

func (c *countingPublisher) PublishSubtitleDownloaded(context.Context, SubtitleDownloadedData) {
	c.downloaded++
}
func (c *countingPublisher) PublishSubtitleUpgraded(context.Context, SubtitleUpgradedData) {
	c.upgraded++
}
func (c *countingPublisher) PublishSubtitleFailed(context.Context, SubtitleFailedData) { c.failed++ }
func (c *countingPublisher) PublishSearchFailed(context.Context, SearchFailedData)     { c.searchFailed++ }

func TestMultiPublisherFansOut(t *testing.T) {
	a, b := &countingPublisher{}, &countingPublisher{}
	mp := NewMultiPublisher(a, nil, b) // nil is skipped

	ctx := context.Background()
	mp.PublishSubtitleDownloaded(ctx, SubtitleDownloadedData{})
	mp.PublishSubtitleUpgraded(ctx, SubtitleUpgradedData{})
	mp.PublishSubtitleFailed(ctx, SubtitleFailedData{})
	mp.PublishSearchFailed(ctx, SearchFailedData{})

	for name, p := range map[string]*countingPublisher{"a": a, "b": b} {
		if p.downloaded != 1 || p.upgraded != 1 || p.failed != 1 || p.searchFailed != 1 {
			t.Fatalf("publisher %s did not receive each event once: %+v", name, p)
		}
	}
}
