// file: pkg/providers/gestdown/gestdown.go
// version: 2.0.0
// guid: 898074d8-d1d4-4681-936c-e4c2ed0ddaa3
// last-edited: 2026-07-23

// Package gestdown implements a real, keyless subtitle provider for
// gestdown.info, a free REST proxy over Addic7ed's TV-subtitle catalogue.
//
// Addic7ed indexes subtitles by show name, season, and episode rather than by
// file hash, so Fetch parses the media file name into a (series, season,
// episode) tuple via pkg/metadata, resolves the show through Gestdown's search
// endpoint, lists the completed subtitles for the requested language, and
// downloads the first match. No account or API key is required.
//
// Movies are unsupported: Gestdown/Addic7ed is TV-only, and Fetch returns an
// error for anything that does not parse as an episode.
package gestdown

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/metadata"
)

// defaultAPIURL is the public Gestdown API base URL.
const defaultAPIURL = "https://api.gestdown.info"

// Client implements the providers.Provider interface for gestdown.info.
type Client struct {
	// APIURL is the base URL of the Gestdown API (overridable for tests).
	APIURL string
	// HTTPClient is used to make requests.
	HTTPClient *http.Client
	// UserAgent is sent with every request; Gestdown rejects empty agents.
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

// show mirrors one entry of the /shows/search response.
type show struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Seasons []int  `json:"seasons"`
}

// showSearchResponse is the /shows/search/{name} response envelope.
type showSearchResponse struct {
	Shows []show `json:"shows"`
}

// subtitle mirrors one entry of the matchingSubtitles array.
type subtitle struct {
	SubtitleID  string `json:"subtitleId"`
	Version     string `json:"version"`
	Completed   bool   `json:"completed"`
	DownloadURI string `json:"downloadUri"`
	Language    string `json:"language"`
}

// subtitlesResponse is the /subtitles/get/... response envelope.
type subtitlesResponse struct {
	MatchingSubtitles []subtitle `json:"matchingSubtitles"`
}

// Fetch downloads a subtitle for the episode described by mediaPath in lang.
// It returns the subtitle bytes or an error when the file is not a TV episode,
// the show cannot be resolved, or no completed subtitle exists.
func (c *Client) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	info, err := metadata.ParseFileName(mediaPath)
	if err != nil {
		return nil, fmt.Errorf("gestdown: parse file name: %w", err)
	}
	if info.Type != metadata.TypeEpisode {
		return nil, fmt.Errorf("gestdown: only TV episodes are supported (got %q)", mediaPath)
	}
	if strings.TrimSpace(info.Title) == "" {
		return nil, fmt.Errorf("gestdown: could not determine series name from %q", mediaPath)
	}

	shows, err := c.searchShows(ctx, info.Title)
	if err != nil {
		return nil, err
	}
	if len(shows) == 0 {
		return nil, fmt.Errorf("gestdown: show %q not found", info.Title)
	}

	sub, err := c.findSubtitle(ctx, shows, info.Season, info.Episode, lang)
	if err != nil {
		return nil, err
	}
	return c.download(ctx, sub.DownloadURI)
}

// searchShows resolves a series name to candidate shows.
func (c *Client) searchShows(ctx context.Context, title string) ([]show, error) {
	resp, err := c.get(ctx, "/shows/search/"+url.PathEscape(title))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gestdown: show search: status %d", resp.StatusCode)
	}
	var out showSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("gestdown: decode show search: %w", err)
	}
	return out.Shows, nil
}

// findSubtitle walks the candidate shows in relevance order and returns the
// first completed subtitle for the requested episode and language.
func (c *Client) findSubtitle(ctx context.Context, shows []show, season, episode int, lang string) (subtitle, error) {
	for _, sh := range orderShows(shows, season) {
		subs, err := c.listSubtitles(ctx, sh.ID, season, episode, lang)
		if err != nil {
			return subtitle{}, err
		}
		for _, s := range subs {
			if s.Completed && s.DownloadURI != "" {
				return s, nil
			}
		}
	}
	return subtitle{}, fmt.Errorf("gestdown: no completed subtitle for S%02dE%02d (%s)", season, episode, lang)
}

// orderShows ranks candidate shows so those whose season list covers the target
// season are tried first, preserving the API's original order within each group.
func orderShows(shows []show, season int) []show {
	preferred := make([]show, 0, len(shows))
	rest := make([]show, 0, len(shows))
	for _, sh := range shows {
		if containsInt(sh.Seasons, season) {
			preferred = append(preferred, sh)
		} else {
			rest = append(rest, sh)
		}
	}
	return append(preferred, rest...)
}

// containsInt reports whether v appears in xs.
func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// listSubtitles returns the matching subtitles for a show/episode/language.
// A 404 (show or episode not indexed) yields an empty slice rather than an error
// so the caller can try the next candidate show.
//
// lang is forwarded to Gestdown as-is: the API treats ISO 639-1 ("en"),
// ISO 639-2 ("eng"), and the English language name ("English") as equivalent
// (verified live), and the scanner passes 2-letter profile codes, so no
// code→name conversion is needed. Regional variants are distinct, however —
// "pt-BR" and "pt" resolve to different Addic7ed catalogues.
func (c *Client) listSubtitles(ctx context.Context, showID string, season, episode int, lang string) ([]subtitle, error) {
	path := fmt.Sprintf("/subtitles/get/%s/%d/%d/%s", url.PathEscape(showID), season, episode, url.PathEscape(lang))
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gestdown: list subtitles: status %d", resp.StatusCode)
	}
	var out subtitlesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("gestdown: decode subtitles: %w", err)
	}
	return out.MatchingSubtitles, nil
}

// download fetches the subtitle bytes from a relative downloadUri.
func (c *Client) download(ctx context.Context, downloadURI string) ([]byte, error) {
	if downloadURI == "" {
		return nil, fmt.Errorf("gestdown: empty download URI")
	}
	resp, err := c.get(ctx, downloadURI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gestdown: download: status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// get issues a GET against the Gestdown API, joining path onto APIURL and
// attaching the configured User-Agent.
func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.APIURL+path, nil)
	if err != nil {
		return nil, err
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	return c.HTTPClient.Do(req)
}
