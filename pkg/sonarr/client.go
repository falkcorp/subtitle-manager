package sonarr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jdfalk/subtitle-manager/pkg/arr"
	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
)

// Client provides minimal access to the Sonarr API.
type Client struct {
	BaseURL string
	APIKey  string
	// Filters, when set, restrict which items are ingested (monitored-only,
	// excluded series-types/tags) and remap paths. Sync populates it from config
	// when left zero.
	Filters arr.Filters
	client  *http.Client
}

// NewClient returns a configured Sonarr API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type episode struct {
	Monitored bool `json:"monitored"`
	Series    struct {
		Title      string `json:"title"`
		Monitored  bool   `json:"monitored"`
		SeriesType string `json:"seriesType"`
		Tags       []int  `json:"tags"`
	} `json:"series"`
	EpisodeFile struct {
		Path string `json:"path"`
	} `json:"episodeFile"`
	SeasonNumber  int `json:"seasonNumber"`
	EpisodeNumber int `json:"episodeNumber"`
}

// Episodes retrieves all episodes with associated file paths.
func (c *Client) Episodes(ctx context.Context) ([]database.MediaItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v3/episode?includeEpisodeFile=true", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.APIKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var e []episode
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		return nil, err
	}
	var items []database.MediaItem
	for _, ep := range e {
		if ep.EpisodeFile.Path == "" {
			continue
		}
		if c.Filters.MonitoredOnly && (!ep.Monitored || !ep.Series.Monitored) {
			continue
		}
		if c.Filters.TypeExcluded(ep.Series.SeriesType) || c.Filters.TagsExcluded(ep.Series.Tags) {
			continue
		}
		path := c.Filters.MapPath(ep.EpisodeFile.Path)
		items = append(items, database.MediaItem{Path: path, Title: ep.Series.Title, Season: ep.SeasonNumber, Episode: ep.EpisodeNumber})
	}
	return items, nil
}

// Sync fetches episodes and stores new items in store.
func Sync(ctx context.Context, c *Client, store database.SubtitleStore) error {
	logger := logging.GetLogger("sonarr")
	if c.Filters.IsZero() {
		c.Filters = arr.FiltersFromViper("integrations.sonarr")
	}
	eps, err := c.Episodes(ctx)
	if err != nil {
		return err
	}
	existing, err := store.ListMediaItems()
	if err != nil {
		return err
	}
	existingMap := make(map[string]database.MediaItem, len(existing))
	for _, it := range existing {
		existingMap[it.Path] = it
	}
	for _, it := range eps {
		if rec, ok := existingMap[it.Path]; ok {
			if rec.Title != it.Title || rec.Season != it.Season || rec.Episode != it.Episode {
				logger.Warnf("media conflict for %s: stored %q S%dE%d vs remote %q S%dE%d", it.Path, rec.Title, rec.Season, rec.Episode, it.Title, it.Season, it.Episode)
			}
			continue
		}
		if err := store.InsertMediaItem(&it); err != nil {
			return err
		}
	}
	return nil
}
