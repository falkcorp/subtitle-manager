// file: pkg/providers/napiprojekt/napiprojekt.go
// version: 2.0.0
// guid: 4b8e1d70-9a25-4c63-8f01-5d2e7a6c9b34
// last-edited: 2026-07-23

// Package napiprojekt implements a real, keyless subtitle provider for the
// Polish napiprojekt.pl service. It identifies a file by a napiprojekt-specific
// hash (MD5 of the first 10 MiB plus a fixed digest transform) and downloads
// the matching subtitle — the same anonymous, hash-based protocol subliminal
// and Bazarr use. No account or API key is required.
package napiprojekt

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// hashChunk is the number of leading bytes of the media file hashed to identify
// it to napiprojekt (10 MiB, matching the reference implementations).
const hashChunk = 10 * 1024 * 1024

// Client implements the providers.Provider interface for napiprojekt.pl.
type Client struct {
	// APIURL is the base URL of the napiprojekt service (overridable for tests).
	APIURL string
	// HTTPClient is used to make requests.
	HTTPClient *http.Client
}

// New returns a Client configured with reasonable defaults.
func New() *Client {
	return &Client{
		APIURL:     "http://napiprojekt.pl",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Fetch downloads the subtitle matching mediaPath from napiprojekt. lang selects
// the napiprojekt language code (defaults to Polish, the service's primary
// language). It returns the subtitle bytes or an error when no match exists.
func (c *Client) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	md5hex, err := fileMD5(mediaPath)
	if err != nil {
		return nil, err
	}
	sub := subHash(md5hex)

	l := strings.ToUpper(strings.TrimSpace(lang))
	if l == "" {
		l = "PL"
	}

	q := url.Values{
		"l":       {l},
		"f":       {md5hex},
		"t":       {sub},
		"v":       {"dreambox"},
		"kolejka": {"false"},
		"nick":    {""},
		"pass":    {""},
		"napios":  {"posix"},
	}
	endpoint := strings.TrimRight(c.APIURL, "/") + "/unit_napisy/dl.php?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("napiprojekt returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// napiprojekt signals "no subtitle for this hash" with a short "NPc" body.
	if len(data) == 0 || strings.HasPrefix(strings.TrimSpace(string(data)), "NPc") {
		return nil, fmt.Errorf("no napiprojekt subtitle for %s", mediaPath)
	}
	return data, nil
}

// fileMD5 returns the hex MD5 of the first hashChunk bytes of path (or the whole
// file if it is smaller), which is how napiprojekt identifies media.
func fileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, io.LimitReader(f, hashChunk)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// subHash derives napiprojekt's 5-character sub-hash from the media MD5 hex
// digest, using the service's fixed index/multiplier/addend transform. This is
// the standard algorithm used by subliminal and Bazarr.
func subHash(md5hex string) string {
	idx := []int{0xe, 0x3, 0x6, 0x8, 0x2}
	mul := []int{2, 2, 5, 4, 3}
	add := []int{0, 0xd, 0x10, 0xb, 0x5}

	var b strings.Builder
	for i := 0; i < 5; i++ {
		// One hex digit selects the offset of a two-digit hex slice.
		g, err := strconv.ParseInt(string(md5hex[idx[i]]), 16, 0)
		if err != nil {
			return ""
		}
		t := add[i] + int(g)
		if t+2 > len(md5hex) {
			return ""
		}
		v, err := strconv.ParseInt(md5hex[t:t+2], 16, 0)
		if err != nil {
			return ""
		}
		s := strconv.FormatInt(v*int64(mul[i]), 16)
		b.WriteByte(s[len(s)-1])
	}
	return b.String()
}
