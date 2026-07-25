// file: pkg/radarr/rescan.go
// version: 1.0.0
// guid: 44eff140-954a-442f-b2ec-a5e63dfa6de7
// last-edited: 2026-07-25

package radarr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// MovieRef is the minimum a rescan needs: the Radarr movie id and the movie
// folder, already translated into subtitle-manager's path space.
type MovieRef struct {
	ID int
	// Path is the movie folder as subtitle-manager sees it, i.e. after
	// Filters.MapPath has been applied.
	Path string
}

// movieResponse mirrors the subset of GET /api/v3/movie we care about.
type movieResponse struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

// MovieRefs lists every movie with its folder, mapped into subtitle-manager's
// path space so results can be compared against local media paths directly.
func (c *Client) MovieRefs(ctx context.Context) ([]MovieRef, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v3/movie", nil)
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
	var raw []movieResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	refs := make([]MovieRef, 0, len(raw))
	for _, m := range raw {
		if m.Path == "" || m.ID == 0 {
			continue
		}
		refs = append(refs, MovieRef{ID: m.ID, Path: c.Filters.MapPath(m.Path)})
	}
	return refs, nil
}

// Rescan asks Radarr to rescan the given movie's folder from disk, which is how
// a newly written subtitle becomes visible to Radarr. Mirrors Bazarr's
// notify_radarr (bazarr/radarr/notify.py).
func (c *Client) Rescan(ctx context.Context, movieID int) error {
	body, err := json.Marshal(map[string]any{"name": "RescanMovie", "movieId": movieID})
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
	// Radarr answers 201 Created for an accepted command; accept any 2xx.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

// MatchPath returns the id of the movie whose folder contains videoPath. See
// sonarr.MatchPath for why matching happens in subtitle-manager's path space
// and why ambiguous matches report none rather than guessing.
func MatchPath(refs []MovieRef, videoPath string) (int, bool) {
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
// "/movies/Alien" from swallowing "/movies/Aliens/Aliens.mkv".
func underFolder(file, dir string) bool {
	if dir == "" {
		return false
	}
	trimmed := strings.TrimSuffix(dir, "/")
	return strings.HasPrefix(file, trimmed+"/")
}
