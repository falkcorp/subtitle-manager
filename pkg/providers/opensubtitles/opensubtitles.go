// file: pkg/providers/opensubtitles/opensubtitles.go
// version: 1.2.0
// guid: 5af27051-d855-4321-9990-c4a59eaabbd5
// last-edited: 2026-08-04

package opensubtitles

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/logging"
)

// LoginResponse represents the response from the OpenSubtitles login API
type LoginResponse struct {
	User struct {
		AllowedTranslations int    `json:"allowed_translations"`
		AllowedDownloads    int    `json:"allowed_downloads"`
		Level               string `json:"level"`
		UserID              int    `json:"user_id"`
		ExtInstalled        bool   `json:"ext_installed"`
		VIP                 bool   `json:"vip"`
	} `json:"user"`
	BaseURL string `json:"base_url"`
	Token   string `json:"token"`
	Status  string `json:"status"`
}

// SearchResult represents a subtitle search result
type SearchResult struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		SubtitleID        string  `json:"subtitle_id"`
		Language          string  `json:"language"`
		DownloadCount     int     `json:"download_count"`
		NewDownloadCount  int     `json:"new_download_count"`
		HearingImpaired   bool    `json:"hearing_impaired"`
		HD                bool    `json:"hd"`
		FPS               float64 `json:"fps"`
		Votes             int     `json:"votes"`
		Ratings           float64 `json:"ratings"`
		FromTrusted       bool    `json:"from_trusted"`
		ForeignPartsOnly  bool    `json:"foreign_parts_only"`
		AutoTranslated    bool    `json:"auto_translated"`
		MachineTranslated bool    `json:"machine_translated"`
		UploadDate        string  `json:"upload_date"`
		Release           string  `json:"release"`
		Comments          string  `json:"comments"`
		LegacySubtitleID  int     `json:"legacy_subtitle_id"`
		Uploader          struct {
			UploaderID int    `json:"uploader_id"`
			Name       string `json:"name"`
			Rank       string `json:"rank"`
		} `json:"uploader"`
		FeatureDetails struct {
			FeatureID   int    `json:"feature_id"`
			FeatureType string `json:"feature_type"`
			Year        int    `json:"year"`
			Title       string `json:"title"`
			MovieName   string `json:"movie_name"`
			ImdbID      int    `json:"imdb_id"`
			TmdbID      int    `json:"tmdb_id"`
		} `json:"feature_details"`
		URL          string `json:"url"`
		RelatedLinks struct {
			Label  string `json:"label"`
			URL    string `json:"url"`
			ImgURL string `json:"img_url"`
		} `json:"related_links"`
		Files []struct {
			FileID   int    `json:"file_id"`
			CDNumber int    `json:"cd_number"`
			FileName string `json:"file_name"`
		} `json:"files"`
	} `json:"attributes"`
}

// SearchResponse represents the response from the search API
type SearchResponse struct {
	TotalPages int            `json:"total_pages"`
	TotalCount int            `json:"total_count"`
	PerPage    int            `json:"per_page"`
	Page       int            `json:"page"`
	Data       []SearchResult `json:"data"`
}

// DownloadResponse represents the download link response
type DownloadResponse struct {
	Link      string `json:"link"`
	FileName  string `json:"file_name"`
	Requests  int    `json:"requests"`
	Remaining int    `json:"remaining"`
	Message   string `json:"message"`
	ResetTime string `json:"reset_time"`
}

// Client implements the providers.Provider interface for OpenSubtitles.
type Client struct {
	// APIURL allows overriding the REST endpoint, mainly for testing.
	APIURL string
	// UserAgent identifies this application to the OpenSubtitles API.
	UserAgent  string
	HTTPClient *http.Client

	// Authentication
	username string
	password string
	token    string
	tokenMu  sync.RWMutex
	tokenExp time.Time
}

// New returns a new Client configured with username/password from viper config.
func New(_ string) *Client {
	apiURL := resolveAPIURL(viper.GetString("opensubtitles.api_url"))
	ua := viper.GetString("opensubtitles.user_agent")
	if ua == "" {
		ua = "subtitle-manager v1.0"
	}

	username := viper.GetString("opensubtitles.username")
	password := viper.GetString("opensubtitles.password")

	return &Client{
		APIURL:     apiURL,
		UserAgent:  ua,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		username:   username,
		password:   password,
	}
}

// DefaultAPIURL is the OpenSubtitles.com REST v1 base this client speaks.
const DefaultAPIURL = "https://api.opensubtitles.com/api/v1"

// legacyHosts are OpenSubtitles endpoints that predate the REST v1 API. This
// client only speaks v1 — POST /login, GET /subtitles?moviehash=..., POST
// /download — so pointing it at one of these produces nothing but failures.
//
// They appear in real configs because they were the correct value for the old
// XML-RPC and legacy REST APIs, and a config carried forward from either keeps
// the stale host. Verified 2026-08-04: rest.opensubtitles.org answers the v1
// search path with 400, while api.opensubtitles.com/api/v1 answers 403 (missing
// key), i.e. the endpoint exists and only wants credentials.
var legacyHosts = map[string]bool{
	"rest.opensubtitles.org": true,
	"api.opensubtitles.org":  true,
	"www.opensubtitles.org":  true,
	"opensubtitles.org":      true,
}

