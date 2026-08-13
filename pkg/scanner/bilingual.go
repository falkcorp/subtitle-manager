// file: pkg/scanner/bilingual.go
// version: 1.0.0
// guid: e44a390a-a81c-4356-8090-da46b181d8cc
// last-edited: 2026-08-12

package scanner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/asticode/go-astisub"
	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/jdfalk/subtitle-manager/pkg/subtitles"
)

// defaultSentinelLang tags the generated bilingual file with a language no real
// track claims, so Plex/Jellyfin/Emby list it as a distinct selectable track
// instead of colliding with the primary language. Matches the default used by
// `dualsub` and POST /api/subtitles/stack.
const defaultSentinelLang = "eo"

// sentinelLang returns the configured sentinel language code.
func sentinelLang() string {
	if lang := viper.GetString("dualsub.sentinel_language"); lang != "" {
		return lang
	}
	return defaultSentinelLang
}

// writeBilingualPair combines the two highest-priority languages a profile
// obtained into one stacked subtitle, written twice.
//
// Two names, because they serve different readers:
//
//   - <base>.<primary>-<secondary>.srt is self-describing and cannot collide
//     with either single-language sidecar, so it is unambiguous on disk.
//   - <base>.<sentinel>.srt is what media servers actually surface. They map a
//     filename language tag to a track label, and "en-es" is not a language
//     they know, so without this the bilingual file would never appear in a
//     player's subtitle menu.
//
// The second is a reflink of the first where the filesystem supports it, so it
// costs no extra space on APFS, btrfs, XFS or OpenZFS 2.2+.
//
// This is purely additive: the per-language sidecars stay exactly where they
// are, and an existing file at either output name is never overwritten — the
// sentinel name could legitimately belong to a real Esperanto subtitle.
func writeBilingualPair(videoPath, primaryLang, secondaryLang string) error {
	logger := logging.GetLogger("scanner")

	format := string(outputFormat())
	primaryPath, err := security.ValidateSubtitleOutputPathWithFormat(videoPath, primaryLang, false, format)
	if err != nil {
		return fmt.Errorf("primary sidecar path: %w", err)
	}
	secondaryPath, err := security.ValidateSubtitleOutputPathWithFormat(videoPath, secondaryLang, false, format)
	if err != nil {
		return fmt.Errorf("secondary sidecar path: %w", err)
	}

	// Both sidecars must actually be on disk. A language can be reported as
	// downloaded and still be missing here if post-processing renamed or
	// converted it, and stacking half a pair would produce a file that looks
	// bilingual but is not.
	for _, p := range []string{primaryPath, secondaryPath} {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("expected sidecar %s: %w", p, err)
		}
	}

	combinedPath, err := security.ValidateSubtitleOutputPathWithFormat(
		videoPath, primaryLang+"-"+secondaryLang, false, format)
	if err != nil {
		return fmt.Errorf("bilingual output path: %w", err)
	}
	sentinelPath, err := security.ValidateSubtitleOutputPathWithFormat(videoPath, sentinelLang(), false, format)
	if err != nil {
		return fmt.Errorf("sentinel output path: %w", err)
	}

	if _, err := os.Stat(combinedPath); err == nil {
		logger.Debugf("bilingual subtitle already exists, leaving it alone: %s", combinedPath)
		return nil
	}

	primarySub, err := astisub.OpenFile(primaryPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", primaryPath, err)
	}
	secondarySub, err := astisub.OpenFile(secondaryPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", secondaryPath, err)
	}
	primarySub.Items = subtitles.StackTracks(primarySub.Items, secondarySub.Items)

	// Write through an os.Root rooted at the media directory. Root confines
	// every operation beneath it at the OS level, so a traversal in the name is
	// refused by the kernel-facing API rather than by our own string checks.
	// The equivalent write in pkg/webserver/stacksubs.go was a genuine CodeQL
	// high-severity finding when it used os.Create; do not simplify this back.
	root, err := os.OpenRoot(filepath.Dir(combinedPath))
	if err != nil {
		return fmt.Errorf("opening media directory: %w", err)
	}
	defer root.Close()

	f, err := root.Create(filepath.Base(combinedPath))
	if err != nil {
		return fmt.Errorf("creating %s: %w", combinedPath, err)
	}
	if err := primarySub.WriteToSRT(f); err != nil {
		f.Close()
		return fmt.Errorf("writing %s: %w", combinedPath, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", combinedPath, err)
	}

	// The sentinel copy is what players show, but a failure here is not worth
	// discarding the combined file that already succeeded — most often it just
	// means a real Esperanto subtitle already occupies the name.
	if err := subtitles.CloneFile(combinedPath, sentinelPath); err != nil {
		logger.Warnf("bilingual subtitle written to %s, but the player-visible copy at %s was not created: %v",
			combinedPath, sentinelPath, err)
		return nil
	}

	logger.Infof("wrote bilingual subtitle %s (%s over %s) and player-visible copy %s",
		combinedPath, primaryLang, secondaryLang, sentinelPath)
	return nil
}
