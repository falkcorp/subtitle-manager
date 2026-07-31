// file: pkg/postprocess/postprocess.go
// version: 1.2.0
// guid: e05a04c4-5ff1-4ed0-8c91-3b114e2a3d94
// last-edited: 2026-07-31

// Package postprocess implements the Bazarr-style post-processing pipeline that
// runs on a subtitle after it is downloaded/written: UTF-8 re-encoding,
// permission (chmod) enforcement, automatic sync against the media, and an
// optional custom user script. Every step is opt-in via config, so with no
// configuration the pipeline is a no-op and the download path is unchanged.
//
// Config keys (viper):
//
//	postprocess.utf8_encoding  bool    convert downloaded subtitles to UTF-8
//	postprocess.chmod          string  octal mode applied to the file, e.g. "0644"
//	postprocess.auto_sync      bool    sync the subtitle to the media after download
//	postprocess.custom_script  string  shell command run after download
package postprocess

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/asticode/go-astisub"
	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/subtitles"
	"github.com/jdfalk/subtitle-manager/pkg/syncer"
)

// EncodeUTF8 converts subtitle bytes to UTF-8. If the data is already valid
// UTF-8 it is returned unchanged (minus any BOM). Otherwise the charset is
// detected (e.g. Windows-1252, ISO-8859-1) and transcoded. On detection/decode
// failure the original bytes are returned so a download is never lost.
//
// The implementation lives in pkg/subtitles because format conversion needs
// the same normalisation and cannot import this package (postprocess already
// depends on subtitles, not the other way round). This remains the name the
// rest of the application calls.
func EncodeUTF8(data []byte) []byte {
	return subtitles.EncodeUTF8(data)
}

// EncodeUTF8IfEnabled applies EncodeUTF8 only when postprocess.utf8_encoding is
// set. Called on the downloaded bytes before they are written.
func EncodeUTF8IfEnabled(data []byte) []byte {
	if !viper.GetBool("postprocess.utf8_encoding") {
		return data
	}
	return EncodeUTF8(data)
}

// Info carries optional metadata about the download for the custom
// post-processing script and its score-threshold gate.
type Info struct {
	// Provider is the subtitle provider name (may be empty).
	Provider string
	// Score is the normalized quality score in [0,1], or nil when unknown.
	Score *float64
}

// AfterDownload runs the post-write steps (chmod, auto-sync, custom script) on a
// subtitle that has just been written for the given media file. Each step is
// gated by config and errors are logged, not returned, so post-processing never
// fails a completed download. info supplies the provider and score exposed to
// the custom script and the score-threshold gate.
func AfterDownload(ctx context.Context, subtitlePath, mediaPath, lang string, info Info) {
	logger := logging.GetLogger("postprocess")

	if mode := viper.GetString("postprocess.chmod"); mode != "" {
		if m, err := strconv.ParseUint(mode, 8, 32); err == nil {
			if err := os.Chmod(subtitlePath, os.FileMode(m)); err != nil {
				logger.Warnf("chmod %s: %v", subtitlePath, err)
			}
		} else {
			logger.Warnf("invalid postprocess.chmod %q: %v", mode, err)
		}
	}

	if viper.GetBool("postprocess.auto_sync") && mediaPath != "" {
		if err := autoSync(mediaPath, subtitlePath); err != nil {
			logger.Warnf("auto-sync %s: %v", subtitlePath, err)
		}
	}

	if script := viper.GetString("postprocess.custom_script"); script != "" {
		if scoreBelowThreshold(info.Score) {
			logger.Debugf("skipping custom script: score below postprocess.score_threshold")
		} else if err := runScript(ctx, script, subtitlePath, mediaPath, lang, info); err != nil {
			logger.Warnf("custom script: %v", err)
		}
	}
}

// scoreBelowThreshold reports whether a known score is below the configured
// postprocess.score_threshold (0-100). An unset threshold, or an unknown score,
// never gates.
func scoreBelowThreshold(score *float64) bool {
	threshold := viper.GetInt("postprocess.score_threshold")
	if threshold <= 0 || score == nil {
		return false
	}
	return *score*100 < float64(threshold)
}

// autoSync synchronizes subtitlePath to mediaPath (using embedded subtitles as
// the reference) and overwrites the file with the aligned result.
func autoSync(mediaPath, subtitlePath string) error {
	items, err := syncer.Sync(mediaPath, subtitlePath, syncer.Options{UseEmbedded: true})
	if err != nil {
		return err
	}
	f, err := os.Create(subtitlePath)
	if err != nil {
		return err
	}
	defer f.Close()
	sub := astisub.Subtitles{Items: items}
	return sub.WriteToSRT(f)
}

// runScript executes the configured shell command, passing the subtitle, media
// and language via environment variables (SM_SUBTITLE_PATH / SM_MEDIA_PATH /
// SM_LANG) so paths are never interpolated into the command string.
func runScript(ctx context.Context, script, subtitlePath, mediaPath, lang string, info Info) error {
	score := ""
	if info.Score != nil {
		score = strconv.Itoa(int(*info.Score * 100))
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Env = append(os.Environ(),
		"SM_SUBTITLE_PATH="+subtitlePath,
		"SM_MEDIA_PATH="+mediaPath,
		"SM_LANG="+lang,
		"SM_PROVIDER="+info.Provider,
		"SM_SCORE="+score,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, bytes.TrimSpace(out))
	}
	return nil
}
