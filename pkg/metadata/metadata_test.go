package metadata

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/testutil"
)

func TestParseFileName(t *testing.T) {
	m, err := ParseFileName("Show.Name.S01E02-GRP!.mkv")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if m.Type != TypeEpisode || m.Title != "Show Name" || m.Season != 1 || m.Episode != 2 || m.ReleaseGroup != "GRP" {
		t.Fatalf("unexpected result: %+v", m)
	}
	m, err = ParseFileName("Movie Title (2020)-TEAM!.mp4")
	if err != nil {
		t.Fatalf("parse movie: %v", err)
	}
	if m.Type != TypeMovie || m.Title != "Movie Title" || m.Year != 2020 || m.ReleaseGroup != "TEAM" {
		t.Fatalf("unexpected movie: %+v", m)
	}
}

func TestQueryMovie(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search/movie" {
			fmt.Fprint(w, `{"results":[{"id":1,"title":"Test","release_date":"2020-01-01"}]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	SetTMDBAPIBase(srv.URL)

	m, err := QueryMovie(context.Background(), "Test", 0, "key")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if m.TMDBID != 1 || m.Title != "Test" || m.Year != 2020 {
		t.Fatalf("unexpected movie: %+v", m)
	}
}

func TestQueryEpisode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/tv":
			fmt.Fprint(w, `{"results":[{"id":2,"name":"Show"}]}`)
		case "/tv/2/season/1/episode/3":
			fmt.Fprint(w, `{"id":5,"name":"Episode"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	SetTMDBAPIBase(srv.URL)

	m, err := QueryEpisode(context.Background(), "Show", 1, 3, "key")
	if err != nil {
		t.Fatalf("query ep: %v", err)
	}
	if m.TMDBID != 5 || m.Title != "Show" || m.EpisodeTitle != "Episode" || m.Season != 1 || m.Episode != 3 {
		t.Fatalf("unexpected episode: %+v", m)
	}
}

func TestSearchMovies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search/movie" {
			fmt.Fprint(w, `{"results":[{"id":1,"title":"A","release_date":"2020-01-01"},{"id":2,"title":"B","release_date":"2021-01-01"}]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	SetTMDBAPIBase(srv.URL)

	res, err := SearchMovies(context.Background(), "t", 0, 2, "k")
	if err != nil {
		t.Fatalf("search movies: %v", err)
	}
	if len(res) != 2 || res[1].TMDBID != 2 {
		t.Fatalf("unexpected results %+v", res)
	}
}

func TestSearchShows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search/tv" {
			fmt.Fprint(w, `{"results":[{"id":3,"name":"Show"}]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	SetTMDBAPIBase(srv.URL)

	res, err := SearchShows(context.Background(), "show", 5, "k")
	if err != nil {
		t.Fatalf("search shows: %v", err)
	}
	if len(res) != 1 || res[0].TMDBID != 3 {
		t.Fatalf("unexpected results %+v", res)
	}
}

func TestFetchMovieMetadata(t *testing.T) {
	tmdbSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search/movie" {
			fmt.Fprint(w, `{"results":[{"id":1,"title":"Test","release_date":"2020-01-01"}]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer tmdbSrv.Close()
	omdbSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Response":"True","Language":"English,Spanish","imdbRating":"7.5"}`)
	}))
	defer omdbSrv.Close()

	SetTMDBAPIBase(tmdbSrv.URL)
	SetOMDBAPIBase(omdbSrv.URL)

	m, err := FetchMovieMetadata(context.Background(), "Test", 2020, "key", "ok")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if m.Rating != 7.5 || len(m.Languages) != 2 || m.Languages[0] != "English" {
		t.Fatalf("unexpected movie info: %+v", m)
	}
}

func TestFetchEpisodeMetadata(t *testing.T) {
	tmdbSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/tv":
			fmt.Fprint(w, `{"results":[{"id":2,"name":"Show"}]}`)
		case "/tv/2/season/1/episode/3":
			fmt.Fprint(w, `{"id":5,"name":"Episode"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer tmdbSrv.Close()
	omdbSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Response":"True","Language":"English","imdbRating":"8.0"}`)
	}))
	defer omdbSrv.Close()

	SetTMDBAPIBase(tmdbSrv.URL)
	SetOMDBAPIBase(omdbSrv.URL)

	m, err := FetchEpisodeMetadata(context.Background(), "Show", 1, 3, "k", "o")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if m.Rating != 8.0 || len(m.Languages) != 1 || m.EpisodeTitle != "Episode" {
		t.Fatalf("unexpected episode info: %+v", m)
	}
}

func TestFetchMovieMetadataByID(t *testing.T) {
	tmdbSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/movie/5" {
			fmt.Fprint(w, `{"id":5,"title":"ByID","release_date":"2021-05-01"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer tmdbSrv.Close()
	omdbSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Response":"True","Language":"German","imdbRating":"6.5"}`)
	}))
	defer omdbSrv.Close()

	SetTMDBAPIBase(tmdbSrv.URL)
	SetOMDBAPIBase(omdbSrv.URL)

	m, err := FetchMovieMetadataByID(context.Background(), 5, "k", "o")
	if err != nil {
		t.Fatalf("fetch by id: %v", err)
	}
	if m.Title != "ByID" || m.Year != 2021 || m.Rating != 6.5 || len(m.Languages) != 1 {
		t.Fatalf("unexpected movie %+v", m)
	}
}

func TestFetchEpisodeMetadataByID(t *testing.T) {
	tmdbSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tv/3/season/2/episode/4" {
			fmt.Fprint(w, `{"id":7,"name":"ID Episode","show_name":"Show"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer tmdbSrv.Close()
	omdbSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Response":"True","Language":"Italian","imdbRating":"7.0"}`)
	}))
	defer omdbSrv.Close()

	SetTMDBAPIBase(tmdbSrv.URL)
	SetOMDBAPIBase(omdbSrv.URL)

	m, err := FetchEpisodeMetadataByID(context.Background(), 3, 2, 4, "k", "o")
	if err != nil {
		t.Fatalf("fetch by id: %v", err)
	}
	if m.TMDBID != 7 || m.EpisodeTitle != "ID Episode" || m.Rating != 7.0 || len(m.Languages) != 1 {
		t.Fatalf("unexpected ep %+v", m)
	}
}

func TestScanLibraryProgress(t *testing.T) {
	// This test uses SQLite file database, skip if not available
	if err := testutil.CheckSQLiteSupport(); err != nil {
		t.Skipf("SQLite support not available: %v", err)
	}

	dir := t.TempDir()
	// A realistic release name. "movie-GRP.mkv" is not one: ParseFileName
	// rejects it outright with "unrecognised file name", so the scan stored an
	// item with no release group and the assertion below could never pass. The
	// parser is correct; the fixture was not.
	video := filepath.Join(dir, "Movie.2020.1080p.BluRay.x264-GRP.mkv")
	if err := os.WriteFile(video, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	store, err := database.OpenSQLStore(filepath.Join(dir, "db.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	var n int
	cb := func(string) { n++ }
	if err := ScanLibraryProgress(context.Background(), dir, store, cb); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if n != 1 {
		t.Fatalf("callback %d", n)
	}
	items, err := store.ListMediaItems()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].Path != video || items[0].ReleaseGroup != "GRP" {
		t.Fatalf("items %+v", items)
	}
}
