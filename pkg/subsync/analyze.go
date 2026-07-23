// file: pkg/subsync/analyze.go
// version: 1.0.0
// guid: 4a285ebc-78ec-4122-a588-f91307b65522

// Package subsync verifies that a subtitle stays aligned with a media file's
// audio across its runtime — detecting both a constant offset and linear
// "speed up / slow down" drift (e.g. a 23.976↔25 fps mismatch) — rather than
// only correcting a single global offset like pkg/syncer does.
package subsync

import (
	"fmt"
	"math"
	"time"
)

// Anchor is a measured local timing error between a subtitle and the audio at a
// point in the media. OffsetMs is (subtitle − audio) in milliseconds: positive
// means the subtitle cue is later than the spoken words. Confidence is 0..1.
type Anchor struct {
	At         time.Duration
	OffsetMs   float64
	Confidence float64
}

// DriftReport summarizes subtitle-vs-audio alignment across anchors.
type DriftReport struct {
	Anchors []Anchor // all measured anchors (including low-confidence)
	Used    int      // anchors that passed the confidence filter

	// SlopeMsPerSec is drift: ms of added offset per second of runtime.
	SlopeMsPerSec float64
	// InterceptMs is the fitted constant offset at t=0.
	InterceptMs float64
	// MaxResidualMs / RMSEMs describe how well a linear model fits (fit quality).
	MaxResidualMs float64
	RMSEMs        float64

	InSync         bool   // within tolerance at every usable anchor
	ConstantOffset bool   // significant offset, ~zero slope (a plain shift fixes it)
	RateDrift      bool   // significant slope (speed up / slow down)
	LikelyCause    string // human-readable cause when RateDrift is set
}

// Analyze fits offset = slope·t + intercept over the anchors (t in seconds,
// offset in ms) and classifies the result.
//
//   - maxResidualMs: per-anchor tolerance; InSync requires every usable anchor's
//     absolute offset to be within this.
//   - minConfidence: anchors below this are ignored (noisy transcription, music,
//     silence).
//   - driftSlopeThresholdMs: |slope| at or above this (ms per second) is treated
//     as rate drift rather than a constant offset.
//
// It is a pure function so the classification is unit-testable without audio or
// Whisper.
func Analyze(anchors []Anchor, maxResidualMs, minConfidence, driftSlopeThresholdMs float64) DriftReport {
	rep := DriftReport{Anchors: anchors}

	var ts, offs []float64
	var maxAbsOffset float64
	for _, a := range anchors {
		if a.Confidence < minConfidence {
			continue
		}
		ts = append(ts, a.At.Seconds())
		offs = append(offs, a.OffsetMs)
		if math.Abs(a.OffsetMs) > maxAbsOffset {
			maxAbsOffset = math.Abs(a.OffsetMs)
		}
	}
	rep.Used = len(ts)
	if rep.Used == 0 {
		return rep
	}
	if rep.Used == 1 {
		rep.InterceptMs = offs[0]
		rep.InSync = math.Abs(offs[0]) <= maxResidualMs
		rep.ConstantOffset = !rep.InSync
		return rep
	}

	n := float64(len(ts))
	var sx, sy, sxx, sxy float64
	for i := range ts {
		sx += ts[i]
		sy += offs[i]
		sxx += ts[i] * ts[i]
		sxy += ts[i] * offs[i]
	}
	denom := n*sxx - sx*sx
	if denom == 0 {
		// All anchors at the same timestamp: cannot fit a slope; treat the mean
		// offset as a constant offset.
		rep.InterceptMs = sy / n
		rep.InSync = maxAbsOffset <= maxResidualMs
		rep.ConstantOffset = !rep.InSync
		return rep
	}

	slope := (n*sxy - sx*sy) / denom
	intercept := (sy - slope*sx) / n
	rep.SlopeMsPerSec = slope
	rep.InterceptMs = intercept

	var sumSq float64
	for i := range ts {
		r := offs[i] - (slope*ts[i] + intercept)
		if a := math.Abs(r); a > rep.MaxResidualMs {
			rep.MaxResidualMs = a
		}
		sumSq += r * r
	}
	rep.RMSEMs = math.Sqrt(sumSq / n)

	rep.InSync = maxAbsOffset <= maxResidualMs
	rep.RateDrift = math.Abs(slope) >= driftSlopeThresholdMs
	rep.ConstantOffset = !rep.RateDrift && math.Abs(intercept) > maxResidualMs
	if rep.RateDrift {
		rep.LikelyCause = likelyCause(slope)
	}
	return rep
}

// likelyCause names a probable framerate mismatch for a given drift slope
// (ms per second), or returns a generic description. A rate factor r means the
// offset grows by (r−1)·1000 ms per second.
func likelyCause(slopeMsPerSec float64) string {
	r := 1 + slopeMsPerSec/1000
	cands := []struct {
		ratio float64
		name  string
	}{
		{25.0 / 24.0, "framerate 24→25 (PAL speed-up)"},
		{24.0 / 25.0, "framerate 25→24"},
		{25.0 / 23.976, "framerate 23.976→25"},
		{23.976 / 25.0, "framerate 25→23.976"},
		{30.0 / 29.97, "framerate 29.97→30"},
		{29.97 / 30.0, "framerate 30→29.97"},
	}
	best := ""
	bestDiff := 0.005 // ratio tolerance
	for _, c := range cands {
		if d := math.Abs(r - c.ratio); d < bestDiff {
			bestDiff = d
			best = c.name
		}
	}
	if best == "" {
		return fmt.Sprintf("linear drift ~%.1f ms/s (rate %.4f)", slopeMsPerSec, r)
	}
	return best
}
