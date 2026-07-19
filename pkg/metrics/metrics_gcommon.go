//go:build gcommonmetrics

// file: pkg/metrics/metrics_gcommon.go
// version: 1.0.0
// guid: 0d1e2f3a-4b5c-6d7e-8f9a-0b1c2d3e4f5g

package metrics

import (
	"context"

	gmetrics "github.com/falkcorp/gcommon/v2/pkg/metricspb/v2"
)

var (
	// Provider exposes the gcommon metrics provider.
	Provider gmetrics.Provider

	ProviderRequests    gmetrics.Counter
	TranslationRequests gmetrics.Counter
	APIRequests         gmetrics.Counter
	RequestDuration     gmetrics.Histogram
	ActiveSessions      gmetrics.Gauge
	SubtitleDownloads   gmetrics.Counter
)

func InitGcommon() error {
	var err error
	Provider, err = gmetrics.NewProvider(gmetrics.Config{
		Enabled:  true,
		Provider: "prometheus",
	})
	if err != nil {
		return err
	}
	ProviderRequests = Provider.Counter("subtitle_manager_provider_requests_total")
	TranslationRequests = Provider.Counter("subtitle_manager_translation_requests_total")
	APIRequests = Provider.Counter("subtitle_manager_api_requests_total")
	RequestDuration = Provider.Histogram("subtitle_manager_request_duration_seconds")
	ActiveSessions = Provider.Gauge("subtitle_manager_active_sessions")
	SubtitleDownloads = Provider.Counter("subtitle_manager_subtitle_downloads_total")
	return Provider.Start(context.Background())
}
