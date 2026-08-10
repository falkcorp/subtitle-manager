// file: pkg/subtitles/stack.go
// version: 1.0.0
// guid: c40d17b9-8e35-4a72-b016-5f92da3c7e64
// last-edited: 2026-08-10

package subtitles

import (
	"sort"
	"time"

	"github.com/asticode/go-astisub"
)

// StackTracks combines two subtitle tracks into one bilingual track, stacking
// the secondary language beneath the primary inside a single cue.
//
// This is the "double subs" shape: one cue carrying both languages, which
// renders as two lines. It is deliberately different from MergeTracks, which
// concatenates and sorts so the two tracks stay separate cues sharing a
// timestamp — a player shows those as competing overlapping subtitles rather
// than as one bilingual line pair.
//
// It is also different from BuildDualSub, which produces the same stacked shape
// but only from a machine translation of a single input. StackTracks is for the
// case where both languages already exist on disk as sidecar files.
//
// Cues are paired by greatest temporal overlap rather than exact timestamp
// equality, because independently sourced tracks are rarely frame-identical.
// Each secondary cue is used at most once. A secondary cue that overlaps
// nothing is kept as its own cue rather than dropped, so no dialogue is lost.
//
// The inputs are not mutated; primary cues are copied before their lines are
// extended.
func StackTracks(primary, secondary []*astisub.Item) []*astisub.Item {
	used := make([]bool, len(secondary))
	out := make([]*astisub.Item, 0, len(primary)+len(secondary))

	for _, p := range primary {
		if p == nil {
			continue
		}
		stacked := *p
		stacked.Lines = append([]astisub.Line(nil), p.Lines...)

		if best := bestOverlap(p, secondary, used); best >= 0 {
			used[best] = true
			stacked.Lines = append(stacked.Lines, secondary[best].Lines...)
		}
		out = append(out, &stacked)
	}

	// Secondary cues with no counterpart still carry dialogue.
	for i, sItem := range secondary {
		if !used[i] && sItem != nil {
			out = append(out, sItem)
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].StartAt < out[j].StartAt
	})
	return out
}

// bestOverlap returns the index of the unused secondary cue sharing the most
// time with p, or -1 when none overlaps it at all.
func bestOverlap(p *astisub.Item, secondary []*astisub.Item, used []bool) int {
	best, bestDur := -1, time.Duration(0)
	for i, s := range secondary {
		if used[i] || s == nil {
			continue
		}
		start := p.StartAt
		if s.StartAt > start {
			start = s.StartAt
		}
		end := p.EndAt
		if s.EndAt < end {
			end = s.EndAt
		}
		if d := end - start; d > bestDur {
			best, bestDur = i, d
		}
	}
	return best
}
