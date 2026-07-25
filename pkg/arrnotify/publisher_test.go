// file: pkg/arrnotify/publisher_test.go
// version: 1.0.0
// guid: 65a3269c-5455-48cc-9056-9055485b4c24
// last-edited: 2026-07-25

package arrnotify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/events"
	"github.com/jdfalk/subtitle-manager/pkg/radarr"
	"github.com/jdfalk/subtitle-manager/pkg/sonarr"
)

// arrStub is a fake Sonarr or Radarr exposing the two endpoints the rescan
// path uses, and recording what it was asked to do.
type arrStub struct {
	listPath string
	listBody string

	mu        sync.Mutex
	listCalls int32
	rescanned []int
	listFails bool
}

func (s *arrStub) start(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case s.listPath:
			atomic.AddInt32(&s.listCalls, 1)
			if s.listFails {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = io.WriteString(w, s.listBody)
		case "/api/v3/command":
			var cmd struct {
				SeriesID int `json:"seriesId"`
				MovieID  int `json:"movieId"`
			}
			_ = json.NewDecoder(r.Body).Decode(&cmd)
			id := cmd.SeriesID
			if id == 0 {
				id = cmd.MovieID
			}
			s.mu.Lock()
			s.rescanned = append(s.rescanned, id)
			s.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (s *arrStub) ids() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int(nil), s.rescanned...)
}

func newSonarrStub(t *testing.T, body string) (*arrStub, *sonarr.Client) {
	t.Helper()
	stub := &arrStub{listPath: "/api/v3/series", listBody: body}
	return stub, sonarr.NewClient(stub.start(t).URL, "k")
}

func newRadarrStub(t *testing.T, body string) (*arrStub, *radarr.Client) {
	t.Helper()
	stub := &arrStub{listPath: "/api/v3/movie", listBody: body}
	return stub, radarr.NewClient(stub.start(t).URL, "k")
}

func downloaded(path string) events.SubtitleDownloadedData {
	return events.SubtitleDownloadedData{FilePath: path, Language: "en", Provider: "podnapisi"}
}

// TestDownloadTriggersSonarrRescan is the core behaviour: writing a subtitle for
// an episode asks Sonarr to rescan that series.
func TestDownloadTriggersSonarrRescan(t *testing.T) {
	stub, sc := newSonarrStub(t, `[{"id":7,"path":"/tv/Show"}]`)
	p := NewPublisher(sc, nil)

	p.PublishSubtitleDownloaded(context.Background(), downloaded("/tv/Show/S01E01.mkv"))

	if got := stub.ids(); len(got) != 1 || got[0] != 7 {
		t.Fatalf("rescanned = %v, want [7]", got)
	}
}

// TestDownloadTriggersRadarrRescan covers the movie path, including that Sonarr
// being present but not owning the file falls through to Radarr.
func TestDownloadTriggersRadarrRescan(t *testing.T) {
	sonarrStub, sc := newSonarrStub(t, `[{"id":7,"path":"/tv/Show"}]`)
	radarrStub, rc := newRadarrStub(t, `[{"id":11,"path":"/movies/Film"}]`)
	p := NewPublisher(sc, rc)

	p.PublishSubtitleDownloaded(context.Background(), downloaded("/movies/Film/Film.mkv"))

	if got := radarrStub.ids(); len(got) != 1 || got[0] != 11 {
		t.Fatalf("radarr rescanned = %v, want [11]", got)
	}
	if got := sonarrStub.ids(); len(got) != 0 {
		t.Fatalf("sonarr rescanned = %v, want none", got)
	}
}

// TestUpgradeTriggersRescan: an upgraded subtitle is a new file on disk too.
func TestUpgradeTriggersRescan(t *testing.T) {
	stub, sc := newSonarrStub(t, `[{"id":7,"path":"/tv/Show"}]`)
	p := NewPublisher(sc, nil)

	p.PublishSubtitleUpgraded(context.Background(), events.SubtitleUpgradedData{
		FilePath: "/tv/Show/S01E01.mkv", Language: "en",
	})

	if got := stub.ids(); len(got) != 1 || got[0] != 7 {
		t.Fatalf("rescanned = %v, want [7]", got)
	}
}

// TestFailureEventsDoNotRescan: nothing was written, so nothing to rescan.
func TestFailureEventsDoNotRescan(t *testing.T) {
	stub, sc := newSonarrStub(t, `[{"id":7,"path":"/tv/Show"}]`)
	p := NewPublisher(sc, nil)

	p.PublishSubtitleFailed(context.Background(), events.SubtitleFailedData{FilePath: "/tv/Show/S01E01.mkv"})
	p.PublishSearchFailed(context.Background(), events.SearchFailedData{Query: "Show"})

	if got := stub.ids(); len(got) != 0 {
		t.Fatalf("rescanned = %v, want none", got)
	}
	if n := atomic.LoadInt32(&stub.listCalls); n != 0 {
		t.Fatalf("listed %d times, want 0", n)
	}
}

