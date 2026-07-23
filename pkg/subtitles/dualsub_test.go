// file: pkg/subtitles/dualsub_test.go
// version: 1.0.0
// guid: 0d6f4a1c-9e52-4b7a-8c31-5f2d7e9a1b04

package subtitles

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/translator"
)

// TestGenerateDualSubtitles verifies the bilingual output keeps the original
// text and appends the translation as a second stacked line per cue.
func TestGenerateDualSubtitles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		qs := r.URL.Query()["q"]
		if len(qs) == 0 {
			qs = []string{""}
		}
		parts := make([]string, len(qs))
		for i := range qs {
			parts[i] = `{"translatedText":"你好"}`
		}
		fmt.Fprintf(w, `{"data":{"translations":[%s]}}`, strings.Join(parts, ","))
	}))
	defer srv.Close()
	translator.SetGoogleAPIURL(srv.URL)
	defer translator.SetGoogleAPIURL("https://translation.googleapis.com/language/translate/v2")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	src := filepath.Join(wd, "../../testdata/simple.srt")
	inData, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	dir := t.TempDir()
	inPath := filepath.Join(dir, "in.srt")
	if err := os.WriteFile(inPath, inData, 0644); err != nil {
		t.Fatalf("write in: %v", err)
	}
	viper.Set("media_directory", dir)
	t.Cleanup(viper.Reset)

	// Esperanto sentinel tag, exactly as the pipeline intends.
	out := filepath.Join(dir, "in.eo.srt")
	if err := GenerateDualSubtitles(inPath, out, "zh", "google", "k", "", ""); err != nil {
		t.Fatalf("dual: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	got := string(data)
	// Original preserved.
	if !strings.Contains(got, "Hello world") {
		t.Fatalf("original text missing from dual subs:\n%s", got)
	}
	// Translation appended.
	if !strings.Contains(got, "你好") {
		t.Fatalf("translation missing from dual subs:\n%s", got)
	}
	// Original must appear before the translation within the first cue.
	if idxOrig, idxT := strings.Index(got, "Hello world"), strings.Index(got, "你好"); idxOrig > idxT {
		t.Fatalf("translation should be stacked below original; got original@%d translation@%d", idxOrig, idxT)
	}
}
