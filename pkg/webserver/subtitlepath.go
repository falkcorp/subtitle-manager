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
//
// CodeQL flags the os.Stat calls below as "uncontrolled data in a path
// expression" because it cannot see through the validation. videoPath has
// already passed security.ValidateAndSanitizePath, and
// ValidateSubtitleOutputPathWithFormat re-validates the constructed path
// against the allowed base directories and rejects the format outright unless
// it is srt, vtt or ass. Both constraints are pinned by tests in pkg/security
// (TestValidateSubtitleOutputPathWithFormatConfinesPath and
// ...RejectsBadFormat), so the assumption breaks a test rather than breaking
// quietly. The calls are read-only existence checks either way.
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
