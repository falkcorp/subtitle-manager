// file: pkg/arrnotify/publisher.go
// version: 1.0.0
// guid: 3e3f1217-8acc-47fe-9320-733f390155e6
// last-edited: 2026-07-25

// Package arrnotify tells Sonarr/Radarr to rescan a title's folder after
// subtitle-manager writes a subtitle for it, so the *arr picks the new file up
// instead of waiting for its own scheduled scan. This is Bazarr's
// notify_sonarr/notify_radarr behaviour (bazarr/sonarr/notify.py,
// bazarr/radarr/notify.py).
//
// The *arr item id is resolved from the media path at event time rather than
// persisted alongside the media item: media_items is written by hand-rolled SQL
// in ~15 places across three backends and there is no migration framework, so
// adding a column is a disproportionately wide change. The trade-off is that a
// rescan is lost (logged, not retried) if the *arr is unreachable at that
// moment.
package arrnotify

import (
	"context"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/arr"
	"github.com/jdfalk/subtitle-manager/pkg/events"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/radarr"
	"github.com/jdfalk/subtitle-manager/pkg/sonarr"
)

// defaultTTL bounds how stale a cached series/movie listing may be. A library
// scan can write hundreds of subtitles, and /api/v3/series is a large response,
// so refetching per subtitle would hammer the *arr for no benefit: a title's
// folder does not move mid-scan.
const defaultTTL = 5 * time.Minute

// Publisher implements events.EventPublisher, issuing an *arr rescan whenever a
// subtitle is written. Failures are logged and swallowed -- a rescan is a
// convenience, and must never fail the subtitle pipeline that produced it.
type Publisher struct {
	Sonarr *sonarr.Client
	Radarr *radarr.Client
	// TTL bounds cached listings; zero means defaultTTL.
	TTL time.Duration

	// now is swappable so tests can drive cache expiry without sleeping.
	now func() time.Time

	mu       sync.Mutex
	series   []sonarr.SeriesRef
	seriesAt time.Time
	movies   []radarr.MovieRef
	moviesAt time.Time
}

// NewPublisher returns a Publisher for the given clients. Either may be nil.
func NewPublisher(s *sonarr.Client, r *radarr.Client) *Publisher {
	return &Publisher{Sonarr: s, Radarr: r, now: time.Now}
}

// NewPublisherFromConfig builds a Publisher from viper config, returning nil
// when neither integration is enabled or both have opted out of rescanning.
// Rescan defaults to on for an enabled integration, matching Bazarr, which
// notifies unconditionally.
func NewPublisherFromConfig() *Publisher {
	var s *sonarr.Client
	if url := arr.BaseURL("integrations.sonarr"); url != "" && rescanEnabled("integrations.sonarr") {
		s = sonarr.NewClient(url, viper.GetString("integrations.sonarr.api_key"))
		s.Filters = arr.FiltersFromViper("integrations.sonarr")
	}
	var r *radarr.Client
	if url := arr.BaseURL("integrations.radarr"); url != "" && rescanEnabled("integrations.radarr") {
		r = radarr.NewClient(url, viper.GetString("integrations.radarr.api_key"))
		r.Filters = arr.FiltersFromViper("integrations.radarr")
	}
	if s == nil && r == nil {
		return nil
	}
	return NewPublisher(s, r)
}

// rescanEnabled reports whether rescan-after-download is on for the given
// integration prefix. Absent config means enabled, so operators who already
// have an *arr configured get parity behaviour without editing their config.
func rescanEnabled(prefix string) bool {
	key := prefix + ".rescan_after_download"
	if !viper.IsSet(key) {
		return true
	}
	return viper.GetBool(key)
}

func (p *Publisher) ttl() time.Duration {
	if p.TTL > 0 {
		return p.TTL
	}
	return defaultTTL
}

func (p *Publisher) clock() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

// PublishSubtitleDownloaded triggers a rescan for the downloaded subtitle's
// media file.
func (p *Publisher) PublishSubtitleDownloaded(ctx context.Context, data events.SubtitleDownloadedData) {
	p.rescan(ctx, data.FilePath)
}

// PublishSubtitleUpgraded triggers a rescan for the upgraded subtitle's media
// file: the replacement is a new file on disk as far as the *arr is concerned.
func (p *Publisher) PublishSubtitleUpgraded(ctx context.Context, data events.SubtitleUpgradedData) {
	p.rescan(ctx, data.FilePath)
}

// PublishSubtitleFailed is a no-op: nothing was written, so there is nothing to
// rescan.
func (p *Publisher) PublishSubtitleFailed(context.Context, events.SubtitleFailedData) {}

// PublishSearchFailed is a no-op for the same reason.
func (p *Publisher) PublishSearchFailed(context.Context, events.SearchFailedData) {}

// rescan resolves videoPath to a Sonarr series or Radarr movie and asks that
// *arr to rescan it. Sonarr is tried first; a path can only belong to one.
func (p *Publisher) rescan(ctx context.Context, videoPath string) {
	if p == nil || videoPath == "" {
		return
	}
	logger := logging.GetLogger("arrnotify")

	// The lock is deliberately held across the listing fetch: during a library
	// scan many downloads land at once, and serialising here collapses what
	// would otherwise be a burst of identical /api/v3 requests into one.
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Sonarr != nil {
		if refs, err := p.seriesRefs(ctx); err != nil {
			logger.Warnf("cannot list Sonarr series for %s: %v", videoPath, err)
		} else if id, ok := sonarr.MatchPath(refs, videoPath); ok {
			if err := p.Sonarr.Rescan(ctx, id); err != nil {
				logger.Warnf("Sonarr rescan of series %d failed: %v", id, err)
			} else {
				logger.Debugf("requested Sonarr rescan of series %d for %s", id, videoPath)
			}
			return
		}
	}
	if p.Radarr != nil {
		if refs, err := p.movieRefs(ctx); err != nil {
			logger.Warnf("cannot list Radarr movies for %s: %v", videoPath, err)
		} else if id, ok := radarr.MatchPath(refs, videoPath); ok {
			if err := p.Radarr.Rescan(ctx, id); err != nil {
				logger.Warnf("Radarr rescan of movie %d failed: %v", id, err)
			} else {
				logger.Debugf("requested Radarr rescan of movie %d for %s", id, videoPath)
			}
			return
		}
	}
	logger.Debugf("no Sonarr/Radarr item owns %s; skipping rescan", videoPath)
}

// seriesRefs returns the cached Sonarr series listing, refreshing it when stale.
// Callers must hold p.mu.
func (p *Publisher) seriesRefs(ctx context.Context) ([]sonarr.SeriesRef, error) {
	if p.series != nil && p.clock().Sub(p.seriesAt) < p.ttl() {
		return p.series, nil
	}
	refs, err := p.Sonarr.Series(ctx)
	if err != nil {
		return nil, err
	}
	p.series, p.seriesAt = refs, p.clock()
	return refs, nil
}

// movieRefs returns the cached Radarr movie listing, refreshing it when stale.
// Callers must hold p.mu.
func (p *Publisher) movieRefs(ctx context.Context) ([]radarr.MovieRef, error) {
	if p.movies != nil && p.clock().Sub(p.moviesAt) < p.ttl() {
		return p.movies, nil
	}
	refs, err := p.Radarr.MovieRefs(ctx)
	if err != nil {
		return nil, err
	}
	p.movies, p.moviesAt = refs, p.clock()
	return refs, nil
}
