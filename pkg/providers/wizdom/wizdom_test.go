// file: pkg/providers/wizdom/wizdom_test.go
// version: 2.0.1
// guid: 5d9a3e08-7b14-4c62-8f01-3a6b2d70e491
// last-edited: 2026-07-31

package wizdom

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// hebrewCP1255Body is the opening of a real subtitle downloaded from
// wizdom.xyz: a valid SRT whose text is Windows-1255 Hebrew, so the bytes are
// not valid UTF-8.
var hebrewCP1255Body = []byte("1\r\n00:00:54,542 --> 00:00:59,323\r\n" +
	"-\xe7\xe5\xee\xe5\xfa \xf9\xec \xfa\xf7\xe5\xe5\xe4-\r\n")

// wantHebrewUTF8 is the same text after decoding, i.e. what Fetch must return.
const wantHebrewUTF8 = "-חומות של תקווה-"

// newZIP builds a ZIP archive holding one entry.
func newZIP(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create(name)
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := f.Write(body); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

// recordedRequests captures what a Client asked for.
type recordedRequests struct {
	imdb   string
	search string
	sub    string
}

// newTestClient wires a Client to two httptest servers standing in for the
// IMDb suggestion service and the wizdom API. searchBody is served from
// /search; the archive is served from /files/sub/{id}.
func newTestClient(t *testing.T, suggestBody, searchBody string, archive []byte) (*Client, *recordedRequests) {
	t.Helper()
	rec := &recordedRequests{}

	imdbSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.imdb = r.URL.Path
		fmt.Fprint(w, suggestBody)
	}))
	t.Cleanup(imdbSrv.Close)

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/search"):
			rec.search = r.URL.String()
			fmt.Fprint(w, searchBody)
		case strings.HasPrefix(r.URL.Path, "/files/sub/"):
			rec.sub = r.URL.Path
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(apiSrv.Close)

	c := New()
	c.APIURL = apiSrv.URL
	c.IMDbURL = imdbSrv.URL
	return c, rec
}

// movieSuggest mirrors a real IMDb suggestion response for a movie.
const movieSuggest = `{"d":[
  {"id":"tt0111161","l":"The Shawshank Redemption","y":1994,"qid":"movie"},
  {"id":"tt5027774","l":"Shawshank: The Redeeming Feature","y":2001,"qid":"tvMovie"}
]}`

// seriesSuggest mirrors a real IMDb suggestion response for a TV series.
const seriesSuggest = `{"d":[
  {"id":"tt0944947","l":"Game of Thrones","y":2011,"qid":"tvSeries"}
]}`

// TestFetchMovie walks the whole path for a movie: title resolved to an IMDb
// ID, searched, and the archive extracted and transcoded.
func TestFetchMovie(t *testing.T) {
	archive := newZIP(t, "Shawshank.srt", hebrewCP1255Body)
	search := `[{"id":87292,"versioname":"Shawshank.Redemption.DVDRip.XviD-Flaket","score":0}]`
	c, rec := newTestClient(t, movieSuggest, search, archive)

	got, err := c.Fetch(context.Background(), "/media/The Shawshank Redemption (1994).mkv", "he")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}

	if !strings.Contains(string(got), wantHebrewUTF8) {
		t.Errorf("subtitle was not transcoded to UTF-8: got %q", got)
	}
	if !strings.Contains(rec.search, "imdb=tt0111161") {
		t.Errorf("wrong IMDb ID in search: %s", rec.search)
	}
	if strings.Contains(rec.search, "season=") || strings.Contains(rec.search, "episode=") {
		t.Errorf("movie query must not carry season/episode: %s", rec.search)
	}
	if rec.sub != "/files/sub/87292" {
		t.Errorf("wrong download path: %s", rec.sub)
	}
}

// TestFetchEpisode pins that TV queries carry season and episode, the only
// thing distinguishing one episode's subtitles from another's.
func TestFetchEpisode(t *testing.T) {
	archive := newZIP(t, "got.srt", hebrewCP1255Body)
	search := `[{"id":88452,"versioname":"Game.Of.Thrones.S01E01.720P.BRRIP.x264.AC3-HOPE","score":0}]`
	c, rec := newTestClient(t, seriesSuggest, search, archive)

	if _, err := c.Fetch(context.Background(), "/tv/Game.of.Thrones.S01E02.1080p.mkv", "he"); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	for _, want := range []string{"imdb=tt0944947", "season=1", "episode=2"} {
		if !strings.Contains(rec.search, want) {
			t.Errorf("search query missing %q: %s", want, rec.search)
		}
	}
}

// TestFetchRejectsNonHebrew is the guard that matters most.
//
// Wizdom hosts Hebrew only and its API takes no language parameter, so a
// request for English would otherwise return Hebrew bytes, which the scanner
// writes to "movie.en.srt" and reports as a success — and, worse, the
// "successful" fetch ends the search before any provider that actually has
// English is consulted.
func TestFetchRejectsNonHebrew(t *testing.T) {
	archive := newZIP(t, "x.srt", hebrewCP1255Body)
	search := `[{"id":1,"versioname":"whatever","score":0}]`
	c, rec := newTestClient(t, movieSuggest, search, archive)

	for _, lang := range []string{"en", "es", "fr", ""} {
		t.Run("lang="+lang, func(t *testing.T) {
			if _, err := c.Fetch(context.Background(), "/media/Movie (1994).mkv", lang); err == nil {
				t.Fatalf("Fetch(%q) succeeded; a non-Hebrew request must fail", lang)
			}
		})
	}
	if rec.search != "" {
		t.Errorf("a rejected language must not reach the API, but it queried %s", rec.search)
	}
}

