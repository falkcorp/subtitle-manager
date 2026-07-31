// file: pkg/scanner/format_test.go
// version: 1.0.0
// guid: 1f6b8043-3a29-4e57-9c8b-04d7e21a5f6c
// last-edited: 2026-07-31

package scanner

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/asticode/go-astisub"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"

	providersmocks "github.com/jdfalk/subtitle-manager/pkg/providers/mocks"
)

// srtFromProvider is what a provider hands back: SRT, regardless of what the
// operator wants on disk.
const srtFromProvider = "1\n00:00:01,000 --> 00:00:02,500\nFirst line\n\n" +
	"2\n00:00:03,000 --> 00:00:04,500\nSecond line\n"

// runProcessFile downloads srtFromProvider into a temp dir with
// subtitles.format set to format, and returns the temp dir.
func runProcessFile(t *testing.T, format string) string {
	t.Helper()
	dir := t.TempDir()
	viper.Reset()
	viper.Set("media_directory", dir)
	if format != "" {
		viper.Set("subtitles.format", format)
	}
	t.Cleanup(viper.Reset)

	vid := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	m := providersmocks.NewMockProvider(t)
	m.On("Fetch", mock.Anything, mock.Anything, "en").Return([]byte(srtFromProvider), nil)
	if err := ProcessFile(context.Background(), vid, "en", "test", m, false, nil); err != nil {
		t.Fatalf("process: %v", err)
	}
	return dir
}

// TestProcessFileWritesConfiguredFormat is the end-to-end assertion.
//
// It checks the file name *and* re-parses the contents with the reader for
// that format. Checking only the extension would pass even if SRT bytes were
// written into a ".vtt" file — a file no player accepts, and exactly the kind
// of quiet disagreement between two halves of the system that is invisible
// until something is actually run.
func TestProcessFileWritesConfiguredFormat(t *testing.T) {
	for _, tc := range []struct {
		format string
		want   string
		parse  func([]byte) (*astisub.Subtitles, error)
	}{
		{"", "movie.en.srt", func(b []byte) (*astisub.Subtitles, error) { return astisub.ReadFromSRT(bytes.NewReader(b)) }},
		{"srt", "movie.en.srt", func(b []byte) (*astisub.Subtitles, error) { return astisub.ReadFromSRT(bytes.NewReader(b)) }},
		{"vtt", "movie.en.vtt", func(b []byte) (*astisub.Subtitles, error) { return astisub.ReadFromWebVTT(bytes.NewReader(b)) }},
		{"ass", "movie.en.ass", func(b []byte) (*astisub.Subtitles, error) { return astisub.ReadFromSSA(bytes.NewReader(b)) }},
	} {
		name := tc.format
		if name == "" {
			name = "unset(default)"
		}
		t.Run(name, func(t *testing.T) {
			dir := runProcessFile(t, tc.format)

			out := filepath.Join(dir, tc.want)
			data, err := os.ReadFile(out)
			if err != nil {
				entries, _ := os.ReadDir(dir)
				var names []string
				for _, e := range entries {
					names = append(names, e.Name())
				}
				t.Fatalf("expected %s; directory holds %v", tc.want, names)
			}
			sub, err := tc.parse(data)
			if err != nil {
				t.Fatalf("%s does not parse as %s: %v\n%s", tc.want, tc.format, err, data)
			}
			if len(sub.Items) != 2 {
				t.Errorf("%s holds %d cues, want 2:\n%s", tc.want, len(sub.Items), data)
			}
		})
	}
}

// TestProcessFileBadFormatFallsBackToSRT pins that a bad config entry does not
// cost the download. Refusing to fetch subtitles because a preference string
// is misspelled is the worse of the two failures.
func TestProcessFileBadFormatFallsBackToSRT(t *testing.T) {
	dir := runProcessFile(t, "definitely-not-a-format")
	if _, err := os.Stat(filepath.Join(dir, "movie.en.srt")); err != nil {
		entries, _ := os.ReadDir(dir)
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected a fallback movie.en.srt; directory holds %v", names)
	}
}

// TestProcessFileSkipsExistingInConfiguredFormat pins the "already have it"
// check against the configured container.
//
// The check runs before the download, so it has to look for the extension the
// operator actually uses. Looking for ".srt" while writing ".vtt" means every
// scan re-downloads every subtitle it already has — no visible error, just a
// provider hammered on every pass and an endlessly rewritten library.
func TestProcessFileSkipsExistingInConfiguredFormat(t *testing.T) {
	dir := t.TempDir()
	viper.Reset()
	viper.Set("media_directory", dir)
	viper.Set("subtitles.format", "vtt")
	t.Cleanup(viper.Reset)

	vid := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	existing := filepath.Join(dir, "movie.en.vtt")
	if err := os.WriteFile(existing, []byte("WEBVTT\n\nalready here\n"), 0644); err != nil {
		t.Fatalf("create subtitle: %v", err)
	}

	// No Fetch expectation: reaching the provider at all is the failure.
	m := providersmocks.NewMockProvider(t)
	if err := ProcessFile(context.Background(), vid, "en", "test", m, false, nil); err != nil {
		t.Fatalf("process: %v", err)
	}
	m.AssertExpectations(t)

	data, err := os.ReadFile(existing)
	if err != nil {
		t.Fatalf("read subtitle: %v", err)
	}
	if !bytes.Contains(data, []byte("already here")) {
		t.Errorf("existing subtitle was overwritten: %q", data)
	}
}
