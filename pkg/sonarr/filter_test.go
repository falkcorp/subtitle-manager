// file: pkg/sonarr/filter_test.go
// version: 1.0.0
// guid: 3d9c8a71-4e26-4b2f-8a05-7c1e9d3b6f22

package sonarr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jdfalk/subtitle-manager/pkg/arr"
)

// TestEpisodesFiltering verifies monitored-only, series-type and tag exclusion,
// and path mapping are applied when ingesting Sonarr episodes.
func TestEpisodesFiltering(t *testing.T) {
	const body = `[
	  {"monitored":true,"series":{"title":"Keep","monitored":true,"seriesType":"standard","tags":[1]},"episodeFile":{"path":"/tv/Keep/e1.mkv"},"seasonNumber":1,"episodeNumber":1},
	  {"monitored":false,"series":{"title":"Unmon","monitored":true,"seriesType":"standard","tags":[]},"episodeFile":{"path":"/tv/Unmon/e1.mkv"},"seasonNumber":1,"episodeNumber":1},
	  {"monitored":true,"series":{"title":"Anime","monitored":true,"seriesType":"anime","tags":[]},"episodeFile":{"path":"/tv/Anime/e1.mkv"},"seasonNumber":1,"episodeNumber":1},
	  {"monitored":true,"series":{"title":"Tagged","monitored":true,"seriesType":"standard","tags":[9]},"episodeFile":{"path":"/tv/Tagged/e1.mkv"},"seasonNumber":1,"episodeNumber":1}
	]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "k" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := &Client{
		BaseURL: srv.URL,
		APIKey:  "k",
		client:  srv.Client(),
		Filters: arr.Filters{
			MonitoredOnly:       true,
			ExcludedSeriesTypes: []string{"anime"},
			ExcludedTagIDs:      []int{9},
			PathMappings:        [][2]string{{"/tv", "/media/tv"}},
		},
	}

	items, err := c.Episodes(context.Background())
	if err != nil {
		t.Fatalf("episodes: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item after filtering, got %d: %+v", len(items), items)
	}
	if items[0].Title != "Keep" {
		t.Fatalf("expected Keep, got %q", items[0].Title)
	}
	if items[0].Path != "/media/tv/Keep/e1.mkv" {
		t.Fatalf("expected mapped path, got %q", items[0].Path)
	}
}
