// file: pkg/subsync/analyze_test.go
// version: 1.0.0
// guid: 6edb8fe6-6267-4627-b640-b12ba5a1ef86

package subsync

import (
	"math"
	"testing"
	"time"
)

// anchorsFrom builds anchors at evenly spaced timestamps whose offset follows
// offset(t) = slopeMsPerSec*t + interceptMs, all at full confidence.
func anchorsFrom(n int, spacing time.Duration, slopeMsPerSec, interceptMs float64) []Anchor {
	as := make([]Anchor, n)
	for i := 0; i < n; i++ {
		at := time.Duration(i) * spacing
		as[i] = Anchor{At: at, OffsetMs: slopeMsPerSec*at.Seconds() + interceptMs, Confidence: 1}
	}
	return as
}

func TestAnalyzeInSync(t *testing.T) {
	rep := Analyze(anchorsFrom(8, 5*time.Minute, 0, 20), 150, 0.5, 5)
	if !rep.InSync {
		t.Fatalf("expected in sync, got %+v", rep)
	}
	if rep.RateDrift || rep.ConstantOffset {
		t.Fatalf("expected no drift/offset flags, got %+v", rep)
	}
}

func TestAnalyzeConstantOffset(t *testing.T) {
	rep := Analyze(anchorsFrom(8, 5*time.Minute, 0, 900), 150, 0.5, 5)
	if rep.InSync {
		t.Fatalf("900ms offset should not be in sync")
	}
	if !rep.ConstantOffset || rep.RateDrift {
		t.Fatalf("expected constant offset, got %+v", rep)
	}
	if math.Abs(rep.InterceptMs-900) > 1 {
		t.Fatalf("intercept ~900 expected, got %.1f", rep.InterceptMs)
	}
	if math.Abs(rep.SlopeMsPerSec) > 0.01 {
		t.Fatalf("slope ~0 expected, got %.4f", rep.SlopeMsPerSec)
	}
}

func TestAnalyzeRateDriftFramerate(t *testing.T) {
	// Subtitle timed for 25 fps played at 23.976 → rate 23.976/25, slope = (r-1)*1000.
	r := 23.976 / 25.0
	slope := (r - 1) * 1000 // ≈ -40.96 ms/s
	rep := Analyze(anchorsFrom(10, 6*time.Minute, slope, 0), 150, 0.5, 5)
	if !rep.RateDrift {
		t.Fatalf("expected rate drift for slope %.2f, got %+v", slope, rep)
	}
	if rep.InSync {
		t.Fatalf("rate-drifting subtitle should not be in sync")
	}
	if math.Abs(rep.SlopeMsPerSec-slope) > 0.5 {
		t.Fatalf("slope %.2f expected, got %.2f", slope, rep.SlopeMsPerSec)
	}
	if rep.LikelyCause == "" || rep.LikelyCause[:9] != "framerate" {
		t.Fatalf("expected a framerate cause, got %q", rep.LikelyCause)
	}
}

func TestAnalyzeConfidenceFilter(t *testing.T) {
	as := anchorsFrom(4, 5*time.Minute, 0, 10)
	// Add a wildly wrong, low-confidence anchor that must be ignored.
	as = append(as, Anchor{At: 30 * time.Minute, OffsetMs: 9000, Confidence: 0.1})
	rep := Analyze(as, 150, 0.5, 5)
	if rep.Used != 4 {
		t.Fatalf("low-confidence anchor should be filtered; used=%d", rep.Used)
	}
	if !rep.InSync {
		t.Fatalf("expected in sync after filtering noise, got %+v", rep)
	}
}

func TestAnalyzeEmpty(t *testing.T) {
	rep := Analyze(nil, 150, 0.5, 5)
	if rep.Used != 0 || rep.InSync {
		t.Fatalf("empty anchors should yield Used=0 and not in-sync, got %+v", rep)
	}
}
