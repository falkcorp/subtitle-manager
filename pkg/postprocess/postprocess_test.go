// file: pkg/postprocess/postprocess_test.go
// version: 1.0.0
// guid: e8717c91-78e9-4330-bdda-91b8e9cd31fa

package postprocess

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// TestEncodeUTF8 verifies valid UTF-8 passes through (BOM stripped) and a
// Windows-1252/Latin-1 byte is transcoded to UTF-8.
func TestEncodeUTF8(t *testing.T) {
	// Valid UTF-8 with a BOM.
	in := append([]byte{0xEF, 0xBB, 0xBF}, []byte("héllo")...)
	got := EncodeUTF8(in)
	if string(got) != "héllo" {
		t.Fatalf("expected BOM stripped UTF-8, got %q", got)
	}

	// "café" with é as a single Latin-1 byte 0xE9 (invalid UTF-8).
	latin1 := []byte{'c', 'a', 'f', 0xE9}
	out := EncodeUTF8(latin1)
	if !strings.Contains(string(out), "café") {
		t.Fatalf("expected transcoded café, got %q (% x)", out, out)
	}
}

// TestEncodeUTF8IfEnabledGating verifies the toggle.
func TestEncodeUTF8IfEnabledGating(t *testing.T) {
	defer viper.Reset()
	latin1 := []byte{'c', 'a', 'f', 0xE9}

	viper.Set("postprocess.utf8_encoding", false)
	if got := EncodeUTF8IfEnabled(latin1); string(got) != string(latin1) {
		t.Fatalf("disabled should pass through unchanged")
	}
	viper.Set("postprocess.utf8_encoding", true)
	if got := EncodeUTF8IfEnabled(latin1); !strings.Contains(string(got), "café") {
		t.Fatalf("enabled should transcode, got %q", got)
	}
}

// TestAfterDownloadChmodAndScript verifies chmod and the custom script run with
// the documented environment variables.
func TestAfterDownloadChmodAndScript(t *testing.T) {
	defer viper.Reset()
	dir := t.TempDir()
	sub := filepath.Join(dir, "movie.en.srt")
	if err := os.WriteFile(sub, []byte("1\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	marker := filepath.Join(dir, "ran.txt")

	viper.Set("postprocess.chmod", "0640")
	// Script records the env vars it received.
	viper.Set("postprocess.custom_script", "printf '%s|%s|%s' \"$SM_SUBTITLE_PATH\" \"$SM_MEDIA_PATH\" \"$SM_LANG\" > "+marker)

	AfterDownload(context.Background(), sub, filepath.Join(dir, "movie.mkv"), "en", Info{})

	fi, err := os.Stat(sub)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Mode().Perm() != 0o640 {
		t.Fatalf("expected mode 0640, got %o", fi.Mode().Perm())
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("script did not run (no marker): %v", err)
	}
	want := sub + "|" + filepath.Join(dir, "movie.mkv") + "|en"
	if string(data) != want {
		t.Fatalf("script env mismatch: got %q want %q", data, want)
	}
}

// TestAfterDownloadProviderScoreVars verifies SM_PROVIDER and SM_SCORE are
// exposed to the custom script.
func TestAfterDownloadProviderScoreVars(t *testing.T) {
	defer viper.Reset()
	dir := t.TempDir()
	sub := filepath.Join(dir, "movie.en.srt")
	if err := os.WriteFile(sub, []byte("1\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	marker := filepath.Join(dir, "vars.txt")
	viper.Set("postprocess.custom_script", "printf '%s|%s' \"$SM_PROVIDER\" \"$SM_SCORE\" > "+marker)

	score := 0.87
	AfterDownload(context.Background(), sub, filepath.Join(dir, "movie.mkv"), "en", Info{Provider: "opensubtitles", Score: &score})

	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("script did not run: %v", err)
	}
	if string(data) != "opensubtitles|87" {
		t.Fatalf("unexpected provider/score vars: %q", data)
	}
}

// TestAfterDownloadScoreThreshold verifies the custom script is skipped when the
// score is below postprocess.score_threshold.
func TestAfterDownloadScoreThreshold(t *testing.T) {
	defer viper.Reset()
	dir := t.TempDir()
	sub := filepath.Join(dir, "movie.en.srt")
	if err := os.WriteFile(sub, []byte("1\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	marker := filepath.Join(dir, "ran.txt")
	viper.Set("postprocess.custom_script", "touch "+marker)
	viper.Set("postprocess.score_threshold", 80)

	// Below threshold → skipped.
	low := 0.50
	AfterDownload(context.Background(), sub, filepath.Join(dir, "movie.mkv"), "en", Info{Score: &low})
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("script should be skipped below score threshold")
	}

	// At/above threshold → runs.
	high := 0.90
	AfterDownload(context.Background(), sub, filepath.Join(dir, "movie.mkv"), "en", Info{Score: &high})
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("script should run above threshold: %v", err)
	}
}
