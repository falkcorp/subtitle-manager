// file: pkg/providers/wizdom/wizdom.go
// version: 2.0.1
// guid: 8b2f4d17-5c60-42ae-9d38-71e0a6c4b3f5
// last-edited: 2026-07-31

// Package wizdom implements a real, keyless subtitle provider for wizdom.xyz,
// a Hebrew subtitle database.
//
// Wizdom exposes a small public JSON API that needs no account and no API key:
// /api/search takes an IMDb title ID (plus season and episode for TV) and
// returns candidate subtitles, and /api/files/sub/{id} returns a ZIP archive
// holding the subtitle files.
//
// Two consequences shape this implementation.
//
// The API indexes by IMDb ID, but the Provider interface only supplies a file
// path, so Fetch resolves the parsed title to an IMDb ID itself — see imdb.go.
//
// Wizdom serves Hebrew only, so Fetch rejects any other language outright
// rather than returning Hebrew bytes for an English request. Nothing further
// down the pipeline inspects the language of the text it is handed: the
// scanner would have written Hebrew to "movie.en.srt" and reported success.
package wizdom

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"github.com/jdfalk/subtitle-manager/pkg/metadata"
)

// defaultAPIURL is the public wizdom.xyz API base.
const defaultAPIURL = "https://wizdom.xyz/api"

// hebrewCodes are the language codes this provider answers to. Wizdom hosts
// Hebrew subtitles exclusively. "iw" is the deprecated ISO 639-1 code for
// Hebrew that some tooling still emits, and is accepted for the same reason
// Go's own language tags still recognise it.
var hebrewCodes = map[string]bool{"he": true, "heb": true, "iw": true}

// subtitleExtensions are the archive entries treated as subtitle files when no
// ".srt" entry is present.
var subtitleExtensions = []string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".txt"}

// Client implements the providers.Provider interface for wizdom.xyz.
type Client struct {
	// APIURL is the base URL of the wizdom API (overridable for tests).
	APIURL string
	// IMDbURL is the base URL of the IMDb suggestion service (overridable for tests).
	IMDbURL string
	// HTTPClient is used to make requests.
	HTTPClient *http.Client
	// UserAgent is sent with every request.
	UserAgent string

	// imdbCache memoises title-to-IMDb-ID resolutions for this Client.
	imdbCache imdbCache
}

// New returns a Client configured with reasonable defaults.
func New() *Client {
	return &Client{
		APIURL:     defaultAPIURL,
		IMDbURL:    defaultIMDbURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		UserAgent:  "subtitle-manager",
	}
}

// result is one entry of the /api/search response array.
type result struct {
	ID int `json:"id"`
	// VersionName is the release the subtitle was timed against, e.g.
	// "Game.Of.Thrones.S01E01.720P.BRRIP.x264.AC3-HOPE".
	VersionName string `json:"versioname"`
}

// Fetch downloads a Hebrew subtitle for the media described by mediaPath.
//
// lang must name Hebrew; any other language is an error, because wizdom has
// nothing else to offer and a wrong-language subtitle that "succeeds" is worse
// than a clean miss — it ends the search, so the providers that could have
// found the requested language are never consulted.
func (c *Client) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	if !hebrewCodes[strings.ToLower(strings.TrimSpace(lang))] {
		return nil, fmt.Errorf("wizdom: only Hebrew is available, not %q", lang)
	}

	info, err := metadata.ParseFileName(mediaPath)
	if err != nil {
		return nil, fmt.Errorf("wizdom: parse file name: %w", err)
	}

	imdbID, err := c.resolveIMDbID(ctx, info)
	if err != nil {
		return nil, err
	}

	results, err := c.search(ctx, imdbID, info)
	if err != nil {
		return nil, err
	}
	best := selectBest(results, mediaPath)
	if best == nil {
		return nil, fmt.Errorf("wizdom: no subtitle found for %q", mediaPath)
	}
	return c.download(ctx, best.ID)
}

// search queries /api/search for candidate subtitles.
func (c *Client) search(ctx context.Context, imdbID string, info *metadata.MediaInfo) ([]result, error) {
	q := url.Values{}
	q.Set("action", "by_id")
	q.Set("imdb", imdbID)
	q.Set("version", "1")
	if info.Type == metadata.TypeEpisode {
		q.Set("season", strconv.Itoa(info.Season))
		q.Set("episode", strconv.Itoa(info.Episode))
	}

	resp, err := c.get(ctx, "/search?"+q.Encode())
	if err != nil {
		return nil, fmt.Errorf("wizdom: search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wizdom: search: status %d", resp.StatusCode)
	}
	var out []result
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("wizdom: decode search: %w", err)
	}
	return out, nil
}

