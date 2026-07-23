// file: pkg/monitoring/monitor_processfile_test.go
// version: 1.0.0
// guid: 4c8e1b70-2d95-4a38-b6f0-1e7c3a9d0562

package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
)

// TestCheckLanguageRoutesThroughProcessFile verifies the monitoring loop now
// uses scanner.ProcessFile: with all providers failing but Whisper fallback
// enabled, a subtitle is still written — behaviour the old bare-fetch path
// (providers.FetchFromAll + storeSubtitle) could not produce.
func TestCheckLanguageRoutesThroughProcessFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "1\n00:00:00,000 --> 00:00:01,000\ntranscribed\n")
	}))
	defer srv.Close()

	dir := t.TempDir()
	viper.Reset()
	viper.Set("media_directory", dir)
	viper.Set("whisper.fallback_enabled", true)
	viper.Set("whisper.transcribe_url", srv.URL)
	defer viper.Reset()

	vid := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(vid, []byte("media"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	store, err := database.OpenPebble(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	m := &EpisodeMonitor{store: store, logger: logging.GetLogger("test")}
	item := &MonitoredItem{ID: "1", Path: vid, Languages: []string{"en"}}

	// Bound the context so the provider sweep (which tries every stub) gives up
	// quickly; the Whisper fallback uses its own context and still writes.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := m.checkLanguage(ctx, item, "en"); err != nil {
		t.Fatalf("checkLanguage: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "movie.en.srt")); err != nil {
		t.Fatalf("expected subtitle written via ProcessFile+whisper fallback: %v", err)
	}
}
