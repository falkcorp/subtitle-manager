// file: pkg/proxy/proxy_test.go
// version: 1.0.0
// guid: 9a1c4e73-2d80-4f56-b3e9-0c7f1a6d5824

package proxy

import (
	"net/http"
	"testing"
)

func TestProxyForURL(t *testing.T) {
	// Empty → environment behaviour (non-nil func), no error.
	fn, err := ProxyForURL("")
	if err != nil || fn == nil {
		t.Fatalf("empty proxy: fn==nil is %v, err=%v", fn == nil, err)
	}

	// Valid → routes requests to the configured proxy host.
	fn, err = ProxyForURL("http://proxy.local:3128")
	if err != nil {
		t.Fatalf("valid proxy: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/movie", nil)
	u, err := fn(req)
	if err != nil || u == nil || u.Host != "proxy.local:3128" {
		t.Fatalf("expected proxy host proxy.local:3128, got u=%v err=%v", u, err)
	}

	// Malformed → error.
	if _, err := ProxyForURL("://nope"); err == nil {
		t.Fatal("expected error for malformed proxy URL")
	}
}

func TestConfigureSetsDefaultTransport(t *testing.T) {
	orig := http.DefaultTransport.(*http.Transport).Proxy
	defer func() { http.DefaultTransport.(*http.Transport).Proxy = orig }()

	if err := Configure("http://proxy.local:8080"); err != nil {
		t.Fatalf("configure: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	u, err := http.DefaultTransport.(*http.Transport).Proxy(req)
	if err != nil || u == nil || u.Host != "proxy.local:8080" {
		t.Fatalf("default transport proxy not set: u=%v err=%v", u, err)
	}
}
