// file: pkg/providers/validate_test.go
// version: 1.1.0
// guid: 3e7d1b96-0c45-4a28-95f1-8b60e2a4c7d3
// last-edited: 2026-07-31

package providers

import (
	"context"
	"strings"
	"testing"
)

func TestLooksLikeSubtitle(t *testing.T) {
	srt := "1\n00:00:01,000 --> 00:00:02,000\nHello world\n\n2\n00:00:03,000 --> 00:00:04,000\nGoodbye\n"
	ass := "[Script Info]\nTitle: x\n\n[Events]\nFormat: Layer, Start, End\nDialogue: 0,0:00:01.00,0:00:02.00,Default,,hi\n"
	vtt := "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nHello world, this is a caption line\n"

	for name, tc := range map[string]struct {
		in   string
		want bool
	}{
		"srt":    {srt, true},
		"ass":    {ass, true},
		"webvtt": {vtt, true},

		// The case that prompted this: a live host answering 200 to anything.
		// Fetching a nonexistent media path wrote this to disk as a subtitle.
		"the literal OK a parked host returned": {"OK", false},

		"empty":           {"", false},
		"html error page": {"<!doctype html><html><head><title>404 Not Found</title></head><body>Not found here at all</body></html>", false},
		"json api error":  {`{"error":"not found","message":"no subtitle available for this media file"}`, false},
		"plain prose":     {strings.Repeat("just some text that is long enough to pass a length check. ", 3), false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := looksLikeSubtitle([]byte(tc.in)); got != tc.want {
				t.Errorf("looksLikeSubtitle(%.40q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// okProvider mimics a stub provider whose hostname resolves to something that
// answers 200 to every request.
type okProvider struct{}

func (okProvider) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	return []byte("OK"), nil
}

// realProvider returns an actual subtitle.
type realProvider struct{}

func (realProvider) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	return []byte("1\n00:00:01,000 --> 00:00:02,000\nHello world\n"), nil
}

// TestFetchRejectsNonSubtitleAndContinues is the behavioural guard.
//
// A provider that answers 200 with junk used to end the search successfully:
// the junk was written to disk as a subtitle, and — the worse half — the real
// provider behind it was never consulted. Ordering the junk provider first
// reproduces exactly that.
func TestFetchRejectsNonSubtitleAndContinues(t *testing.T) {
	RegisterFactory("junk200", func() Provider { return okProvider{} })
	RegisterFactory("realsub", func() Provider { return realProvider{} })

	useConcurrency(t, 4)
	useInstances(t,
		Instance{ID: "junk200-1", Name: "junk200", Enabled: true, Priority: 100},
		Instance{ID: "realsub-1", Name: "realsub", Enabled: true, Priority: 1},
	)

	data, id, err := FetchFromAll(context.Background(), "/media/movie.mkv", "en", "")
	if err != nil {
		t.Fatalf("FetchFromAll: %v", err)
	}
	if id != "realsub-1" {
		t.Errorf("winner = %q, want realsub-1: a provider returning %q was accepted "+
			"as a subtitle and ended the search", id, "OK")
	}
	if !strings.Contains(string(data), "-->") {
		t.Errorf("data = %q, want a real subtitle", data)
	}
}

// hebrewCP1255SRT is the opening of a real subtitle downloaded from
// wizdom.xyz. It is a valid SRT whose text is Hebrew encoded in Windows-1255,
// so the bytes are *not* valid UTF-8 — the shape most of the world's
// non-English subtitles actually arrive in.
var hebrewCP1255SRT = []byte("1\r\n00:00:54,542 --> 00:00:59,323\r\n" +
	"-\xe7\xe5\xee\xe5\xfa \xf9\xec \xfa\xf7\xe5\xe5\xe4-\r\n\r\n" +
	"2\r\n00:00:59,671 --> 00:01:11,600\r\n" +
	"\xf1\xe5\xf0\xeb\xf8\xef \xec\xe2\xe9\xf8\xf1\xe0 \xe6\xe0\xfa \xf2\"\xe9\r\n")

// TestLooksLikeSubtitleAcceptsLegacyEncodings guards the rule that validation
// keys on structure, not on character encoding.
//
// Requiring valid UTF-8 here silently discarded every subtitle distributed in
// a legacy single-byte codepage — Windows-1255 Hebrew, 1251 Cyrillic, 1252
// Western European. The loss was invisible: the fetch was recorded as a
// provider failure and the search simply moved on to the next provider, so a
// working provider looked like a dead one. Transcoding belongs downstream in
// postprocess.EncodeUTF8; this function only decides "is this a subtitle".
func TestLooksLikeSubtitleAcceptsLegacyEncodings(t *testing.T) {
	// Windows-1252: "Ça va, mon frère?" — invalid UTF-8, unmistakably an SRT.
	frenchCP1252 := []byte("1\n00:00:01,000 --> 00:00:02,000\n\xc7a va, mon fr\xe8re ?\n")
	// Windows-1251 Cyrillic inside an ASS Dialogue line.
	russianCP1251 := []byte("[Script Info]\nTitle: x\n\n[Events]\n" +
		"Dialogue: 0,0:00:01.00,0:00:02.00,Default,,\xcf\xf0\xe8\xe2\xe5\xf2\n")

	for name, in := range map[string][]byte{
		"windows-1255 hebrew srt from wizdom": hebrewCP1255SRT,
		"windows-1252 french srt":             frenchCP1252,
		"windows-1251 russian ass":            russianCP1251,
	} {
		t.Run(name, func(t *testing.T) {
			if !looksLikeSubtitle(in) {
				t.Errorf("looksLikeSubtitle rejected a valid non-UTF-8 subtitle (% x...)", in[:16])
			}
		})
	}
}

// TestLooksLikeSubtitleRejectsBinary pins the replacement for the UTF-8 check.
// Dropping utf8.Valid must not let archives or images through: a ZIP that was
// never extracted, or a PNG an error page served, would otherwise be written
// next to the media as a subtitle.
func TestLooksLikeSubtitleRejectsBinary(t *testing.T) {
	// A ZIP local-file header whose stored name happens to contain "-->",
	// so only the NUL check can reject it.
	zipWithMarker := append([]byte("PK\x03\x04\x14\x00\x00\x00\x00\x00"),
		[]byte("a-->b.srt\x00\x00\x00padding to clear the length floor")...)
	png := append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"),
		[]byte("\x00\x00\x01\x00\x00\x00\x01\x00\x08\x06--> not a cue\x00\x00")...)

	for name, in := range map[string][]byte{
		"zip archive": zipWithMarker,
		"png image":   png,
	} {
		t.Run(name, func(t *testing.T) {
			if looksLikeSubtitle(in) {
				t.Errorf("looksLikeSubtitle accepted binary data (% x...)", in[:8])
			}
		})
	}
}
