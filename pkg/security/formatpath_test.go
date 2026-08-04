// file: pkg/security/formatpath_test.go
// version: 1.1.0
// guid: 6d0a94f1-58c3-4e27-b91f-72e5c0846a3b
// last-edited: 2026-08-04

package security

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// TestValidateSubtitleOutputPathWithFormatRejectsBadFormat pins the format
// allowlist. The value becomes a file-name suffix, so anything off the list
// must be refused rather than sanitised into a path.
func TestValidateSubtitleOutputPathWithFormatRejectsBadFormat(t *testing.T) {
	const dir = "/opt/media"
	viper.Set("media_directory", dir)
	defer viper.Reset()

	video := dir + "/movie.mkv"
	for _, bad := range []string{"../../etc/passwd", "srt/../../evil", "exe", "sub", "srt.exe"} {
		if got, err := ValidateSubtitleOutputPathWithFormat(video, "en", false, bad); err == nil {
			t.Errorf("format %q accepted, produced %q", bad, got)
		}
	}
	for _, ok := range []string{"srt", "vtt", "ass", ".vtt", "VTT", ""} {
		if _, err := ValidateSubtitleOutputPathWithFormat(video, "en", false, ok); err != nil {
			t.Errorf("format %q rejected: %v", ok, err)
		}
	}
}

// TestValidateSubtitleOutputPathWithFormatConfinesPath pins that the video
// path cannot escape the configured media directories.
//
// This is what makes the os.Stat in webserver.writtenSubtitlePath safe, and
// CodeQL flags that call because it cannot see this constraint. Pinning it
// here means the assumption breaks a test rather than breaking quietly.
func TestValidateSubtitleOutputPathWithFormatConfinesPath(t *testing.T) {
	// A fixed path rather than t.TempDir(): the temp-directory escape hatch in
	// ValidateAndSanitizePath is open inside a test binary, so a temp-rooted
	// media directory would not exercise the confinement being asserted here.
	// (The hatch is no longer unconditional — it is closed in production. See
	// allowTempDirPaths in security.go and tempdir_gate_test.go.)
	const dir = "/opt/media"
	viper.Set("media_directory", dir)
	defer viper.Reset()

	for _, escape := range []string{
		dir + "/../../etc/passwd.mkv",
		dir + "/../secrets/other.mkv",
		"../../../../etc/shadow.mkv",
		"/etc/passwd.mkv",
	} {
		got, err := ValidateSubtitleOutputPathWithFormat(escape, "en", false, "vtt")
		if err == nil && !strings.HasPrefix(got, dir+"/") {
			t.Errorf("path escaped the media directory: %q -> %q", escape, got)
		}
	}

	// The in-bounds case must still work, or the test above passes vacuously.
	if got, err := ValidateSubtitleOutputPathWithFormat(dir+"/movie.mkv", "en", false, "vtt"); err != nil || got != dir+"/movie.en.vtt" {
		t.Errorf("in-bounds path rejected: %q, %v", got, err)
	}
}
