package radarr

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

// Client provides minimal access to the Radarr API.
type Client struct {
	BaseURL string
	APIKey  string
	// Filters restricts which movies are ingested and remaps paths; Sync
	// populates it from config when left zero.
	Filters arr.Filters
	client  *http.Client
}

// NewClient returns a configured Radarr API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type movie struct {
	Title     string `json:"title"`
	Monitored bool   `json:"monitored"`
	Tags      []int  `json:"tags"`
	MovieFile struct {
		Path string `json:"path"`
	} `json:"movieFile"`
}

// Movies retrieves all movies with associated file paths.
func (c *Client) Movies(ctx context.Context) ([]database.MediaItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v3/movie?includeMovieFile=true", nil)
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
	var m []movie
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	var items []database.MediaItem
	for _, mv := range m {
		if mv.MovieFile.Path == "" {
			continue
		}
		if c.Filters.MonitoredOnly && !mv.Monitored {
			continue
		}
		if c.Filters.TagsExcluded(mv.Tags) {
			continue
		}
		items = append(items, database.MediaItem{Path: c.Filters.MapPath(mv.MovieFile.Path), Title: mv.Title})
	}
	return items, nil
}

// Sync fetches the movie library and stores new items in store.
func Sync(ctx context.Context, c *Client, store database.SubtitleStore) error {
	logger := logging.GetLogger("radarr")
	if c.Filters.IsZero() {
		c.Filters = arr.FiltersFromViper("integrations.radarr")
	}
	movies, err := c.Movies(ctx)
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
	for _, it := range movies {
		if rec, ok := existingMap[it.Path]; ok {
			if rec.Title != it.Title {
				logger.Warnf("media conflict for %s: stored %q vs remote %q", it.Path, rec.Title, it.Title)
			}
			continue
		}
		if err := store.InsertMediaItem(&it); err != nil {
			return err
		}
	}
	return nil
}
