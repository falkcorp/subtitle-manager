// file: pkg/scanner/scanner.go
// version: 1.6.0
// guid: ad2ef6ba-8afa-4ced-8508-0c535dbb23fd
// last-edited: 2026-08-01
package scanner

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sourcegraph/conc/pool"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/events"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/metadata"
	"github.com/jdfalk/subtitle-manager/pkg/postprocess"
	"github.com/jdfalk/subtitle-manager/pkg/providers"
	"github.com/jdfalk/subtitle-manager/pkg/security"
)

// ScanDirectory walks through the directory and downloads subtitles for video files
// using provider p for the given language. providerName is stored in download
// history. If upgrade is false existing subtitle files are skipped.
func ScanDirectory(ctx context.Context, dir, lang string, providerName string, p providers.Provider, upgrade bool, workers int, store database.SubtitleStore) error {
	logger := logging.GetLogger("scanner")
	sanitizedDir, err := security.ValidateAndSanitizePath(dir)
	if err != nil {
		logger.Warnf("invalid path: %v", err)
		return err
	}
	work := pool.New().WithErrors().WithMaxGoroutines(workers)
	err = filepath.WalkDir(sanitizedDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isVideoFile(path) {
			return nil
		}
		f := filepath.Clean(path)
		work.Go(func() error {
			logger.Debugf("process %s", f)
			return ProcessFile(ctx, f, lang, providerName, p, upgrade, store)
		})
		return nil
	})
	if err != nil {
		return err
	}
	if err := work.Wait(); err != nil {
		return err
	}
	if store != nil {
		if err := metadata.ScanLibrary(ctx, sanitizedDir, store); err != nil {
			logger.Warnf("scan library: %v", err)
		}
	}
	return nil
}

