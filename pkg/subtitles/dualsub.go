// file: pkg/subtitles/dualsub.go
// version: 1.0.0
// guid: 64b56784-2864-4efa-bea4-60caa16c1ffa

package subtitles

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/asticode/go-astisub"

	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/jdfalk/subtitle-manager/pkg/translator"
)

// GenerateDualSubtitles reads the subtitle file at inPath, translates each cue's
// text to targetLang, and writes a bilingual ("double subs") SRT to outPath.
//
// Unlike TranslateFileToSRT — which REPLACES each cue's text with its translation
// — this PRESERVES the original line(s) and APPENDS the translation as an
// additional stacked line, so each cue shows the original above and the
// translation below. Translations are cached in-memory so identical source lines
// are only translated once per file.
//
// The output is intended to be tagged with a sentinel language code (Esperanto,
// "eo", by convention) so media players/servers do not treat it as a real
// single-language track. See docs/WHISPER_PIPELINE_DECISIONS.md (W5). This
// produces stacked-SRT double subs; styled ASS positioning is a future option
// (it depends on the currently-stubbed multi-format writer).
func GenerateDualSubtitles(inPath, outPath, targetLang, service, googleKey, gptKey, grpcAddr string) error {
	validatedInPath, err := security.ValidateAndSanitizePath(inPath)
	if err != nil {
		return fmt.Errorf("invalid input path: %w", err)
	}
	validatedOutPath, err := security.ValidateAndSanitizePath(outPath)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	sub, err := astisub.OpenFile(validatedInPath)
	if err != nil {
		return err
	}

	cache := make(map[string]string, len(sub.Items))
	for _, item := range sub.Items {
		orig := itemText(item)
		if orig == "" {
			continue
		}
		t, ok := cache[orig]
		if !ok {
			t, err = translator.Translate(service, orig, targetLang, googleKey, gptKey, grpcAddr)
			if err != nil {
				return fmt.Errorf("translate %q: %w", orig, err)
			}
			cache[orig] = t
		}
		if strings.TrimSpace(t) == "" {
			continue
		}
		// Preserve the original line(s); append the translation as a new stacked
		// line so SRT renders original-over-translation.
		item.Lines = append(item.Lines, astisub.Line{Items: []astisub.LineItem{{Text: t}}})
	}

	buf := &bytes.Buffer{}
	if err := sub.WriteToSRT(buf); err != nil {
		return err
	}
	return os.WriteFile(validatedOutPath, buf.Bytes(), 0644)
}

// itemText concatenates every line's text in a subtitle item into a single
// string, joining lines with a space so the whole cue is translated as one unit
// (rather than only the first line, as the single-language translator does).
func itemText(item *astisub.Item) string {
	var b strings.Builder
	for i, line := range item.Lines {
		for _, li := range line.Items {
			b.WriteString(li.Text)
		}
		if i < len(item.Lines)-1 {
			b.WriteString(" ")
		}
	}
	return strings.TrimSpace(b.String())
}
