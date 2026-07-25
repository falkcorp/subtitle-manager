// file: pkg/sonarr/rescan.go
// version: 1.0.0
// guid: ef11acdf-7c70-406c-8d0e-e033603dbbb3
// last-edited: 2026-07-25

package sonarr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// SeriesRef is the minimum a rescan needs: the Sonarr series id and the series
// folder, already translated into subtitle-manager's path space.
type SeriesRef struct {
	ID int
	// Path is the series folder as subtitle-manager sees it, i.e. after
	// Filters.MapPath has been applied.
	Path string
}

// seriesResponse mirrors the subset of GET /api/v3/series we care about.
type seriesResponse struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

// Series lists every series with its folder, mapped into subtitle-manager's
// path space so results can be compared against local media paths directly.
func (c *Client) Series(ctx context.Context) ([]SeriesRef, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v3/series", nil)
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
	var raw []seriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	refs := make([]SeriesRef, 0, len(raw))
	for _, s := range raw {
		if s.Path == "" || s.ID == 0 {
			continue
		}
		refs = append(refs, SeriesRef{ID: s.ID, Path: c.Filters.MapPath(s.Path)})
	}
	return refs, nil
}

// Rescan asks Sonarr to rescan the given series' folder from disk, which is how
// a newly written subtitle becomes visible to Sonarr. Mirrors Bazarr's
// notify_sonarr (bazarr/sonarr/notify.py).
func (c *Client) Rescan(ctx context.Context, seriesID int) error {
	body, err := json.Marshal(map[string]any{"name": "RescanSeries", "seriesId": seriesID})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v3/command", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Sonarr answers 201 Created for an accepted command; accept any 2xx.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

// MatchPath returns the id of the series whose folder contains videoPath.
//
// Matching happens in subtitle-manager's path space: SeriesRef.Path has already
// been through Filters.MapPath, so we never have to invert the mapping (which
// is not reliably invertible -- MapPath is longest-prefix-wins, so two distinct
// Sonarr roots can collapse onto one local root).
//
// The longest matching folder wins so nested libraries behave. If two series
// map to the same folder the match is ambiguous and we report none rather than
// guess, since rescanning the wrong series is a silent no-op that is hard to
// debug.
func MatchPath(refs []SeriesRef, videoPath string) (int, bool) {
	bestLen, bestID, ambiguous := -1, 0, false
	for _, r := range refs {
		if !underFolder(videoPath, r.Path) {
			continue
		}
		switch {
		case len(r.Path) > bestLen:
			bestLen, bestID, ambiguous = len(r.Path), r.ID, false
		case len(r.Path) == bestLen && r.ID != bestID:
			ambiguous = true
		}
	}
	if bestLen < 0 || ambiguous {
		return 0, false
	}
	return bestID, true
}

// underFolder reports whether file sits inside dir. The separator check stops
// "/tv/Show" from swallowing "/tv/Show Name/S01E01.mkv".
func underFolder(file, dir string) bool {
	if dir == "" {
		return false
	}
	trimmed := strings.TrimSuffix(dir, "/")
	return strings.HasPrefix(file, trimmed+"/")
}
