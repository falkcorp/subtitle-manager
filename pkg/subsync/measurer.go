// file: pkg/subsync/measurer.go
// version: 1.0.0
// guid: 42201e57-1e84-4b64-ba44-6ac2c8a1e15d

package subsync

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"github.com/asticode/go-astisub"

	"github.com/jdfalk/subtitle-manager/pkg/audio"
	"github.com/jdfalk/subtitle-manager/pkg/transcriber"
)

// TranscribeFunc transcribes an audio file to SRT bytes for the given language.
// Injecting it lets the measurer target either a user-provided OpenAI-compatible
// server or a self-hosted ASR webservice, and makes the glue testable.
type TranscribeFunc func(ctx context.Context, audioPath, lang string) ([]byte, error)

// maxMatchMs is how close (in ms) a transcribed cue's start must be to a
// subtitle cue's start for the two to be considered the same line. Beyond this
// the match is treated as spurious. It bounds how large an offset a single
// anchor can measure; larger drift is still caught at earlier anchors where the
// accumulated offset is smaller. See docs/WHISPER_PIPELINE_DECISIONS.md (W4).
const maxMatchMs = 3000.0

// measureWindow computes the local offset between subtitle cues and transcribed
// ("spoken") cues for a window that starts at `at` in the media. Spoken cue
// timestamps are relative to the window, so `at` is added to place them on the
// media timeline. For each spoken cue it finds the nearest subtitle cue by start
// time (within maxMatchMs) and records subStart−spokenStart; the anchor offset is
// the median of those diffs and confidence is the fraction of spoken cues matched.
//
// Pure function: unit-testable without ffmpeg or Whisper.
func measureWindow(subCues, spoken []*astisub.Item, at time.Duration) Anchor {
	if len(spoken) == 0 {
		return Anchor{At: at, Confidence: 0}
	}
	var diffs []float64
	matched := 0
	for _, sp := range spoken {
		abs := at + sp.StartAt
		bestDiff := math.MaxFloat64
		var bestSigned float64
		for _, sc := range subCues {
			d := float64(sc.StartAt-abs) / float64(time.Millisecond)
			if math.Abs(d) < bestDiff {
				bestDiff = math.Abs(d)
				bestSigned = d
			}
		}
		if bestDiff <= maxMatchMs {
			diffs = append(diffs, bestSigned)
			matched++
		}
	}
	if len(diffs) == 0 {
		return Anchor{At: at, Confidence: 0}
	}
	return Anchor{At: at, OffsetMs: median(diffs), Confidence: float64(matched) / float64(len(spoken))}
}

// median returns the median of xs (xs is copied, not mutated).
func median(xs []float64) float64 {
	c := append([]float64(nil), xs...)
	sort.Float64s(c)
	n := len(c)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return c[n/2]
	}
	return (c[n/2-1] + c[n/2]) / 2
}

// NewWhisperMeasurer builds an AnchorMeasurer that, per window, extracts the
// audio with ffmpeg, transcribes it, and measures the offset against the
// subtitle cues loaded from subtitlePath. If transcribe is nil, it defaults to
// transcriber.WhisperTranscribe with whisperKey (OpenAI-compatible; point its
// base URL at a self-hosted server for local Whisper).
func NewWhisperMeasurer(mediaPath, subtitlePath, lang, whisperKey string, audioTrack int, transcribe TranscribeFunc) (AnchorMeasurer, error) {
	sub, err := astisub.OpenFile(subtitlePath)
	if err != nil {
		return nil, fmt.Errorf("open subtitle: %w", err)
	}
	if transcribe == nil {
		transcribe = func(_ context.Context, audioPath, l string) ([]byte, error) {
			return transcriber.WhisperTranscribe(audioPath, l, whisperKey)
		}
	}
	return func(ctx context.Context, at, window time.Duration) (Anchor, error) {
		wav, err := audio.ExtractTrackWithDuration(mediaPath, audioTrack, at, window)
		if err != nil {
			return Anchor{}, fmt.Errorf("extract audio window: %w", err)
		}
		defer os.Remove(wav)

		data, err := transcribe(ctx, wav, lang)
		if err != nil {
			return Anchor{}, fmt.Errorf("transcribe window: %w", err)
		}
		spoken, err := astisub.ReadFromSRT(bytes.NewReader(data))
		if err != nil {
			return Anchor{}, fmt.Errorf("parse transcription: %w", err)
		}
		return measureWindow(sub.Items, spoken.Items, at), nil
	}, nil
}