// TestUnknownPathIsSkipped: a file no *arr owns must not trigger a rescan.
func TestUnknownPathIsSkipped(t *testing.T) {
	stub, sc := newSonarrStub(t, `[{"id":7,"path":"/tv/Show"}]`)
	p := NewPublisher(sc, nil)

	p.PublishSubtitleDownloaded(context.Background(), downloaded("/other/Thing/file.mkv"))

	if got := stub.ids(); len(got) != 0 {
		t.Fatalf("rescanned = %v, want none", got)
	}
}

// TestListingIsCached: a library scan writing many subtitles must not refetch
// the (large) series listing per subtitle.
func TestListingIsCached(t *testing.T) {
	stub, sc := newSonarrStub(t, `[{"id":7,"path":"/tv/Show"}]`)
	p := NewPublisher(sc, nil)

	for i := 0; i < 5; i++ {
		p.PublishSubtitleDownloaded(context.Background(), downloaded("/tv/Show/S01E01.mkv"))
	}

	if n := atomic.LoadInt32(&stub.listCalls); n != 1 {
		t.Fatalf("listed %d times, want 1 (cached)", n)
	}
	if got := stub.ids(); len(got) != 5 {
		t.Fatalf("rescanned %d times, want 5", len(got))
	}
}

// TestCacheExpires: past the TTL the listing is refetched, so a series added
// mid-run is eventually seen.
func TestCacheExpires(t *testing.T) {
	stub, sc := newSonarrStub(t, `[{"id":7,"path":"/tv/Show"}]`)
	p := NewPublisher(sc, nil)
	p.TTL = time.Minute

	now := time.Unix(1000, 0)
	p.now = func() time.Time { return now }

	p.PublishSubtitleDownloaded(context.Background(), downloaded("/tv/Show/S01E01.mkv"))
	now = now.Add(2 * time.Minute)
	p.PublishSubtitleDownloaded(context.Background(), downloaded("/tv/Show/S01E02.mkv"))

	if n := atomic.LoadInt32(&stub.listCalls); n != 2 {
		t.Fatalf("listed %d times, want 2 (cache expired)", n)
	}
}

// TestListingFailureIsFailOpen: an unreachable Sonarr must not panic or block,
// and must not stop Radarr from being consulted. The rescan is simply lost --
// the documented trade-off of resolving ids at event time.
func TestListingFailureIsFailOpen(t *testing.T) {
	sonarrStub := &arrStub{listPath: "/api/v3/series", listFails: true}
	sc := sonarr.NewClient(sonarrStub.start(t).URL, "k")
	radarrStub, rc := newRadarrStub(t, `[{"id":11,"path":"/movies/Film"}]`)
	p := NewPublisher(sc, rc)

	p.PublishSubtitleDownloaded(context.Background(), downloaded("/movies/Film/Film.mkv"))

	if got := radarrStub.ids(); len(got) != 1 || got[0] != 11 {
		t.Fatalf("radarr rescanned = %v, want [11] despite Sonarr being down", got)
	}
}

// TestNilPublisherAndEmptyPathAreSafe covers the degenerate calls the event bus
// can make before configuration is complete.
func TestNilPublisherAndEmptyPathAreSafe(t *testing.T) {
	var p *Publisher
	p.PublishSubtitleDownloaded(context.Background(), downloaded("/tv/Show/S01E01.mkv"))

	stub, sc := newSonarrStub(t, `[{"id":7,"path":"/tv/Show"}]`)
	real := NewPublisher(sc, nil)
	real.PublishSubtitleDownloaded(context.Background(), downloaded(""))
	if n := atomic.LoadInt32(&stub.listCalls); n != 0 {
		t.Fatalf("listed %d times for empty path, want 0", n)
	}
}

// TestConcurrentEventsAreSerialised exercises the mutex under -race and
// confirms the burst still produces exactly one listing fetch.
func TestConcurrentEventsAreSerialised(t *testing.T) {
	stub, sc := newSonarrStub(t, `[{"id":7,"path":"/tv/Show"}]`)
	p := NewPublisher(sc, nil)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.PublishSubtitleDownloaded(context.Background(), downloaded("/tv/Show/S01E01.mkv"))
		}()
	}
	wg.Wait()

	if n := atomic.LoadInt32(&stub.listCalls); n != 1 {
		t.Fatalf("listed %d times, want 1", n)
	}
	if got := stub.ids(); len(got) != 20 {
		t.Fatalf("rescanned %d times, want 20", len(got))
	}
}

// TestPublisherSatisfiesEventPublisher pins the interface contract that makes
// this wireable into events.NewMultiPublisher.
func TestPublisherSatisfiesEventPublisher(t *testing.T) {
	var _ events.EventPublisher = (*Publisher)(nil)
}
