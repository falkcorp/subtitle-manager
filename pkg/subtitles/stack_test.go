// file: pkg/subtitles/stack_test.go
// version: 1.0.0
// guid: 7b2e94a1-3c05-4d68-91af-2e6d70c5b843
// last-edited: 2026-08-10

package subtitles

import (
	"testing"
	"time"

	"github.com/asticode/go-astisub"
)

// item builds a subtitle cue with one line of text.
func item(start, end time.Duration, text string) *astisub.Item {
	return &astisub.Item{
		StartAt: start,
		EndAt:   end,
		Lines:   []astisub.Line{{Items: []astisub.LineItem{{Text: text}}}},
	}
}

// lines returns each rendered line of a cue, so a stacked cue can be asserted
// on directly rather than through a formatted SRT blob.
func lines(it *astisub.Item) []string {
	out := make([]string, 0, len(it.Lines))
	for _, l := range it.Lines {
		s := ""
		for _, li := range l.Items {
			s += li.Text
		}
		out = append(out, s)
	}
	return out
}

const s = time.Second

// StackTracks exists because neither existing path produces a bilingual file
// from two subtitles the user already has: MergeTracks interleaves them into
// separate cues sharing a timestamp (which competing renderers overlap), and
// BuildDualSub stacks correctly but only from a machine translation of a single
// input.
func TestStackTracksCombinesAlignedCuesIntoOne(t *testing.T) {
	primary := []*astisub.Item{
		item(1*s, 4*s, "My name is Walter Hartwell White."),
		item(5*s, 9*s, "I live at 308 Negra Arroyo Lane."),
	}
	secondary := []*astisub.Item{
		item(1*s, 4*s, "Me llamo Walter Hartwell White."),
		item(5*s, 9*s, "Vivo en 308 Negra Arroyo Lane."),
	}

	got := StackTracks(primary, secondary)

	if len(got) != 2 {
		t.Fatalf("expected 2 stacked cues, got %d", len(got))
	}
	want := []string{"My name is Walter Hartwell White.", "Me llamo Walter Hartwell White."}
	if g := lines(got[0]); len(g) != 2 || g[0] != want[0] || g[1] != want[1] {
		t.Errorf("cue 0 lines = %q, want %q", g, want)
	}
	if got[0].StartAt != 1*s || got[0].EndAt != 4*s {
		t.Errorf("cue 0 timing = %v-%v, want 1s-4s", got[0].StartAt, got[0].EndAt)
	}
}

// Real tracks are rarely frame-identical, so alignment must tolerate drift and
// pair cues by overlap rather than by exact timestamp equality.
func TestStackTracksAlignsImperfectlyTimedCues(t *testing.T) {
	primary := []*astisub.Item{item(1000*time.Millisecond, 4000*time.Millisecond, "Hello there.")}
	secondary := []*astisub.Item{item(1200*time.Millisecond, 4300*time.Millisecond, "Hola.")}

	got := StackTracks(primary, secondary)

	if len(got) != 1 {
		t.Fatalf("expected the offset cue to stack, got %d cues", len(got))
	}
	if g := lines(got[0]); len(g) != 2 || g[1] != "Hola." {
		t.Errorf("lines = %q, want the translation stacked beneath", g)
	}
}

// A cue with no counterpart must survive as its own cue. Dropping it would
// silently lose dialogue, which is worse than the interleaving this replaces.
func TestStackTracksKeepsUnmatchedSecondaryCues(t *testing.T) {
	primary := []*astisub.Item{item(1*s, 2*s, "First.")}
	secondary := []*astisub.Item{
		item(1*s, 2*s, "Primero."),
		item(30*s, 31*s, "Suelto."),
	}

	got := StackTracks(primary, secondary)

	if len(got) != 2 {
		t.Fatalf("expected 2 cues (1 stacked + 1 orphan), got %d", len(got))
	}
	if g := lines(got[0]); len(g) != 2 {
		t.Errorf("first cue should be stacked, got %q", g)
	}
	if g := lines(got[1]); len(g) != 1 || g[0] != "Suelto." {
		t.Errorf("orphan cue = %q, want [Suelto.]", g)
	}
	if got[1].StartAt != 30*s {
		t.Errorf("orphan kept wrong timing %v", got[1].StartAt)
	}
}

// One secondary cue must not be pasted onto several primary cues.
func TestStackTracksConsumesEachSecondaryCueOnce(t *testing.T) {
	primary := []*astisub.Item{
		item(1000*time.Millisecond, 2000*time.Millisecond, "One."),
		item(2000*time.Millisecond, 3000*time.Millisecond, "Two."),
	}
	secondary := []*astisub.Item{item(1000*time.Millisecond, 3000*time.Millisecond, "Uno y dos.")}

	got := StackTracks(primary, secondary)

	stacked := 0
	for _, it := range got {
		if len(lines(it)) > 1 {
			stacked++
		}
	}
	if stacked != 1 {
		t.Errorf("expected the secondary cue to be used once, it stacked onto %d cues", stacked)
	}
}
