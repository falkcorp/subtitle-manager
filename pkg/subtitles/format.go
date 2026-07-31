// file: pkg/subtitles/format.go
// version: 1.0.0
// guid: 7e51c9a3-2d84-4f60-b17c-8039e5a2461d
// last-edited: 2026-07-31

package subtitles

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/asticode/go-astisub"
	"github.com/spf13/viper"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// Format identifies a subtitle container this application can write.
type Format string

// The formats a download may be written as.
const (
	FormatSRT Format = "srt"
	FormatVTT Format = "vtt"
	FormatASS Format = "ass"
)

// DefaultFormat is what a download is written as when nothing is configured.
// SRT is the most widely supported container and is what every existing
// installation already has on disk, so it stays the default.
const DefaultFormat = FormatSRT

// utf8BOM is stripped from decoded output.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// ParseFormat validates a caller-supplied format name.
//
// The result reaches a file name that is joined onto a media directory, so
// this is an allowlist and unknown input is rejected rather than sanitised.
// Sanitising a path fragment invites the one case the filter missed; refusing
// anything not on a three-entry list does not.
func ParseFormat(s string) (Format, error) {
	switch f := Format(strings.ToLower(strings.TrimSpace(s))); f {
	case "":
		return DefaultFormat, nil
	case FormatSRT, FormatVTT, FormatASS:
		return f, nil
	default:
		return "", fmt.Errorf("unsupported subtitle format %q (want srt, vtt or ass)", s)
	}
}

// Ext returns the file extension for f, leading dot included.
func (f Format) Ext() string { return "." + string(f) }

// EncodeUTF8 converts subtitle bytes to UTF-8. Data that is already valid
// UTF-8 is returned unchanged, minus any BOM. Otherwise the charset is
// detected (Windows-1252, ISO-8859-1 and friends) and transcoded. On a
// detection or decode failure the original bytes are returned, so normalising
// never loses a download.
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

// DetectFormat identifies the container a subtitle payload is already in.
//
// Format is inferred from the bytes because the download path never has a file
// name to go on: a provider hands back an anonymous []byte, and astisub's
// readers each require the format to be known before they are called.
func DetectFormat(data []byte) (Format, bool) {
	head := data
	if len(head) > 4096 {
		head = head[:4096]
	}
	lower := bytes.ToLower(bytes.TrimPrefix(head, utf8BOM))

	// WebVTT is required to open with the WEBVTT signature.
	if bytes.HasPrefix(bytes.TrimLeft(lower, " \t\r\n"), []byte("webvtt")) {
		return FormatVTT, true
	}
	// SubStation Alpha carries section headers; both v4 and v4+ are read and
	// written by astisub's SSA support.
	if bytes.Contains(lower, []byte("[script info]")) ||
		bytes.Contains(lower, []byte("[v4+ styles]")) ||
		bytes.Contains(lower, []byte("[v4 styles]")) ||
		bytes.Contains(lower, []byte("[events]")) {
		return FormatASS, true
	}
	// Anything else with a cue arrow is SRT.
	if bytes.Contains(head, []byte("-->")) {
		return FormatSRT, true
	}
	return "", false
}

// Convert re-encodes a subtitle payload into want.
//
// Input that is already in the target format is returned untouched rather than
// parsed and re-serialised. That is deliberate: a round trip through astisub
// is lossy for anything it does not model — styling, positioning, unusual
// cue metadata — and there is no reason to pay that cost on the default path,
// where the requested format is the one the provider already sent.
//
// Bytes are normalised to UTF-8 first. Parsing has to happen on decoded text:
// a Windows-1255 or 1252 payload fed to a parser expecting UTF-8 round-trips
// into mojibake that is then written into the new container, where nothing
// downstream can tell it from correctly converted text.
func Convert(data []byte, want Format) ([]byte, error) {
	if want == "" {
		want = DefaultFormat
	}
	data = EncodeUTF8(data)

	have, ok := DetectFormat(data)
	if !ok {
		return nil, fmt.Errorf("subtitles: unrecognised subtitle format")
	}
	if have == want {
		return data, nil
	}

	var (
		sub *astisub.Subtitles
		err error
	)
	switch have {
	case FormatSRT:
		sub, err = astisub.ReadFromSRT(bytes.NewReader(data))
	case FormatVTT:
		sub, err = astisub.ReadFromWebVTT(bytes.NewReader(data))
	case FormatASS:
		sub, err = astisub.ReadFromSSA(bytes.NewReader(data))
	}
	if err != nil {
		return nil, fmt.Errorf("subtitles: read %s: %w", have, err)
	}

	buf := &bytes.Buffer{}
	switch want {
	case FormatSRT:
		err = sub.WriteToSRT(buf)
	case FormatVTT:
		err = sub.WriteToWebVTT(buf)
	case FormatASS:
		err = sub.WriteToSSA(buf)
	}
	if err != nil {
		return nil, fmt.Errorf("subtitles: write %s: %w", want, err)
	}
	return buf.Bytes(), nil
}

// SingleLanguageNaming reports whether subtitles are written without the
// language code in the file name ("<base>.srt" rather than
// "<base>.<lang>.srt"), matching Bazarr's single-language naming option.
func SingleLanguageNaming() bool { return viper.GetBool("subtitles.single_language") }
