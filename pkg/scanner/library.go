// file: pkg/scanner/library.go
// version: 1.0.0
// guid: acade7fb-427e-4e78-b36f-894bf25b622c

package scanner

import (
	"context"
	"fmt"
	"os"

	"github.com/sourcegraph/conc/pool"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/providers"
)

// ProcessLibrary downloads subtitles for every media item recorded in the
// store's library. The library is populated by scanlib (filesystem scan) and by
// the Sonarr/Radarr sync, both of which persist database.MediaItem rows keyed by
// the local media file path. This function is the bridge that lets a
// Sonarr/Radarr pull actually drive subtitle search: previously nothing read the
// persisted library to fetch subtitles.
//
// Behaviour and design decisions (see docs/WHISPER_PIPELINE_DECISIONS.md, W1):
//   - It reuses ProcessFile per item, so language validation, output-path
//     construction, existing-subtitle skipping, upgrade logic, download history,
//     and failure events are identical to the directory scanner.
//   - It is best-effort: a single item's fetch failure is logged and does not
//     abort the sweep (unlike the pool-error propagation used elsewhere), because
//     a missing subtitle for one title should not stop the rest of the library.
//   - Items whose file no longer exists on disk, or that are not video files, are
//     skipped rather than treated as errors (the library can lag the filesystem).
//
// When p is nil, ProcessFile falls back to providers.FetchFromAll across all
// configured providers.
func ProcessLibrary(ctx context.Context, lang, providerName string, p providers.Provider, upgrade bool, workers int, store database.SubtitleStore) error {
	logger := logging.GetLogger("scanner")
	if store == nil {
		return fmt.Errorf("library search requires a database store")
	}
	items, err := store.ListMediaItems()
	if err != nil {
		return fmt.Errorf("list media items: %w", err)
	}
	if workers < 1 {
		workers = 1
	}
	logger.Infof("library search over %d media items (lang=%s)", len(items), lang)

	work := pool.New().WithErrors().WithMaxGoroutines(workers)
	for i := range items {
		item := items[i]
		if item.Path == "" || !isVideoFile(item.Path) {
			continue
		}
		if _, err := os.Stat(item.Path); err != nil {
			logger.Debugf("skip missing library file %s", item.Path)
			continue
		}
		work.Go(func() error {
			logger.Debugf("library search %s", item.Path)
			if err := ProcessFile(ctx, item.Path, lang, providerName, p, upgrade, store); err != nil {
				// Best-effort: log and continue so one failure does not abort the sweep.
				logger.Warnf("library search %s: %v", item.Path, err)
			}
			return nil
		})
	}
	return work.Wait()
}
