// file: pkg/providers/podnapisi/podnapisi.go
// version: 2.0.0
// guid: 1dc059b8-703d-4392-9b37-b0fb79f8eabe
// last-edited: 2026-07-24

// Package podnapisi implements a real, keyless subtitle provider for
// podnapisi.net using its public advanced-search JSON API.
//
// Podnapisi indexes subtitles by title, season, and episode (for TV) or title
// and year (for movies) rather than by file hash, so Fetch parses the media
// file name into a (title, type, season, episode, year) tuple via pkg/metadata,
// queries /subtitles/search/advanced, selects the first result whose media type
// matches, and downloads it. Downloads arrive as a ZIP archive, so Fetch
// extracts the subtitle file from the archive before returning it.
//
// No account or API key is required: the search endpoint only needs an
// Accept: application/json header, and the download endpoint is public.
package podnapisi

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/metadata"
)

// defaultAPIURL is the public Podnapisi subtitles base URL. Both the search
// endpoint ("/search/advanced") and download endpoint ("/{id}/download") hang
// off it.
const defaultAPIURL = "https://www.podnapisi.net/subtitles"

// subtitleExtensions are the archive entries Fetch treats as subtitle files
// when no ".srt" entry is present.
var subtitleExtensions = []string{".srt", ".ass", ".ssa", ".vtt", ".sub", ".txt"}

// Client implements the providers.Provider interface for podnapisi.net.
type Client struct {
	// APIURL is the base URL of the Podnapisi subtitles API (overridable for tests).
	APIURL string
	// HTTPClient is used to make requests.
	HTTPClient *http.Client
	// UserAgent is sent with every request; some Podnapisi edges reject empty agents.
	UserAgent string
}

// New returns a Client configured with reasonable defaults.
func New() *Client {
	return &Client{
		APIURL:     defaultAPIURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		UserAgent:  "subtitle-manager",
	}
}

// searchResponse is the /search/advanced response envelope.
type searchResponse struct {
	Data     []result `json:"data"`
	Page     int      `json:"page"`
	AllPages int      `json:"all_pages"`
}

// result mirrors one entry of the search "data" array. Only the fields Fetch
// needs are decoded.
type result struct {
	ID       string      `json:"id"`
	Language string      `json:"language"`
	Movie    resultMovie `json:"movie"`
}

// resultMovie carries the media-type discriminator ("movie", "tv-series", or
// "mini-series") used to reject entries of the wrong kind.
type resultMovie struct {
	Type string `json:"type"`
}

// Fetch downloads a subtitle for the media described by mediaPath in lang.
// It returns the subtitle bytes or an error when the file name cannot be
// parsed, no matching subtitle is found, or the archive holds no subtitle file.
//
// lang is forwarded to Podnapisi as-is: the API expects the ISO 639-1 code
// ("en"), which is exactly what the scanner passes (verified against
// subliminal's recorded requests), so no code conversion is performed.
func (c *Client) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	info, err := metadata.ParseFileName(mediaPath)
	if err != nil {
		return nil, fmt.Errorf("podnapisi: parse file name: %w", err)
	}
	if strings.TrimSpace(info.Title) == "" {
		return nil, fmt.Errorf("podnapisi: could not determine title from %q", mediaPath)
	}

	res, err := c.search(ctx, info, lang)
	if err != nil {
		return nil, err
	}
	if res.ID == "" {
		return nil, fmt.Errorf("podnapisi: no subtitle found for %q (%s)", mediaPath, lang)
	}
	return c.download(ctx, res.ID)
}

// search queries the advanced-search endpoint and returns the first result
// whose media type matches the parsed file. The server filters by language,
// season, and episode, but the type check guards against a movie result being
// returned for an episode query (and vice versa).
func (c *Client) search(ctx context.Context, info *metadata.MediaInfo, lang string) (result, error) {
	q := url.Values{}
	q.Set("keywords", info.Title)
	q.Set("language", lang)
	if info.Type == metadata.TypeEpisode {
		q.Set("seasons", strconv.Itoa(info.Season))
		q.Set("episodes", strconv.Itoa(info.Episode))
		// Podnapisi distinguishes full series from mini-series; request both so
		// either kind of TV entry matches.
		q.Add("movie_type", "tv-series")
		q.Add("movie_type", "mini-series")
	} else {
		q.Set("movie_type", "movie")
		if info.Year > 0 {
			q.Set("year", strconv.Itoa(info.Year))
		}
	}

	resp, err := c.get(ctx, "/search/advanced?"+q.Encode())
	if err != nil {
		return result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return result{}, fmt.Errorf("podnapisi: search: status %d", resp.StatusCode)
	}
	var out searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return result{}, fmt.Errorf("podnapisi: decode search: %w", err)
	}

	wantMovie := info.Type == metadata.TypeMovie
	for _, r := range out.Data {
		if r.ID == "" {
			continue
		}
		if (r.Movie.Type == "movie") != wantMovie {
			continue
		}
		return r, nil
	}
	return result{}, nil
}

// download fetches the subtitle archive for id and returns the extracted
// subtitle bytes. Podnapisi serves a ZIP even for a single subtitle.
func (c *Client) download(ctx context.Context, id string) ([]byte, error) {
	resp, err := c.get(ctx, "/"+url.PathEscape(id)+"/download?container=zip")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("podnapisi: download: status %d", resp.StatusCode)
	}
	archive, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("podnapisi: read archive: %w", err)
	}
	return extractSubtitle(archive)
}

// extractSubtitle returns the bytes of the first subtitle-looking file in a ZIP
// archive, preferring a ".srt" entry and falling back to any recognised
// subtitle extension. It errors when the archive is unreadable or holds none.
func extractSubtitle(archive []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("podnapisi: open archive: %w", err)
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
	return nil, fmt.Errorf("podnapisi: no subtitle file in archive")
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
		return nil, fmt.Errorf("podnapisi: open %q in archive: %w", f.Name, err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("podnapisi: read %q in archive: %w", f.Name, err)
	}
	return b, nil
}

// get issues a GET against the Podnapisi API, joining path onto APIURL and
// attaching the configured User-Agent and a JSON Accept header (the search
// endpoint returns HTML unless application/json is requested).
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
