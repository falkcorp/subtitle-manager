// file: pkg/subtitles/format_test.go
// version: 1.0.0
// guid: c8e2f45b-9037-41da-b6e8-2f0a71c5934e
// last-edited: 2026-07-31

package subtitles

import (
	"bytes"
	"strings"
	"testing"

	"github.com/asticode/go-astisub"
)

const srtSample = "1\n00:00:01,000 --> 00:00:02,500\nFirst line\n\n" +
	"2\n00:00:03,000 --> 00:00:04,500\nSecond line\n"

const vttSample = "WEBVTT\n\n00:00:01.000 --> 00:00:02.500\nFirst line\n\n" +
	"00:00:03.000 --> 00:00:04.500\nSecond line\n"

const assSample = "[Script Info]\nScriptType: v4.00+\n\n" +
	"[V4+ Styles]\nFormat: Name\nStyle: Default\n\n" +
	"[Events]\nFormat: Layer, Start, End, Style, Text\n" +
	"Dialogue: 0,0:00:01.00,0:00:02.50,Default,First line\n" +
	"Dialogue: 0,0:00:03.00,0:00:04.50,Default,Second line\n"

// TestParseFormat pins the allowlist. The value reaches a file name joined
// onto a media directory, so anything off the list must be an error rather
// than something sanitised into a path.
func TestParseFormat(t *testing.T) {
	for in, want := range map[string]Format{
		"srt": FormatSRT, "vtt": FormatVTT, "ass": FormatASS,
		"SRT": FormatSRT, " vtt ": FormatVTT,
		"": DefaultFormat,
	} {
		got, err := ParseFormat(in)
		if err != nil || got != want {
			t.Errorf("ParseFormat(%q) = %q, %v; want %q, nil", in, got, err, want)
		}
	}
	for _, bad := range []string{"ssa/../../etc/passwd", "../../evil", "sub", "exe", "srt.exe", "."} {
		if got, err := ParseFormat(bad); err == nil {
			t.Errorf("ParseFormat(%q) = %q with no error; must be rejected", bad, got)
		}
	}
}

// TestDetectFormat pins the byte sniffer. It exists because the download path
// has an anonymous []byte from a provider and no file name to infer from.
func TestDetectFormat(t *testing.T) {
	for name, tc := range map[string]struct {
		in   string
		want Format
		ok   bool
	}{
		"srt":              {srtSample, FormatSRT, true},
		"vtt":              {vttSample, FormatVTT, true},
		"ass":              {assSample, FormatASS, true},
		"vtt with bom":     {"\xef\xbb\xbf" + vttSample, FormatVTT, true},
		"vtt beats arrows": {vttSample, FormatVTT, true},
		"not a subtitle":   {"just some prose with no cues at all", "", false},
		"empty":            {"", "", false},
	} {
		t.Run(name, func(t *testing.T) {
			got, ok := DetectFormat([]byte(tc.in))
			if got != tc.want || ok != tc.ok {
				t.Errorf("DetectFormat = %q,%v; want %q,%v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

// TestConvertProducesParseableOutput is the assertion that matters.
//
// It re-parses the output with the reader for the *target* format rather than
// merely checking that a marker string appears. Writing SRT bytes into a file
// named ".vtt" would satisfy any extension- or substring-level check while
// producing a file no player accepts; only round-tripping through the target
// reader catches that.
func TestConvertProducesParseableOutput(t *testing.T) {
	sources := map[Format]string{FormatSRT: srtSample, FormatVTT: vttSample, FormatASS: assSample}
	readers := map[Format]func(*bytes.Reader) (*astisub.Subtitles, error){
		FormatSRT: func(r *bytes.Reader) (*astisub.Subtitles, error) { return astisub.ReadFromSRT(r) },
		FormatVTT: func(r *bytes.Reader) (*astisub.Subtitles, error) { return astisub.ReadFromWebVTT(r) },
		FormatASS: func(r *bytes.Reader) (*astisub.Subtitles, error) { return astisub.ReadFromSSA(r) },
	}

	for from, src := range sources {
		for to := range readers {
			t.Run(string(from)+"->"+string(to), func(t *testing.T) {
				out, err := Convert([]byte(src), to)
				if err != nil {
					t.Fatalf("convert: %v", err)
				}
				sub, err := readers[to](bytes.NewReader(out))
				if err != nil {
					t.Fatalf("output does not parse as %s: %v\n%s", to, err, out)
				}
				if len(sub.Items) != 2 {
					t.Errorf("got %d cues, want 2 — conversion lost content:\n%s", len(sub.Items), out)
				}
				if got, _ := DetectFormat(out); got != to {
					t.Errorf("output sniffs as %s, want %s", got, to)
				}
			})
		}
	}
}

// TestConvertSameFormatIsUntouched pins the passthrough. Round-tripping
// through astisub is lossy for styling it does not model, so the default path
// — where the requested format is the one the provider already sent — must not
// pay that cost.
func TestConvertSameFormatIsUntouched(t *testing.T) {
	out, err := Convert([]byte(srtSample), FormatSRT)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if string(out) != srtSample {
		t.Errorf("same-format conversion rewrote the payload:\ngot  %q\nwant %q", out, srtSample)
	}
}

// TestConvertNormalisesEncodingBeforeParsing pins the ordering.
//
// A legacy-codepage payload parsed as if it were UTF-8 round-trips into
// mojibake that is then written into the new container, where nothing
// downstream can distinguish it from correct text.
func TestConvertNormalisesEncodingBeforeParsing(t *testing.T) {
	// Windows-1252: "Ça va, mon frère ?" — invalid UTF-8.
	legacy := []byte("1\n00:00:01,000 --> 00:00:02,500\n\xc7a va, mon fr\xe8re ?\n")

	out, err := Convert(legacy, FormatVTT)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if !strings.Contains(string(out), "Ça va, mon frère ?") {
		t.Errorf("text was not decoded before conversion: %q", out)
	}
	if _, err := astisub.ReadFromWebVTT(bytes.NewReader(out)); err != nil {
		t.Errorf("output is not valid WebVTT: %v", err)
	}
}

// TestConvertRejectsNonSubtitle pins that junk is an error rather than an
// empty file written under a subtitle name.
func TestConvertRejectsNonSubtitle(t *testing.T) {
	for _, in := range []string{"OK", "<html><body>404</body></html>", ""} {
		if out, err := Convert([]byte(in), FormatVTT); err == nil {
			t.Errorf("Convert(%q) succeeded, returning %q", in, out)
		}
	}
}
