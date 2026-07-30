// file: pkg/providers/validate.go
// version: 1.0.0
// guid: f6bafd75-4871-464c-adce-fb0e0890cf03
// last-edited: 2026-07-30

package providers

import (
	"bytes"
	"errors"
	"unicode/utf8"
)

// ErrNotSubtitle reports that a provider returned something that is not a
// subtitle. It is treated as a fetch failure so the search moves on.
var ErrNotSubtitle = errors.New("provider response is not a subtitle")

// minSubtitleBytes is a floor below which no real subtitle exists. A single
// SRT cue — index, timecode line, one line of text — is already ~40 bytes.
const minSubtitleBytes = 32

// looksLikeSubtitle reports whether data plausibly contains subtitles.
//
// # Why this is necessary
//
// Most of the registered providers are stubs that GET
// https://api.<name>.com/subtitles/... — hostnames nobody controls. Several
// resolve to parked pages, captive portals or generic API gateways that answer
// 200 to anything. Without this check, subtitle-manager accepted whatever came
// back: fetching a *nonexistent* media path succeeded and wrote a 2-byte file
// containing "OK" next to the media, reporting success.
//
// The second-order effect is worse than the junk file. A provider that
// "succeeds" ends the search, so a stub answering 200 to everything preempts
// the real providers that would have found an actual subtitle — the more
// providers are configured, the likelier a search is silently poisoned.
//
// The check is deliberately permissive about format and strict about
// structure: SRT and WebVTT both use "-->" between timestamps, and
// SubStation Alpha has a [Script Info] or [Events] section with Dialogue
// lines. Anything without one of those markers is not a subtitle in a format
// this application can use, whatever else it may be.
func looksLikeSubtitle(data []byte) bool {
	if len(data) < minSubtitleBytes {
		return false
	}
	// A subtitle is text. Binary payloads are archives that should already have
	// been extracted, or error pages that happen to be long.
	if !utf8.Valid(data) {
		return false
	}

	// Only the head needs inspecting; markers appear early in every format.
	head := data
	if len(head) > 4096 {
		head = head[:4096]
	}
	lower := bytes.ToLower(head)

	// SRT and WebVTT.
	if bytes.Contains(head, []byte("-->")) {
		return true
	}
	// SubStation Alpha / Advanced SubStation Alpha.
	if bytes.Contains(lower, []byte("[script info]")) ||
		bytes.Contains(lower, []byte("[events]")) ||
		bytes.Contains(lower, []byte("dialogue:")) {
		return true
	}
	return false
}
