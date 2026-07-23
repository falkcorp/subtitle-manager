// file: pkg/subsync/verify.go
// version: 1.0.0
// guid: 6ab465c3-300f-48b8-a82c-416c9c14018e

package subsync

import (
	"context"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/logging"
)

// AnchorMeasurer measures the local subtitle-vs-audio offset at position `at`
// using a window of length `window`. Real implementations extract the audio
// window, transcribe it with Whisper, and compare to the subtitle cues in that
// range (see NewWhisperMeasurer). It is an injected dependency so Verify's
// orchestration is testable without ffmpeg or Whisper.
type AnchorMeasurer func(ctx context.Context, at, window time.Duration) (Anchor, error)

// VerifyOptions configures a drift-verification pass. Zero fields fall back to
// sensible defaults.
type VerifyOptions struct {
	MediaDuration time.Duration // required
	Anchors       int           // sample windows (default 10)
	Window        time.Duration // window length (default 45s)
	SkipStart     time.Duration // skip intro (default 2m)
	SkipEnd       time.Duration // skip credits (default 2m)
	MaxResidualMs float64       // per-anchor tolerance (default 150)
	MinConfidence float64       // ignore weaker anchors (default 0.5)
	DriftSlopeMs  float64       // rate-drift threshold, ms/s (default 5)
}

func (o *VerifyOptions) withDefaults() {
	if o.Anchors <= 0 {
		o.Anchors = 10
	}
	if o.Window <= 0 {
		o.Window = 45 * time.Second
	}
	if o.SkipStart <= 0 {
		o.SkipStart = 2 * time.Minute
	}
	if o.SkipEnd <= 0 {
		o.SkipEnd = 2 * time.Minute
	}
	if o.MaxResidualMs <= 0 {
		o.MaxResidualMs = 150
	}
	if o.MinConfidence <= 0 {
		o.MinConfidence = 0.5
	}
	if o.DriftSlopeMs <= 0 {
		o.DriftSlopeMs = 5
	}
}

// sampleTimes returns Anchors evenly spread across the usable span of the media,
// skipping the configured intro/credits margins. If the margins leave no usable
// span (very short media), it falls back to the whole runtime.
func sampleTimes(o VerifyOptions) []time.Duration {
	start, end := o.SkipStart, o.MediaDuration-o.SkipEnd
	if end <= start {
		start, end = 0, o.MediaDuration
	}
	span := end - start
	if span <= 0 {
		return nil
	}
	step := span / time.Duration(o.Anchors)
	times := make([]time.Duration, 0, o.Anchors)
	for i := 0; i < o.Anchors; i++ {
		at := start + step*time.Duration(i) + step/2
		// Keep the window inside the media.
		if at+o.Window > o.MediaDuration {
			at = o.MediaDuration - o.Window
		}
		if at < 0 {
			at = 0
		}
		times = append(times, at)
	}
	return times
}

// Verify samples anchors across the media, measures each via measure, and
// analyzes the result. Per-anchor measurement errors are logged and skipped so
// one bad window does not fail the whole verification.
func Verify(ctx context.Context, opts VerifyOptions, measure AnchorMeasurer) (DriftReport, error) {
	opts.withDefaults()
	logger := logging.GetLogger("subsync")

	var anchors []Anchor
	for _, at := range sampleTimes(opts) {
		a, err := measure(ctx, at, opts.Window)
		if err != nil {
			logger.Debugf("anchor at %s failed: %v", at, err)
			continue
		}
		anchors = append(anchors, a)
	}
	return Analyze(anchors, opts.MaxResidualMs, opts.MinConfidence, opts.DriftSlopeMs), nil
}
