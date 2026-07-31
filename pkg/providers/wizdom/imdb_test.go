// file: pkg/providers/wizdom/imdb_test.go
// version: 1.0.0
// guid: 9c4f7a20-6e13-4b85-a0d7-52f8b1e3609c
// last-edited: 2026-07-31

package wizdom

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jdfalk/subtitle-manager/pkg/metadata"
)

// newIMDbServer serves body for every request and counts the calls, recording
// the last path so the shard letter can be asserted.
func newIMDbServer(t *testing.T, body string) (*Client, *int32, *string) {
	t.Helper()
	var calls int32
	var lastPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		lastPath = r.URL.Path
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)

	c := New()
	c.IMDbURL = srv.URL
	return c, &calls, &lastPath
}

// mixedSuggest holds a movie and a series sharing a name, which is the case
// that makes filtering on title kind load-bearing.
const mixedSuggest = `{"d":[
  {"id":"tt0903747","l":"Breaking Bad","y":2008,"qid":"tvSeries"},
  {"id":"tt2301451","l":"Breaking Bad","y":2013,"qid":"movie"}
]}`

// TestResolveIMDbIDPicksMatchingKind pins that a movie query never resolves to
// a series and vice versa. Both entries here are plausible title matches, so
// only the kind filter separates them — and picking the wrong one does not
// fail, it downloads confidently wrong subtitles.
func TestResolveIMDbIDPicksMatchingKind(t *testing.T) {
	for name, tc := range map[string]struct {
		info *metadata.MediaInfo
		want string
	}{
		"episode resolves to the series": {
			&metadata.MediaInfo{Type: metadata.TypeEpisode, Title: "Breaking Bad", Season: 1, Episode: 1},
			"tt0903747",
		},
		"movie resolves to the movie": {
			&metadata.MediaInfo{Type: metadata.TypeMovie, Title: "Breaking Bad", Year: 2013},
			"tt2301451",
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, _ := newIMDbServer(t, mixedSuggest)
			got, err := c.resolveIMDbID(context.Background(), tc.info)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if got != tc.want {
				t.Errorf("resolved %s, want %s", got, tc.want)
			}
		})
	}
}

// TestResolveIMDbIDFailsClosed pins that no confident match is an error rather
// than a guess. Returning the nearest title regardless of kind or year is the
// failure mode this whole filter exists to prevent.
func TestResolveIMDbIDFailsClosed(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		info *metadata.MediaInfo
	}{
		"no results at all": {
			`{"d":[]}`,
			&metadata.MediaInfo{Type: metadata.TypeMovie, Title: "Nonexistent Film", Year: 2020},
		},
		"only a series when a movie was asked for": {
			`{"d":[{"id":"tt0903747","l":"Breaking Bad","y":2008,"qid":"tvSeries"}]}`,
			&metadata.MediaInfo{Type: metadata.TypeMovie, Title: "Breaking Bad", Year: 2013},
		},
		"right kind but the year is far off": {
			`{"d":[{"id":"tt0111161","l":"The Shawshank Redemption","y":1994,"qid":"movie"}]}`,
			&metadata.MediaInfo{Type: metadata.TypeMovie, Title: "The Shawshank Redemption", Year: 2015},
		},
		"malformed id": {
			`{"d":[{"id":"../../etc/passwd","l":"Movie","y":2020,"qid":"movie"}]}`,
			&metadata.MediaInfo{Type: metadata.TypeMovie, Title: "Movie", Year: 2020},
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, _, _ := newIMDbServer(t, tc.body)
			got, err := c.resolveIMDbID(context.Background(), tc.info)
			if err == nil {
				t.Fatalf("resolve succeeded with %q; want an error rather than a guess", got)
			}
		})
	}
}

// TestResolveIMDbIDToleratesYearDrift pins the one-year allowance. A film
// dated by its festival premiere routinely disagrees with the release its file
// name was cut from, and rejecting those would lose real matches.
func TestResolveIMDbIDToleratesYearDrift(t *testing.T) {
	body := `{"d":[{"id":"tt0111161","l":"The Shawshank Redemption","y":1994,"qid":"movie"}]}`
	for _, year := range []int{1993, 1994, 1995} {
		c, _, _ := newIMDbServer(t, body)
		info := &metadata.MediaInfo{Type: metadata.TypeMovie, Title: "The Shawshank Redemption", Year: year}
		if _, err := c.resolveIMDbID(context.Background(), info); err != nil {
			t.Errorf("year %d rejected: %v", year, err)
		}
	}
}

// TestResolveIMDbIDCaches pins the memoisation. Scanning a series calls Fetch
// once per episode and every one resolves the same title, so without a cache a
// 60-episode show sends 60 identical requests to a third-party endpoint.
func TestResolveIMDbIDCaches(t *testing.T) {
	c, calls, _ := newIMDbServer(t, seriesSuggest)
	for ep := 1; ep <= 5; ep++ {
		info := &metadata.MediaInfo{Type: metadata.TypeEpisode, Title: "Game of Thrones", Season: 1, Episode: ep}
		if _, err := c.resolveIMDbID(context.Background(), info); err != nil {
			t.Fatalf("episode %d: %v", ep, err)
		}
	}
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Errorf("made %d IMDb requests for 5 episodes of one series, want 1", got)
	}
}

// TestSuggestUsesQueryFirstLetter pins the shard segment. IMDb currently
// ignores it — "/t/game of thrones.json" and "/g/game of thrones.json" return
// identical responses — but sending the query's own first letter is what the
// site does, and costs nothing if the shard ever starts being enforced.
func TestSuggestUsesQueryFirstLetter(t *testing.T) {
	c, _, lastPath := newIMDbServer(t, seriesSuggest)
	info := &metadata.MediaInfo{Type: metadata.TypeEpisode, Title: "Game of Thrones", Season: 1, Episode: 1}
	if _, err := c.resolveIMDbID(context.Background(), info); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !strings.HasPrefix(*lastPath, "/g/") {
		t.Errorf("suggestion path %q does not shard on the query's first letter", *lastPath)
	}
}

// TestNormalizeQuery pins the title cleanup that builds the request path.
func TestNormalizeQuery(t *testing.T) {
	for in, want := range map[string]string{
		"The Shawshank Redemption": "the shawshank redemption",
		"Game.of.Thrones":          "game of thrones",
		"Spider-Man: No Way Home":  "spider man no way home",
		"WALL·E":                   "wall e",
		"  padded  ":               "padded",
		"!!!":                      "",
	} {
		if got := normalizeQuery(in); got != want {
			t.Errorf("normalizeQuery(%q) = %q, want %q", in, got, want)
		}
	}
}
