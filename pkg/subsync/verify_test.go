// file: pkg/subsync/verify_test.go
// version: 1.0.0
// guid: c0acb0d8-9f74-4748-99ec-36b3e0f1990a

package subsync

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/asticode/go-astisub"
)

func items(starts ...time.Duration) []*astisub.Item {
	out := make([]*astisub.Item, len(starts))
	for i, s := range starts {
		out[i] = &astisub.Item{StartAt: s}
	}
	return out
}

func TestMeasureWindowInSync(t *testing.T) {
	at := 10 * time.Minute
	// spoken cues relative to window start; subtitle cues on absolute timeline.
	spoken := items(0, 5*time.Second, 12*time.Second)
	sub := items(at+0, at+5*time.Second, at+12*time.Second)
	a := measureWindow(sub, spoken, at)
	if math.Abs(a.OffsetMs) > 1 {
		t.Fatalf("expected ~0 offset, got %.1f", a.OffsetMs)
	}
	if a.Confidence != 1 {
		t.Fatalf("expected confidence 1, got %.2f", a.Confidence)
	}
}

func TestMeasureWindowConstantOffset(t *testing.T) {
	at := 20 * time.Minute
	spoken := items(0, 4*time.Second, 9*time.Second)
	// subtitle cues are 600ms later than the spoken words.
	off := 600 * time.Millisecond
	sub := items(at+off, at+4*time.Second+off, at+9*time.Second+off)
	a := measureWindow(sub, spoken, at)
	if math.Abs(a.OffsetMs-600) > 1 {
		t.Fatalf("expected ~600ms offset, got %.1f", a.OffsetMs)
	}
	if a.Confidence != 1 {
		t.Fatalf("expected confidence 1, got %.2f", a.Confidence)
	}
}

func TestMeasureWindowNoSpoken(t *testing.T) {
	a := measureWindow(items(1*time.Second), nil, 5*time.Minute)
	if a.Confidence != 0 {
		t.Fatalf("expected confidence 0 for silent window, got %.2f", a.Confidence)
	}
}

func TestMeasureWindowUnmatchedIsLowConfidence(t *testing.T) {
	at := 0 * time.Second
	spoken := items(1 * time.Second)
	// nearest subtitle cue is 10s away — beyond maxMatchMs, so no match.
	sub := items(11 * time.Second)
	a := measureWindow(sub, spoken, at)
	if a.Confidence != 0 {
		t.Fatalf("expected confidence 0 when nothing matches, got %.2f", a.Confidence)
	}
}

// TestVerifyOrchestration drives Verify with a fake measurer that reports a
// framerate-style drift, and checks the report reflects it.
func TestVerifyOrchestration(t *testing.T) {
	opts := VerifyOptions{MediaDuration: 100 * time.Minute}
	r := 23.976 / 25.0
	slope := (r - 1) * 1000 // ms/s
	fake := func(_ context.Context, at, _ time.Duration) (Anchor, error) {
		return Anchor{At: at, OffsetMs: slope * at.Seconds(), Confidence: 1}, nil
	}
	rep, err := Verify(context.Background(), opts, fake)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if rep.Used == 0 {
		t.Fatal("expected anchors to be measured")
	}
	if !rep.RateDrift {
		t.Fatalf("expected rate drift, got %+v", rep)
	}
}

// TestVerifySkipsErroredAnchors ensures a measurer error on some anchors doesn't
// fail the whole pass.
func TestVerifySkipsErroredAnchors(t *testing.T) {
	opts := VerifyOptions{MediaDuration: 60 * time.Minute, Anchors: 6}
	calls := 0
	fake := func(_ context.Context, at, _ time.Duration) (Anchor, error) {
		calls++
		if calls%2 == 0 {
			return Anchor{}, context.DeadlineExceeded
		}
		return Anchor{At: at, OffsetMs: 10, Confidence: 1}, nil
	}
	rep, err := Verify(context.Background(), opts, fake)
	if err != nil {
		t.Fatalf("verify should not fail on per-anchor errors: %v", err)
	}
	if rep.Used == 0 {
		t.Fatal("expected some anchors to survive")
	}
}
