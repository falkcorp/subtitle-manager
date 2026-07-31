// file: pkg/webserver/subtitlepath.go
// version: 1.0.0
// guid: b30c5e94-1a76-4f28-8d51-6c02fa3719be
// last-edited: 2026-07-31

package webserver

import (
	"os"

	"github.com/jdfalk/subtitle-manager/pkg/scanner"
	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/jdfalk/subtitle-manager/pkg/subtitles"
)

// writtenSubtitlePath returns the subtitle file the scanner just wrote for
// videoPath in lang.
//
// The extension follows "subtitles.format", but the scanner falls back to SRT
// when a payload cannot be converted, so the configured format is a
// preference rather than a guarantee. The configured path is preferred and
// SRT is checked as a fallback; if neither exists the configured path is
// returned so the caller still gets a sensible answer to report.
func writtenSubtitlePath(videoPath, lang string) (string, error) {
	format := scanner.OutputFormat()
	single := subtitles.SingleLanguageNaming()

	primary, err := security.ValidateSubtitleOutputPathWithFormat(videoPath, lang, single, string(format))
	if err != nil {
		return "", err
	}
	if _, statErr := os.Stat(primary); statErr == nil {
		return primary, nil
	}
	if format == subtitles.DefaultFormat {
		return primary, nil
	}

	fallback, err := security.ValidateSubtitleOutputPathWithFormat(videoPath, lang, single, string(subtitles.DefaultFormat))
	if err != nil {
		return primary, nil
	}
	if _, statErr := os.Stat(fallback); statErr == nil {
		return fallback, nil
	}
	return primary, nil
}