// ProcessFile downloads a subtitle for path using providerName for history
// tracking. The subtitle is saved next to the media file with the language
// code appended before the extension. If upgrade is false an existing subtitle
// file is left untouched. When upgrade is true and a subtitle already exists,
// the new subtitle replaces it only if the file size is larger, indicating
// potentially better quality.
func ProcessFile(ctx context.Context, path, lang string, providerName string, p providers.Provider, upgrade bool, store database.SubtitleStore) error {
	logger := logging.GetLogger("scanner")

	// Validate and sanitize all user inputs
	sanitizedPath, err := security.ValidateAndSanitizePath(path)
	if err != nil {
		logger.Warnf("invalid path: %v", err)
		return err
	}
	path = sanitizedPath

	// Validate the language code to prevent path traversal attacks
	if err := security.ValidateLanguageCode(lang); err != nil {
		logger.Warnf("invalid language code: %v", err)
		return err
	}
	// Ensure the language code does not contain any path traversal characters
	if strings.Contains(lang, "/") || strings.Contains(lang, "\\") || strings.Contains(lang, "..") {
		logger.Warnf("language code contains invalid characters")
		return fmt.Errorf("invalid language code")
	}

	// Validate provider name if provided
	if err := security.ValidateProviderName(providerName); err != nil {
		logger.Warnf("invalid provider name: %v", err)
		return err
	}

	// Construct and validate the output path securely. When single-language
	// naming is enabled the language code is omitted from the filename.
	outFormat := outputFormat()
	validatedOutputPath, err := security.ValidateSubtitleOutputPathWithFormat(path, lang, singleLanguageNaming(), string(outFormat))
	if err != nil {
		logger.Warnf("invalid subtitle output path: %v", err)
		return err
	}

	if !upgrade {
		if _, err := os.Stat(validatedOutputPath); err == nil {
			return nil
		}
	}
	var data []byte
	// matchScore holds the normalized (0-1) quality score when the scored path
	// produced the subtitle; it stays nil on the plain byte path.
	var matchScore *float64
	scoredHandled := false
	if sp, ok := p.(scoredProvider); ok && scoringEnabled() {
		res, ferr := fetchBestScored(ctx, sp, path, lang)
		if ferr != nil {
			err = ferr
		} else if res == nil {
			// No candidate cleared the minimum score: nothing to download.
			logger.Infof("no subtitle candidate for %s cleared minimum score", path)
			return nil
		} else {
			data = res.data
			norm := float64(res.total) / 100.0
			matchScore = &norm
			scoredHandled = true
			logger.Infof("selected scored subtitle for %s (score %d, release %q)", path, res.total, res.release)
		}
	}
	if !scoredHandled && err == nil {
		if p != nil {
			data, err = p.Fetch(ctx, path, lang)
		} else {
			data, providerName, err = providers.FetchFromAll(ctx, path, lang, "")
		}
	}
	if err != nil {
		// Whisper fallback: transcribe the media when no provider had a subtitle.
		if fbData, ok := whisperFallback(ctx, path, lang); ok {
			logger.Infof("whisper fallback produced a subtitle for %s", path)
			data = fbData
			providerName = "whisper"
			matchScore = nil
			err = nil
		} else {
			logger.Warnf("fetch %s: %v", path, err)

			// Send event for subtitle fetch failure
			events.PublishSubtitleFailed(ctx, events.SubtitleFailedData{
				FilePath:  path,
				Language:  lang,
				Provider:  providerName,
				Error:     err.Error(),
				Timestamp: time.Now(),
			})

			return err
		}
	}
	var wasUpgrade bool
	if upgrade {
		if oldData, rerr := os.ReadFile(validatedOutputPath); rerr == nil {
			if matchScore != nil {
				// Score-based upgrade: only replace the existing subtitle when
				// the new candidate scores strictly higher than what we already
				// downloaded for this video/language.
				if prior := priorMatchScore(store, path, lang); prior != nil && *matchScore <= *prior {
					logger.Debugf("existing subtitle %s scores >= new candidate (%.2f)", validatedOutputPath, *prior)
					return nil
				}
			} else if len(data) <= len(oldData) {
				logger.Debugf("existing subtitle %s is higher quality", validatedOutputPath)
				return nil
			}
			wasUpgrade = true
		}
	}
	// Post-processing: optionally re-encode to UTF-8 before writing.
	data = postprocess.EncodeUTF8IfEnabled(data)

	// Convert into the configured container. A conversion that fails falls
	// back to SRT, so the path is recomputed to match what is actually being
	// written rather than what was asked for — a ".vtt" holding SRT bytes is
	// exactly the kind of quiet disagreement that is invisible until a player
	// refuses the file.
	var wrote subtitleFormat
	data, wrote = convertForOutput(data)
	if wrote != outFormat {
		validatedOutputPath, err = security.ValidateSubtitleOutputPathWithFormat(path, lang, singleLanguageNaming(), string(wrote))
		if err != nil {
			logger.Warnf("invalid subtitle output path: %v", err)
			return err
		}
	}

	if err := os.WriteFile(validatedOutputPath, data, 0644); err != nil {
		logger.Warnf("write %s: %v", validatedOutputPath, err)

		// Send event for file write failure
		events.PublishSubtitleFailed(ctx, events.SubtitleFailedData{
			FilePath:  path,
			Language:  lang,
			Provider:  providerName,
			Error:     "Failed to write subtitle file: " + err.Error(),
			Timestamp: time.Now(),
		})
		return err
	}
	logger.Infof("downloaded subtitle %s", validatedOutputPath)

	// Get file size for webhook event
	var fileSize int64
	if stat, err := os.Stat(validatedOutputPath); err == nil {
		fileSize = stat.Size()
	}

	// Report the real score when the scored path produced this subtitle,
	// otherwise fall back to the neutral default.
	eventScore := 1.0
	if matchScore != nil {
		eventScore = *matchScore
	}

	// Send appropriate event
	if wasUpgrade {
		events.PublishSubtitleUpgraded(ctx, events.SubtitleUpgradedData{
			FilePath:        path,
			NewSubtitlePath: validatedOutputPath,
			Language:        lang,
			NewProvider:     providerName,
			NewScore:        eventScore,
			Timestamp:       time.Now(),
		})
	} else {
		events.PublishSubtitleDownloaded(ctx, events.SubtitleDownloadedData{
			FilePath:     path,
			SubtitlePath: validatedOutputPath,
			Language:     lang,
			Provider:     providerName,
			Score:        eventScore,
			Size:         fileSize,
			Timestamp:    time.Now(),
		})
	}
	if store != nil {
		_ = store.InsertDownload(&database.DownloadRecord{File: validatedOutputPath, VideoFile: path, Provider: providerName, Language: lang, MatchScore: matchScore})
	}
	// Post-processing: chmod, auto-sync, custom script (all opt-in via config).
	postprocess.AfterDownload(ctx, validatedOutputPath, path, lang, postprocess.Info{Provider: providerName, Score: matchScore})
	return nil
}

