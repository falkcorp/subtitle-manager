// file: pkg/scanner/bilingual_test.go
// version: 1.0.0
// guid: 4f19b8ad-6c72-4e30-95b1-8d02ae64c73f
// last-edited: 2026-08-12

package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

const enSRT = `1
00:00:01,000 --> 00:00:03,000
Good morning.

2
00:00:04,000 --> 00:00:06,000
How are you?
`

const esSRT = `1
00:00:01,000 --> 00:00:03,000
Buenos días.

2
00:00:04,000 --> 00:00:06,000
¿Cómo estás?
`

// writeFixture lays down a video file with two sidecars beside it.
func writeFixture(t *testing.T) (dir, video string) {
	t.Helper()
	dir = t.TempDir()
	video = filepath.Join(dir, "Episode.mkv")
	for path, body := range map[string]string{
		video:                                "not really a video",
		filepath.Join(dir, "Episode.en.srt"): enSRT,
		filepath.Join(dir, "Episode.es.srt"): esSRT,
	} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	return dir, video
}

// TestWriteBilingualPairProducesBothFiles is the point of the feature: a
// profile with two languages must leave a combined subtitle on disk under a
// self-describing name AND under the sentinel tag media servers surface.
func TestWriteBilingualPairProducesBothFiles(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	dir, video := writeFixture(t)

	if err := writeBilingualPair(video, "en", "es"); err != nil {
		t.Fatalf("writeBilingualPair: %v", err)
	}

	combined := filepath.Join(dir, "Episode.en-es.srt")
	sentinel := filepath.Join(dir, "Episode.eo.srt")

	combinedBody, err := os.ReadFile(combined)
	if err != nil {
		t.Fatalf("expected a combined subtitle at %s: %v", combined, err)
	}
	sentinelBody, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("expected a player-visible copy at %s: %v", sentinel, err)
	}
	if string(combinedBody) != string(sentinelBody) {
		t.Error("the sentinel copy does not match the combined file")
	}

	// Both languages must appear, and English must come first in the cue —
	// StackTracks puts the primary on top, which is what "en-es" claims.
	body := string(combinedBody)
	if !strings.Contains(body, "Good morning.") || !strings.Contains(body, "Buenos días.") {
		t.Fatalf("combined subtitle is missing one of the languages:\n%s", body)
	}
	if en, es := strings.Index(body, "Good morning."), strings.Index(body, "Buenos días."); en > es {
		t.Errorf("secondary language appears above the primary; en at %d, es at %d\n%s", en, es, body)
	}

	// Additive: the per-language sidecars survive untouched.
	for name, want := range map[string]string{"Episode.en.srt": enSRT, "Episode.es.srt": esSRT} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("sidecar %s was removed: %v", name, err)
			continue
		}
		if string(got) != want {
			t.Errorf("sidecar %s was modified", name)
		}
	}
}

// TestWriteBilingualPairNeverOverwrites protects an operator's existing files.
// The sentinel name is a real language code, so a genuine Esperanto subtitle
// can legitimately already occupy it — and losing it to a generated file would
// be silent data loss.
func TestWriteBilingualPairNeverOverwrites(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	dir, video := writeFixture(t)

	sentinel := filepath.Join(dir, "Episode.eo.srt")
	if err := os.WriteFile(sentinel, []byte("REAL ESPERANTO SUBTITLE"), 0o644); err != nil {
		t.Fatalf("writing existing sentinel: %v", err)
	}

	// Must not fail the scan: the combined file is still worth writing.
	if err := writeBilingualPair(video, "en", "es"); err != nil {
		t.Fatalf("writeBilingualPair: %v", err)
	}

	got, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("reading sentinel: %v", err)
	}
	if string(got) != "REAL ESPERANTO SUBTITLE" {
		t.Errorf("an existing Esperanto subtitle was overwritten: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "Episode.en-es.srt")); err != nil {
		t.Errorf("the combined file should still be written when only the sentinel is blocked: %v", err)
	}
}

// TestWriteBilingualPairRequiresBothSidecars covers the partial case. A
// profile that only obtained one language must not produce something that
// looks bilingual but carries a single track.
func TestWriteBilingualPairRequiresBothSidecars(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	dir, video := writeFixture(t)

	if err := os.Remove(filepath.Join(dir, "Episode.es.srt")); err != nil {
		t.Fatalf("removing sidecar: %v", err)
	}

	if err := writeBilingualPair(video, "en", "es"); err == nil {
		t.Error("expected an error when a sidecar is missing")
	}
	for _, name := range []string{"Episode.en-es.srt", "Episode.eo.srt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			t.Errorf("%s was written despite a missing source language", name)
		}
	}
}

// TestSentinelLangIsConfigurable keeps this consistent with `dualsub` and
// POST /api/subtitles/stack, which read the same key.
func TestSentinelLangIsConfigurable(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	if got := sentinelLang(); got != "eo" {
		t.Errorf("default sentinel = %q, want %q", got, "eo")
	}
	viper.Set("dualsub.sentinel_language", "zxx")
	if got := sentinelLang(); got != "zxx" {
		t.Errorf("configured sentinel = %q, want %q", got, "zxx")
	}
}
