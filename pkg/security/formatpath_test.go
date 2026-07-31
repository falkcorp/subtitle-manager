// file: pkg/security/formatpath_test.go
// version: 1.0.0
// guid: 6d0a94f1-58c3-4e27-b91f-72e5c0846a3b
// last-edited: 2026-07-31

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
	dir := t.TempDir()
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
	dir := t.TempDir()
	viper.Set("media_directory", dir)
	defer viper.Reset()

	for _, escape := range []string{
		dir + "/../../etc/passwd.mkv",
		"../../../../etc/shadow.mkv",
		"/etc/passwd.mkv",
	} {
		got, err := ValidateSubtitleOutputPathWithFormat(escape, "en", false, "vtt")
		if err == nil && !strings.HasPrefix(got, dir) {
			t.Errorf("path escaped the sandbox: %q -> %q", escape, got)
		}
	}
}