// selectBest picks the candidate whose release name best matches the media
// file, and returns nil when there are no candidates.
//
// The API returns a "score" field, but it is 0 on every entry of every
// response observed, so ranking is entirely ours. Candidates are compared by
// how many release tokens they share with the media file name — resolution,
// source, codec and release group all appear as dot-separated tokens in both
// strings — which is what decides whether the timings actually line up.
// Ties keep the API's own order, so the behaviour degrades to "first result"
// exactly when there is nothing to distinguish the candidates.
func selectBest(results []result, mediaPath string) *result {
	var best *result
	bestScore := -1
	// Only the file name: a directory like "/movies/1080p/" would otherwise
	// donate its tokens to every candidate equally and skew the comparison.
	want := releaseTokens(filepath.Base(mediaPath))
	for i := range results {
		if results[i].ID == 0 {
			continue
		}
		score := 0
		for tok := range releaseTokens(results[i].VersionName) {
			if want[tok] {
				score++
			}
		}
		if score > bestScore {
			best, bestScore = &results[i], score
		}
	}
	return best
}

// releaseTokens splits a release name into a set of lowercase alphanumeric
// tokens, dropping single characters that carry no release information.
func releaseTokens(name string) map[string]bool {
	tokens := map[string]bool{}
	for _, tok := range strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		if len(tok) > 1 {
			tokens[tok] = true
		}
	}
	return tokens
}

// download fetches the subtitle archive for id and returns the extracted,
// UTF-8 encoded subtitle bytes.
func (c *Client) download(ctx context.Context, id int) ([]byte, error) {
	resp, err := c.get(ctx, "/files/sub/"+strconv.Itoa(id))
	if err != nil {
		return nil, fmt.Errorf("wizdom: download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wizdom: download: status %d", resp.StatusCode)
	}
	archive, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("wizdom: read archive: %w", err)
	}
	data, err := extractSubtitle(archive)
	if err != nil {
		return nil, err
	}
	return decodeHebrew(data), nil
}

// decodeHebrew converts a wizdom subtitle to UTF-8, leaving it untouched when
// it already is.
//
// Wizdom's archives predominantly hold Windows-1255, the legacy Hebrew
// codepage. That encoding is applied unconditionally rather than sniffed
// because sniffing gets it wrong: charset.DetermineEncoding reports
// ISO-8859-1 for these files, which decodes the same bytes into Latin-1
// mojibake. A Hebrew-only site has exactly one legacy codepage, so naming it
// is both simpler and more accurate than guessing.
//
// On a decode error the original bytes are returned rather than an error, so
// an unexpected encoding costs fidelity but never loses the download.
func decodeHebrew(data []byte) []byte {
	if utf8.Valid(data) {
		return data
	}
	out, _, err := transform.Bytes(charmap.Windows1255.NewDecoder(), data)
	if err != nil {
		return data
	}
	return out
}

// extractSubtitle returns the bytes of the first subtitle-looking file in a ZIP
// archive, preferring a ".srt" entry and falling back to any recognised
// subtitle extension.
func extractSubtitle(archive []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("wizdom: open archive: %w", err)
	}
	var fallback *zip.File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		lower := strings.ToLower(f.Name)
		if strings.HasSuffix(lower, ".srt") {
			return readZipFile(f)
		}
		if fallback == nil && hasSubtitleExtension(lower) {
			fallback = f
		}
	}
	if fallback != nil {
		return readZipFile(fallback)
	}
	return nil, fmt.Errorf("wizdom: no subtitle file in archive")
}

// hasSubtitleExtension reports whether name ends with a known subtitle suffix.
func hasSubtitleExtension(name string) bool {
	for _, ext := range subtitleExtensions {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// readZipFile returns the full decompressed contents of a single archive entry.
func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("wizdom: open %q in archive: %w", f.Name, err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("wizdom: read %q in archive: %w", f.Name, err)
	}
	return b, nil
}

// get issues a GET against the wizdom API.
func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.APIURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	return c.HTTPClient.Do(req)
}
