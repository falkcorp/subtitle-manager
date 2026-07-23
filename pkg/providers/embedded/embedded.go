// file: pkg/providers/embedded/embedded.go
// version: 2.0.0
// guid: 7c1a9e34-2b58-4d06-9f13-8a6e2c4b7d90

// Package embedded provides a subtitle "provider" that extracts subtitle tracks
// already embedded in the media container (via ffmpeg), rather than downloading
// from an external service. It is a genuinely functional source: Bazarr's "use
// embedded subtitles" feature.
package embedded

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/asticode/go-astisub"
	"golang.org/x/text/language"

	"github.com/jdfalk/subtitle-manager/pkg/subtitles"
	"github.com/jdfalk/subtitle-manager/pkg/video"
)

// Client implements the providers.Provider interface by extracting an embedded
// subtitle track that matches the requested language.
type Client struct{}

// New returns a Client.
func New() *Client { return &Client{} }

// Fetch extracts the embedded subtitle track of mediaPath whose language matches
// lang and returns it as SRT bytes. Image-based tracks (PGS/VobSub/DVD) are
// skipped because they cannot be converted to text. It returns an error when no
// matching text track is present.
func (c *Client) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	streams, err := video.SubtitleStreams(mediaPath)
	if err != nil {
		return nil, err
	}

	ordinal := -1
	for _, s := range streams {
		if s.ImageBased() {
			continue
		}
		if languageMatches(s.Language, lang) {
			ordinal = s.Ordinal
			break
		}
	}
	// If nothing matched by language but there is exactly one text track and the
	// caller did not constrain the language, use it.
	if ordinal < 0 && lang == "" {
		var textOnly []video.SubtitleStream
		for _, s := range streams {
			if !s.ImageBased() {
				textOnly = append(textOnly, s)
			}
		}
		if len(textOnly) == 1 {
			ordinal = textOnly[0].Ordinal
		}
	}
	if ordinal < 0 {
		return nil, fmt.Errorf("no embedded %q subtitle track in %s", lang, mediaPath)
	}

	items, err := subtitles.ExtractTrack(mediaPath, ordinal)
	if err != nil {
		return nil, fmt.Errorf("extract embedded track %d: %w", ordinal, err)
	}
	buf := &bytes.Buffer{}
	if err := (&astisub.Subtitles{Items: items}).WriteToSRT(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// languageMatches reports whether a stream language tag matches the requested
// language, tolerating ISO 639-1 vs 639-2 differences (e.g. "en" vs "eng").
func languageMatches(streamLang, want string) bool {
	streamLang = strings.TrimSpace(streamLang)
	want = strings.TrimSpace(want)
	if streamLang == "" || want == "" {
		return false
	}
	a, err1 := language.Parse(streamLang)
	b, err2 := language.Parse(want)
	if err1 == nil && err2 == nil {
		ba, _ := a.Base()
		bb, _ := b.Base()
		return ba == bb
	}
	return strings.EqualFold(streamLang, want)
}
