// file: pkg/providers/napiprojekt/napiprojekt_test.go
// version: 2.0.0
// guid: 8c3f0a95-1e47-4d82-b6a0-9f2d5c7e1043

package napiprojekt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writeMedia(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(p, []byte("some media bytes for hashing"), 0o644); err != nil {
		t.Fatalf("write media: %v", err)
	}
	return p
}

// TestFetchHashProtocol verifies Fetch computes a napiprojekt hash, calls the
// dl.php endpoint with the f/t parameters, and returns the subtitle body.
func TestFetchHashProtocol(t *testing.T) {
	var gotPath, gotF, gotT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotF = r.URL.Query().Get("f")
		gotT = r.URL.Query().Get("t")
		_, _ = w.Write([]byte("1\n00:00:01,000 --> 00:00:02,000\nCzesc\n"))
	}))
	defer srv.Close()

	c := New()
	c.APIURL = srv.URL
	data, err := c.Fetch(context.Background(), writeMedia(t), "pl")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if gotPath != "/unit_napisy/dl.php" {
		t.Fatalf("unexpected path %q", gotPath)
	}
	if len(gotF) != 32 {
		t.Fatalf("expected 32-char md5 hash, got %q", gotF)
	}
	if len(gotT) != 5 {
		t.Fatalf("expected 5-char sub-hash, got %q", gotT)
	}
	if string(data) == "" {
		t.Fatal("expected subtitle body")
	}
}

// TestFetchNotFound verifies the "NPc" sentinel is treated as no-match.
func TestFetchNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("NPc0"))
	}))
	defer srv.Close()

	c := New()
	c.APIURL = srv.URL
	if _, err := c.Fetch(context.Background(), writeMedia(t), "pl"); err == nil {
		t.Fatal("expected error for NPc0 not-found sentinel")
	}
}

// TestSubHashDeterministic verifies the sub-hash transform is stable, 5 chars,
// and hex-only for a representative 32-char MD5 digest.
func TestSubHashDeterministic(t *testing.T) {
	const md5hex = "0123456789abcdef0123456789abcdef"
	h1 := subHash(md5hex)
	h2 := subHash(md5hex)
	if h1 != h2 {
		t.Fatalf("sub-hash not deterministic: %q vs %q", h1, h2)
	}
	if len(h1) != 5 {
		t.Fatalf("expected 5-char sub-hash, got %q", h1)
	}
	for _, ch := range h1 {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			t.Fatalf("sub-hash has non-hex char: %q", h1)
		}
	}
}
