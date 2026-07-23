// file: pkg/providers/embedded/embedded_test.go
// version: 2.0.0
// guid: bc3bc260-6f2f-4daf-bba0-ed75a93d271d

package embedded

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLanguageMatches(t *testing.T) {
	cases := []struct {
		stream, want string
		match        bool
	}{
		{"eng", "en", true},
		{"en", "eng", true},
		{"eng", "eng", true},
		{"fre", "en", false},
		{"", "en", false},
		{"en", "", false},
		{"spa", "es", true},
	}
	for _, c := range cases {
		if got := languageMatches(c.stream, c.want); got != c.match {
			t.Errorf("languageMatches(%q,%q)=%v want %v", c.stream, c.want, got, c.match)
		}
	}
}

// TestFetchExtractsEmbedded muxes an SRT track into an MKV with ffmpeg and
// verifies Fetch extracts it. Skips when ffmpeg is unavailable.
func TestFetchExtractsEmbedded(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()
	srt := filepath.Join(dir, "in.srt")
	const content = "1\n00:00:00,500 --> 00:00:01,500\nHello embedded\n"
	if err := os.WriteFile(srt, []byte(content), 0o644); err != nil {
		t.Fatalf("write srt: %v", err)
	}
	mkv := filepath.Join(dir, "movie.mkv")
	// 1s black video + the SRT muxed as a subtitle track tagged eng.
	mux := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "color=c=black:s=64x64:d=1",
		"-i", srt, "-c:v", "mpeg4", "-c:s", "srt", "-metadata:s:s:0", "language=eng", mkv)
	if out, err := mux.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg mux unavailable in this environment: %v: %s", err, out)
	}

	data, err := New().Fetch(context.Background(), mkv, "en")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !strings.Contains(string(data), "Hello embedded") {
		t.Fatalf("extracted subtitle missing content: %q", data)
	}
}
