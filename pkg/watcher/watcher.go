// file: pkg/watcher/watcher.go
// version: 1.0.0
// guid: b30941d0-038e-4c97-9601-7d2dc16520ad
// last-edited: 2026-08-04

package watcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/providers"
	"github.com/jdfalk/subtitle-manager/pkg/scanner"
	"github.com/jdfalk/subtitle-manager/pkg/security"
)

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

// WatchDirectory monitors dir for new video files and downloads subtitles using
// provider p for the given language. Subtitles are written next to the media
// file with the language code appended before the extension.
func WatchDirectory(ctx context.Context, dir, lang, providerName string, p providers.Provider, store database.SubtitleStore) error {
	logger := logging.GetLogger("watcher")

	sanitizedDir, err := security.ValidateAndSanitizePath(dir)
	if err != nil {
		return err
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	if err := w.Add(sanitizedDir); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-w.Errors:
			logger.Warnf("watch error: %v", err)
		case ev := <-w.Events:
			if ev.Op&(fsnotify.Create|fsnotify.Rename|fsnotify.Write) != 0 && isVideoFile(ev.Name) {
				if err := processWatched(ctx, ev.Name, lang, providerName, p, store); err != nil {
					logger.Warnf("process %s: %v", ev.Name, err)
				}
			}
		}
	}
}

// WatchDirectoryRecursive works like WatchDirectory but monitors dir and all
// of its subdirectories. New directories created while watching are added
// automatically.
func WatchDirectoryRecursive(ctx context.Context, dir, lang, providerName string, p providers.Provider, store database.SubtitleStore) error {
	logger := logging.GetLogger("watcher")

	sanitizedDir, err := security.ValidateAndSanitizePath(dir)
	if err != nil {
		return err
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	if err := filepath.WalkDir(sanitizedDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return w.Add(path)
		}
		return nil
	}); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-w.Errors:
			logger.Warnf("watch error: %v", err)
		case ev := <-w.Events:
			if ev.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					_ = w.Add(ev.Name)
				}
			}
			if ev.Op&(fsnotify.Create|fsnotify.Rename|fsnotify.Write) != 0 && isVideoFile(ev.Name) {
				if err := processWatched(ctx, ev.Name, lang, providerName, p, store); err != nil {
					logger.Warnf("process %s: %v", ev.Name, err)
				}
			}
		}
	}
}

// processWatched downloads subtitles for a newly seen file, honouring the
// file's own language profile when it has one and falling back to the single
// language the watcher was started with.
//
// The watcher is an automated path — nobody named a language for this
// particular file — so an assignment made in the UI should govern it, the same
// way it governs a library scan.
func processWatched(ctx context.Context, path, lang, providerName string,
	p providers.Provider, store database.SubtitleStore) error {
	if handled, err := scanner.ProcessWithProfileIfAssigned(ctx, path, providerName, p, false, store); handled {
		return err
	}
	return scanner.ProcessFile(ctx, path, lang, providerName, p, false, store)
}
