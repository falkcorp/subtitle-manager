// file: pkg/providers/wizdom/imdb.go
// version: 1.0.0
// guid: 4a1e0c72-9f38-4d51-b6ad-2c7e5f0913da
// last-edited: 2026-07-31

package wizdom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/jdfalk/subtitle-manager/pkg/metadata"
)

// defaultIMDbURL is IMDb's public title-suggestion service — the same endpoint
// that backs the search box on imdb.com. It needs no key, no account and no
// registration, which is what keeps this provider keyless.
const defaultIMDbURL = "https://v2.sg.media-imdb.com/suggestion"

// imdbIDPattern guards against a malformed "id" being pasted into a URL.
var imdbIDPattern = regexp.MustCompile(`^tt\d+$`)

// suggestionResponse is the envelope returned by the suggestion service.
type suggestionResponse struct {
	D []suggestion `json:"d"`
}

// suggestion is one candidate title. The field names are IMDb's own
// single-letter keys: "l" is the title, "y" the year, "qid" the kind of title.
type suggestion struct {
	ID    string `json:"id"`
	Title string `json:"l"`
	Year  int    `json:"y"`
	QID   string `json:"qid"`
}

// episodeQIDs are the "qid" values that denote a TV title with episodes.
// A "tvEpisode" is deliberately absent: wizdom is queried with a *series*
// IMDb ID plus season and episode numbers, not with an episode's own ID.
var episodeQIDs = map[string]bool{
	"tvSeries":     true,
	"tvMiniSeries": true,
}

// resolveIMDbID maps parsed media metadata to an IMDb title ID.
//
// # Why the provider has to do this itself
//
// The Provider interface is Fetch(ctx, mediaPath, lang): a file path and a
// language, with no channel for metadata. Wizdom indexes strictly by IMDb ID,
// so a lookup has to happen somewhere, and here is the only place it can.
//
// # Failing closed
//
// The match is filtered on title kind, and on year for movies, and returns an
// error rather than a guess when nothing qualifies. This matters more than it
// looks: a fuzzy title match that returns a TV series' ID for a movie query
// does not fail, it succeeds and downloads confidently wrong subtitles for the
// wrong show. A miss that reports "no match" is recoverable; a plausible wrong
// answer written to disk next to the media is not.
func (c *Client) resolveIMDbID(ctx context.Context, info *metadata.MediaInfo) (string, error) {
	title := strings.TrimSpace(info.Title)
	if title == "" {
		return "", fmt.Errorf("wizdom: empty title")
	}

	key := imdbCacheKey{title: strings.ToLower(title), year: info.Year, kind: info.Type}
	if id, ok := c.imdbCache.load(key); ok {
		return id, nil
	}

	list, err := c.suggest(ctx, title)
	if err != nil {
		return "", err
	}

	wantEpisode := info.Type == metadata.TypeEpisode
	for _, s := range list {
		if !imdbIDPattern.MatchString(s.ID) {
			continue
		}
		if episodeQIDs[s.QID] != wantEpisode {
			continue
		}
		// Only movies carry a reliable year in a file name; a series' file name
		// year, when present at all, is usually the *series* start year and is
		// checked the same way. Allow a one-year drift: a release dated by its
		// festival or its wide release routinely disagrees with IMDb by one.
		if info.Year > 0 && s.Year > 0 && abs(s.Year-info.Year) > 1 {
			continue
		}
		c.imdbCache.store(key, s.ID)
		return s.ID, nil
	}
	return "", fmt.Errorf("wizdom: no IMDb match for %q (%d)", title, info.Year)
}

// suggest queries the suggestion service for a title.
//
// The path carries a single-letter shard before the query. IMDb currently
// ignores it — "/t/game of thrones.json" and "/g/game of thrones.json" return
// byte-identical responses, verified directly — but the site itself sends the
// query's own first letter, so that is what is sent here rather than a
// hard-coded constant that would break if the shard ever started being
// enforced.
func (c *Client) suggest(ctx context.Context, title string) ([]suggestion, error) {
	q := normalizeQuery(title)
	if q == "" {
		return nil, fmt.Errorf("wizdom: title %q has no searchable characters", title)
	}
	endpoint := fmt.Sprintf("%s/%s/%s.json", c.IMDbURL, string(q[0]), url.PathEscape(q))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wizdom: imdb suggest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wizdom: imdb suggest: status %d", resp.StatusCode)
	}
	var out suggestionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("wizdom: decode imdb suggest: %w", err)
	}
	return out.D, nil
}

// normalizeQuery lowercases a title and keeps only letters, digits and single
// spaces, matching what the suggestion service expects in its path segment.
func normalizeQuery(title string) string {
	var b strings.Builder
	lastSpace := true
	for _, r := range strings.ToLower(title) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastSpace = false
		case !lastSpace:
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// abs returns the absolute value of n.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// imdbCacheKey identifies one resolution request.
type imdbCacheKey struct {
	title string
	year  int
	kind  metadata.MediaType
}

// imdbCache memoises resolutions for the lifetime of the Client.
//
// A library scan calls Fetch once per file, and every episode of a series
// resolves the same title. Without this, scanning a 60-episode show sends 60
// identical requests to an undocumented third-party endpoint, which is both
// wasteful and the sort of traffic that gets an IP blocked.
type imdbCache struct {
	mu sync.RWMutex
	m  map[imdbCacheKey]string
}

// load returns a cached ID.
func (c *imdbCache) load(k imdbCacheKey) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	id, ok := c.m[k]
	return id, ok
}

// store records a resolved ID.
func (c *imdbCache) store(k imdbCacheKey, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = map[imdbCacheKey]string{}
	}
	c.m[k] = id
}
