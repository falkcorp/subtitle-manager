// file: pkg/providers/validate_test.go
// version: 1.0.0
// guid: 3e7d1b96-0c45-4a28-95f1-8b60e2a4c7d3
// last-edited: 2026-07-30

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
