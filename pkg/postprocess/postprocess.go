// file: pkg/postprocess/postprocess.go
// version: 1.0.0
// guid: e05a04c4-5ff1-4ed0-8c91-3b114e2a3d94

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
	"unicode/utf8"

	"github.com/asticode/go-astisub"
	"github.com/spf13/viper"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"

	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/syncer"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// EncodeUTF8 converts subtitle bytes to UTF-8. If the data is already valid
// UTF-8 it is returned unchanged (minus any BOM). Otherwise the charset is
// detected (e.g. Windows-1252, ISO-8859-1) and transcoded. On detection/decode
// failure the original bytes are returned so a download is never lost.
func EncodeUTF8(data []byte) []byte {
	if utf8.Valid(data) {
		return bytes.TrimPrefix(data, utf8BOM)
	}
	enc, _, _ := charset.DetermineEncoding(data, "")
	out, _, err := transform.Bytes(enc.NewDecoder(), data)
	if err != nil {
		return data
	}
	return bytes.TrimPrefix(out, utf8BOM)
}

// EncodeUTF8IfEnabled applies EncodeUTF8 only when postprocess.utf8_encoding is
// set. Called on the downloaded bytes before they are written.
func EncodeUTF8IfEnabled(data []byte) []byte {
	if !viper.GetBool("postprocess.utf8_encoding") {
		return data
	}
	return EncodeUTF8(data)
}

// AfterDownload runs the post-write steps (chmod, auto-sync, custom script) on a
// subtitle that has just been written for the given media file. Each step is
// gated by config and errors are logged, not returned, so post-processing never
// fails a completed download.
func AfterDownload(ctx context.Context, subtitlePath, mediaPath, lang string) {
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
		if err := runScript(ctx, script, subtitlePath, mediaPath, lang); err != nil {
			logger.Warnf("custom script: %v", err)
		}
	}
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
func runScript(ctx context.Context, script, subtitlePath, mediaPath, lang string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Env = append(os.Environ(),
		"SM_SUBTITLE_PATH="+subtitlePath,
		"SM_MEDIA_PATH="+mediaPath,
		"SM_LANG="+lang,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, bytes.TrimSpace(out))
	}
	return nil
}