var videoExtensions = []string{".mkv", ".mp4", ".avi", ".mov"}

func isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range videoExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

// ScanDirectoryWithProfiles walks through the directory and downloads subtitles for video files
// using language profiles. Each video file's profile is determined by its media_profiles assignment.
func ScanDirectoryWithProfiles(ctx context.Context, dir string, db *sql.DB, upgrade bool, workers int, store database.SubtitleStore) error {
	logger := logging.GetLogger("scanner")
	sanitizedDir, err := security.ValidateAndSanitizePath(dir)
	if err != nil {
		logger.Warnf("invalid path: %v", err)
		return err
	}
	work := pool.New().WithErrors().WithMaxGoroutines(workers)
	err = filepath.WalkDir(sanitizedDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isVideoFile(path) {
			return nil
		}
		f := filepath.Clean(path)
		work.Go(func() error {
			logger.Debugf("process %s with profiles", f)
			return ProcessFileWithProfile(ctx, f, db, upgrade, store)
		})
		return nil
	})
	if err != nil {
		return err
	}
	if err := work.Wait(); err != nil {
		return err
	}
	if store != nil {
		if err := metadata.ScanLibrary(ctx, sanitizedDir, store); err != nil {
			logger.Warnf("scan library: %v", err)
		}
	}
	return nil
}

// ProcessFileWithProfile downloads subtitles for path using the language
// profile assigned to it, covering every language the profile asks for in
// priority order.
//
// It delegates to ProcessFile once per language rather than fetching and
// writing itself. The previous implementation called providers.FetchWithProfile
// and then wrote the file inline, which meant it silently skipped everything
// ProcessFile does around the download — score gating, the Whisper fallback,
// UTF-8 re-encoding, chmod, auto-sync, the custom post-download script, the
// *arr rescan event and download-history persistence. It also read its
// assignment through pkg/profiles.Service, which keys media_profiles on a
// media_items row id in a raw *sql.DB, while the UI and CLI write path-keyed
// rows through the SubtitleStore — so it could not have found a UI assignment
// even when it was reached. It had no callers.
//
// db is retained for signature compatibility and is unused: the profile is
// resolved through store (or the shared store) so that this path and the web
// UI agree on both the database and the key.
func ProcessFileWithProfile(ctx context.Context, path string, db *sql.DB, upgrade bool, store database.SubtitleStore) error {
	logger := logging.GetLogger("scanner")

	sanitizedPath, err := security.ValidateAndSanitizePath(path)
	if err != nil {
		logger.Warnf("invalid path: %v", err)
		return err
	}

	lookupStore := store
	if lookupStore == nil {
		var serr error
		if lookupStore, serr = database.GetSharedStore(); serr != nil {
			return serr
		}
	}
	defaultID, profilesDefined := defaultProfileID(lookupStore)
	if !profilesDefined {
		return fmt.Errorf("no language profile assigned to %s", sanitizedPath)
	}
	langs, ok := assignedProfileLanguages(sanitizedPath, lookupStore, defaultID)
	if !ok {
		return fmt.Errorf("no language profile assigned to %s", sanitizedPath)
	}

	return processWithAssignedProfile(ctx, sanitizedPath, langs, "", nil, upgrade, store)
}