// TestFetchAcceptsHebrewAliases covers the codes callers actually emit.
func TestFetchAcceptsHebrewAliases(t *testing.T) {
	archive := newZIP(t, "x.srt", hebrewCP1255Body)
	search := `[{"id":1,"versioname":"whatever","score":0}]`
	for _, lang := range []string{"he", "heb", "iw", "HE", " he "} {
		t.Run("lang="+lang, func(t *testing.T) {
			c, _ := newTestClient(t, movieSuggest, search, archive)
			if _, err := c.Fetch(context.Background(), "/media/Movie (1994).mkv", lang); err != nil {
				t.Fatalf("Fetch(%q) failed: %v", lang, err)
			}
		})
	}
}

// TestSelectBestPrefersMatchingRelease pins the ranking. The API scores every
// entry 0, so picking the release whose tokens match the media file is the
// only thing keeping the timings aligned.
func TestSelectBestPrefersMatchingRelease(t *testing.T) {
	results := []result{
		{ID: 1, VersionName: "Game.of.Thrones.S01E01.DVDRip.XviD-NOGRP"},
		{ID: 2, VersionName: "Game.of.Thrones.S01E01.1080p.BluRay.x264-CtrlHD"},
		{ID: 3, VersionName: "Game.of.Thrones.S01E01.HDTV.x264-BATV"},
	}
	got := selectBest(results, "/tv/Game.of.Thrones.S01E01.1080p.BluRay.x264-CtrlHD.mkv")
	if got == nil || got.ID != 2 {
		t.Fatalf("selectBest picked %v, want the matching 1080p BluRay release (id 2)", got)
	}
}

// TestSelectBestEmpty pins that no candidates means no pick, rather than a
// zero-value result whose ID 0 would be downloaded as /files/sub/0.
func TestSelectBestEmpty(t *testing.T) {
	if got := selectBest(nil, "/media/x.mkv"); got != nil {
		t.Fatalf("selectBest(nil) = %v, want nil", got)
	}
	if got := selectBest([]result{{ID: 0, VersionName: "junk"}}, "/media/x.mkv"); got != nil {
		t.Fatalf("selectBest ignored a zero ID: %v", got)
	}
}

// TestFetchNoResults pins that an empty search is an error, not a download.
func TestFetchNoResults(t *testing.T) {
	c, rec := newTestClient(t, movieSuggest, `[]`, nil)
	if _, err := c.Fetch(context.Background(), "/media/Movie (1994).mkv", "he"); err == nil {
		t.Fatal("Fetch succeeded with no search results")
	}
	if rec.sub != "" {
		t.Errorf("downloaded despite no results: %s", rec.sub)
	}
}

// TestDecodeHebrewLeavesUTF8Alone pins that an archive already in UTF-8 is
// passed through untouched rather than double-decoded into mojibake.
func TestDecodeHebrewLeavesUTF8Alone(t *testing.T) {
	utf8Body := []byte("1\n00:00:01,000 --> 00:00:02,000\n" + wantHebrewUTF8 + "\n")
	if got := decodeHebrew(utf8Body); !bytes.Equal(got, utf8Body) {
		t.Errorf("decodeHebrew mangled valid UTF-8: %q", got)
	}
}

// TestExtractSubtitlePrefersSRT pins the archive selection: real wizdom
// archives hold several files, including non-subtitle extras.
func TestExtractSubtitlePrefersSRT(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"readme.nfo": "not a subtitle",
		"movie.srt":  "1\n00:00:01,000 --> 00:00:02,000\nhello\n",
	} {
		f, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		fmt.Fprint(f, body)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	got, err := extractSubtitle(buf.Bytes())
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !strings.Contains(string(got), "-->") {
		t.Errorf("extracted the wrong archive entry: %q", got)
	}
}

// TestSelectBestIgnoresDirectoryTokens pins that ranking looks at the file
// name alone. A media root like "/movies/1080p/" would otherwise donate
// "1080p" to every candidate and outweigh the release name that actually
// determines whether the timings line up.
func TestSelectBestIgnoresDirectoryTokens(t *testing.T) {
	results := []result{
		{ID: 1, VersionName: "Movie.2010.1080p.BluRay.x264-NOMATCH"},
		{ID: 2, VersionName: "Movie.2010.DVDRip.XviD-RIGHT"},
	}
	// The file is the DVDRip; only the *directory* says 1080p.
	got := selectBest(results, "/media/1080p/BluRay/x264/Movie.2010.DVDRip.XviD-RIGHT.mkv")
	if got == nil || got.ID != 2 {
		t.Fatalf("selectBest picked %v; directory tokens outweighed the file name", got)
	}
}
