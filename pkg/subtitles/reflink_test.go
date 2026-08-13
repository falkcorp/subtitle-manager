// file: pkg/subtitles/reflink_test.go
// version: 2.0.0
// guid: 1d7c4f80-2e39-4a56-b8c1-90fa35d27e64
// last-edited: 2026-08-12

package subtitles

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestCloneFileProducesAnIndependentCopy is the contract every backend must
// meet, whether the filesystem gave us a real reflink or we fell back to a
// byte copy: the destination has the source's contents, and writing to one
// does not change the other.
//
// The independence check is what rules out a hardlink. A hardlink would pass a
// naive content comparison while quietly sharing an inode, so rewriting the
// bilingual file in place would also rewrite its sentinel twin.
func TestCloneFileProducesAnIndependentCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Episode.en-es.srt")
	dst := filepath.Join(dir, "Episode.eo.srt")

	want := []byte("1\n00:00:01,000 --> 00:00:02,000\nHello\nHola\n\n")
	if err := os.WriteFile(src, want, 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}

	root := openRoot(t, dir)
	if err := CloneFileIn(root, "Episode.en-es.srt", "Episode.eo.srt"); err != nil {
		t.Fatalf("CloneFileIn: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading clone: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("clone contents = %q, want %q", got, want)
	}

	// Replace the destination and confirm the source is untouched.
	if err := os.WriteFile(dst, []byte("REPLACED"), 0o644); err != nil {
		t.Fatalf("rewriting clone: %v", err)
	}
	after, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("re-reading source: %v", err)
	}
	if string(after) != string(want) {
		t.Errorf("writing the clone modified the source; they share storage.\ngot:  %q\nwant: %q", after, want)
	}
}

// TestCloneFileRefusesToClobber protects the sidecars the operator already has.
// Bilingual output is additive — it must never overwrite an existing subtitle,
// including a genuine Esperanto track that happens to occupy the sentinel name.
func TestCloneFileRefusesToClobber(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Episode.en-es.srt")
	dst := filepath.Join(dir, "Episode.eo.srt")

	if err := os.WriteFile(src, []byte("new"), 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}
	if err := os.WriteFile(dst, []byte("PRE-EXISTING"), 0o644); err != nil {
		t.Fatalf("writing destination: %v", err)
	}

	root := openRoot(t, dir)
	if err := CloneFileIn(root, "Episode.en-es.srt", "Episode.eo.srt"); err == nil {
		t.Error("CloneFileIn overwrote an existing file; it must refuse")
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading destination: %v", err)
	}
	if string(got) != "PRE-EXISTING" {
		t.Errorf("existing file was modified: %q", got)
	}
}

// TestReflinkBackendIsWired calls the platform backend directly.
//
// Without this, a backend that errored on every call would be invisible:
// CloneFile would quietly fall back to copying and every other test here would
// still pass. This cannot assert that reflinking *succeeds* — CI runs on ext4
// or overlayfs, where it legitimately cannot — but it does assert that any
// failure is the expected "unsupported" signal rather than a real fault, and
// reports which path was taken so the log shows whether it was ever exercised.
func TestReflinkBackendIsWired(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.srt")
	dst := filepath.Join(dir, "b.srt")
	if err := os.WriteFile(src, []byte("subtitle"), 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}

	root := openRoot(t, dir)
	switch err := reflink(root, "a.srt", "b.srt"); {
	case err == nil:
		t.Log("reflink succeeded: this filesystem shares blocks")
		got, readErr := os.ReadFile(dst)
		if readErr != nil {
			t.Fatalf("reflink reported success but produced no readable file: %v", readErr)
		}
		if string(got) != "subtitle" {
			t.Errorf("reflinked contents = %q, want %q", got, "subtitle")
		}
	case errors.Is(err, errReflinkUnsupported):
		t.Log("reflink unsupported on this filesystem; CloneFile falls back to copying")
		if _, statErr := os.Stat(dst); statErr == nil {
			t.Error("an unsupported reflink left a file behind; the copy fallback's O_EXCL will fail")
		}
	default:
		t.Errorf("reflink returned an unexpected error: %v", err)
	}
}

// TestCloneFileMissingSource reports a real error rather than silently
// creating an empty destination.
func TestCloneFileMissingSource(t *testing.T) {
	dir := t.TempDir()
	root := openRoot(t, dir)
	if err := CloneFileIn(root, "nope.srt", "out.srt"); err == nil {
		t.Fatal("expected an error for a missing source")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "out.srt")); statErr == nil {
		t.Error("a destination was created despite the source being missing")
	}
}

// openRoot confines every operation in these tests beneath dir, the same way
// the scanner does when it writes beside a media file.
func openRoot(t *testing.T, dir string) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatalf("opening root %s: %v", dir, err)
	}
	t.Cleanup(func() { root.Close() })
	return root
}

// TestCloneFileInRefusesToEscapeTheRoot is why this API takes a *os.Root and
// base names rather than two paths.
//
// The earlier version took paths and passed them straight to os.Open and
// os.OpenFile, which CodeQL flagged as go/path-injection across twelve
// high-severity alerts: a user-derived media path reached a file operation with
// no confinement. os.Root refuses a traversal at the kernel-facing API, so this
// cannot be talked past with string checks.
func TestCloneFileInRefusesToEscapeTheRoot(t *testing.T) {
	base := t.TempDir()
	inside := filepath.Join(base, "media")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inside, "a.srt"), []byte("subtitle"), 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}

	root := openRoot(t, inside)
	if err := CloneFileIn(root, "a.srt", filepath.Join("..", "escaped.srt")); err == nil {
		t.Error("a traversing destination was accepted")
	}
	if _, err := os.Stat(filepath.Join(base, "escaped.srt")); err == nil {
		t.Error("a file was written outside the root")
	}
}
