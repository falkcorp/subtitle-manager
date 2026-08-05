// file: pkg/providers/opensubtitlescom/opensubtitlescom.go
// version: 2.0.0
// guid: 7d3a0c58-2e91-4b46-a5f0-91c48e7d2b03
// last-edited: 2026-08-04

// Package opensubtitlescom exposes the OpenSubtitles.com REST v1 API under the
// name operators and Bazarr configs use for it.
//
// # Why this is a thin wrapper
//
// It used to be an independent stub: it pointed at "api.opensubtitlescom.com" —
// which is not a real host, the API lives at api.opensubtitles.com — and issued
// GET /subtitles/{name}/{lang}, an endpoint that exists in no version of the
// API. It could never have returned a subtitle.
//
// Meanwhile pkg/providers/opensubtitles is already a complete v1 client:
// POST /login, moviehash search against GET /subtitles, and POST /download. So
// the fix is not to write a second implementation but to stop pretending there
// are two providers. Registering both names was actively harmful — the registry
// tried the stub on every fetch wave, where it took one of the concurrent slots
// and always failed.
package opensubtitlescom

import (
	"context"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/providers/opensubtitles"
)

// Client adapts the OpenSubtitles v1 client to the provider name
// "opensubtitlescom".
type Client struct {
	inner *opensubtitles.Client
}

// New returns a client backed by the real OpenSubtitles.com v1 implementation.
//
// Credentials normally live under opensubtitles.*. A config may instead carry
// them under providers.opensubtitlescom.* — the spelling a Bazarr import
// produces for this provider name — so both are read, with opensubtitles.*
// winning. Adding this therefore cannot change a setup that already works.
//
// The fallback is read inline rather than written back to viper. Provider
// constructors run inside the fetch wave's goroutines, so mutating global
// config here would be a data race against viper's non-thread-safe map, and
// would silently rewrite configuration for every other component too.
func New() *Client {
	return &Client{inner: opensubtitles.NewWithCredentials(
		pick("api_url"),
		pick("user_agent"),
		pick("username"),
		pick("password"),
	)}
}

// pick returns opensubtitles.<key>, falling back to
// providers.opensubtitlescom.<key>.
func pick(key string) string {
	if v := viper.GetString("opensubtitles." + key); v != "" {
		return v
	}
	return viper.GetString("providers.opensubtitlescom." + key)
}

// Fetch returns subtitle bytes for mediaPath in lang.
func (c *Client) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	return c.inner.Fetch(ctx, mediaPath, lang)
}

// Search returns candidate subtitle URLs, keeping the Searcher capability the
// underlying client offers.
func (c *Client) Search(ctx context.Context, mediaPath, lang string) ([]string, error) {
	return c.inner.Search(ctx, mediaPath, lang)
}

// SearchWithResults exposes scored search, which is what lets the scanner's
// score-gated download path use this provider instead of falling back to a
// plain fetch.
func (c *Client) SearchWithResults(ctx context.Context, mediaPath, lang string) ([]opensubtitles.SearchResult, error) {
	return c.inner.SearchWithResults(ctx, mediaPath, lang)
}

// FetchByResult downloads a specific search result.
func (c *Client) FetchByResult(ctx context.Context, result opensubtitles.SearchResult) ([]byte, error) {
	return c.inner.FetchByResult(ctx, result)
}