// resolveAPIURL returns the API base to use, correcting a known-legacy host.
//
// An explicitly configured URL is otherwise honoured untouched — overriding it
// is how the tests point the client at httptest, and second-guessing an
// operator's deliberate value would be worse than a clear failure. Only hosts
// that provably cannot serve this client's requests are replaced, and that is
// said out loud rather than done silently.
func resolveAPIURL(configured string) string {
	if configured == "" {
		return DefaultAPIURL
	}
	u, err := url.Parse(configured)
	if err != nil || !legacyHosts[strings.ToLower(u.Hostname())] {
		return configured
	}
	logging.GetLogger("opensubtitles").Warnf(
		"opensubtitles.api_url is %q, which is a legacy endpoint this client cannot use; falling back to %s. Remove the setting or set it to the v1 base to silence this.",
		configured, DefaultAPIURL)
	return DefaultAPIURL
}

// NewWithCredentials builds a client from explicit values.
//
// It exists so a caller that resolves configuration itself does not have to
// mutate global viper state to influence New(). Provider constructors run
// inside the fetch wave's goroutines, one per provider per fetch, so a
// viper.Set there is a data race against viper's non-thread-safe global map —
// and quietly rewrites config for every other component besides.
//
// Empty values fall back to the same defaults New() uses.
func NewWithCredentials(apiURL, userAgent, username, password string) *Client {
	if userAgent == "" {
		userAgent = "subtitle-manager v1.0"
	}
	return &Client{
		APIURL:     resolveAPIURL(apiURL),
		UserAgent:  userAgent,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		username:   username,
		password:   password,
	}
}

// login authenticates with OpenSubtitles and stores the session token
func (c *Client) login(ctx context.Context) error {
	if c.username == "" || c.password == "" {
		return fmt.Errorf("OpenSubtitles username and password not configured")
	}

	loginData := map[string]string{
		"username": c.username,
		"password": c.password,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return fmt.Errorf("failed to marshal login data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.APIURL+"/login", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Api-Key", "") // Empty API key for username/password login

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	c.tokenMu.Lock()
	c.token = loginResp.Token
	c.tokenExp = time.Now().Add(24 * time.Hour) // Tokens typically expire in 24 hours
	c.tokenMu.Unlock()

	return nil
}

// getToken returns a valid authentication token, logging in if necessary
func (c *Client) getToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		token := c.token
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	// Need to login or refresh token
	if c.username == "" || c.password == "" {
		return "", nil
	}
	if err := c.login(ctx); err != nil {
		return "", err
	}

	c.tokenMu.RLock()
	token := c.token
	c.tokenMu.RUnlock()

	return token, nil
}

// Search returns download URLs for matching subtitles without downloading them.
func (c *Client) Search(ctx context.Context, mediaPath, lang string) ([]string, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	hash, size, err := fileHashFunc(mediaPath)
	if err != nil {
		return nil, err
	}

	// Use the new REST API v1 endpoint
	url := fmt.Sprintf("%s/subtitles?moviehash=%x&moviebytesize=%d&languages=%s", c.APIURL, hash, size, lang)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Token might be expired, try to login again
		c.tokenMu.Lock()
		c.token = ""
		c.tokenMu.Unlock()

		token, err = c.getToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("re-authentication failed: %w", err)
		}

		// Retry the request with new token
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err == nil && len(searchResp.Data) > 0 {
		urls := make([]string, 0, len(searchResp.Data))
		for _, result := range searchResp.Data {
			if result.Attributes.SubtitleID != "" {
				downloadURL := fmt.Sprintf("%s/download", c.APIURL)
				urls = append(urls, downloadURL+"?file_id="+result.Attributes.SubtitleID)
			}
		}
		return urls, nil
	}

	var oldFormat []struct {
		SubDownloadLink string `json:"SubDownloadLink"`
	}
	if err := json.Unmarshal(body, &oldFormat); err == nil {
		urls := make([]string, len(oldFormat))
		for i, r := range oldFormat {
			urls[i] = r.SubDownloadLink
		}
		return urls, nil
	}

	return nil, fmt.Errorf("failed to decode search response")
}

// SearchWithResults returns detailed search results for scoring.
func (c *Client) SearchWithResults(ctx context.Context, mediaPath, lang string) ([]SearchResult, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	hash, size, err := fileHashFunc(mediaPath)
	if err != nil {
		return nil, err
	}

	// Use the new REST API v1 endpoint
	url := fmt.Sprintf("%s/subtitles?moviehash=%x&moviebytesize=%d&languages=%s", c.APIURL, hash, size, lang)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Token might be expired, try to login again
		c.tokenMu.Lock()
		c.token = ""
		c.tokenMu.Unlock()

		token, err = c.getToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("re-authentication failed: %w", err)
		}

		// Retry the request with new token
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err == nil {
		return searchResp.Data, nil
	}

	return nil, fmt.Errorf("failed to decode search response")
}

// Fetch downloads the first matching subtitle for mediaPath in lang.
func (c *Client) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	urls, err := c.Search(ctx, mediaPath, lang)
	if err != nil {
		return nil, err
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("no subtitles found")
	}
	return c.download(ctx, urls[0])
}

// downloadURLForResult builds the download URL for a specific search result,
// mirroring the construction used by Search. It returns "" when the result
// carries no subtitle identifier.
func (c *Client) downloadURLForResult(result SearchResult) string {
	if result.Attributes.SubtitleID == "" {
		return ""
	}
	return fmt.Sprintf("%s/download?file_id=%s", c.APIURL, result.Attributes.SubtitleID)
}

// FetchByResult downloads the subtitle for a specific search result rather than
// the first match. This is what lets a caller score candidates and then
// download the selected best one.
func (c *Client) FetchByResult(ctx context.Context, result SearchResult) ([]byte, error) {
	url := c.downloadURLForResult(result)
	if url == "" {
		return nil, fmt.Errorf("search result has no downloadable subtitle id")
	}
	return c.download(ctx, url)
}

// download performs an authenticated GET of a subtitle download URL and returns
// its bytes.
func (c *Client) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("file download failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
