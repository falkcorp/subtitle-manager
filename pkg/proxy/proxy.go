// file: pkg/proxy/proxy.go
// version: 1.0.0
// guid: 5c9e2a01-7f43-4b68-9d2c-6a1f0e3b8547
// last-edited: 2026-07-23

// Package proxy configures an outbound HTTP proxy for the process. Setting a
// proxy routes every HTTP client that uses http.DefaultTransport (which is all
// of subtitle-manager's provider/integration clients, since they leave
// Transport nil) through the configured proxy — matching Bazarr's proxy
// setting.
package proxy

import (
	"fmt"
	"net/http"
	"net/url"
)

// ProxyForURL returns the proxy function to install for the given proxy URL.
// An empty proxyURL yields http.ProxyFromEnvironment (the standard
// HTTP(S)_PROXY/NO_PROXY behaviour). A malformed URL returns an error.
func ProxyForURL(proxyURL string) (func(*http.Request) (*url.URL, error), error) {
	if proxyURL == "" {
		return http.ProxyFromEnvironment, nil
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL %q: %w", proxyURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid proxy URL %q: scheme and host are required", proxyURL)
	}
	return http.ProxyURL(u), nil
}

// Configure installs proxyURL as the proxy on http.DefaultTransport. When
// proxyURL is empty the environment-variable proxy behaviour is (re)applied.
// It is a no-op if the default transport has been replaced with a non-standard
// type.
func Configure(proxyURL string) error {
	fn, err := ProxyForURL(proxyURL)
	if err != nil {
		return err
	}
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		t.Proxy = fn
	}
	return nil
}
