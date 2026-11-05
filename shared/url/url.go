// Package url provides URL normalization utilities for Atlassian CLI tools.
package url

import (
	"errors"
	"net/netip"
	neturl "net/url"
	"strings"
)

// ErrRequiresHTTPS is returned for a cleartext non-loopback URL.
var ErrRequiresHTTPS = errors.New("url must use https unless it is loopback http")

// RequireSecureOrLoopback permits HTTPS endpoints and local HTTP endpoints.
// Scheme-less URLs are normalized to HTTPS, as they are by API clients.
func RequireSecureOrLoopback(u string) error {
	if strings.Contains(u, "://") && !HasScheme(u) {
		return ErrRequiresHTTPS
	}
	u = NormalizeURL(u)
	parsed, err := neturl.Parse(u)
	if err != nil || parsed.Host == "" {
		return ErrRequiresHTTPS
	}
	if parsed.Scheme == "https" || IsLoopbackHTTP(u) {
		return nil
	}
	return ErrRequiresHTTPS
}

// NormalizeURL ensures the URL has an https scheme and no trailing slashes.
// If the URL is empty, it returns an empty string.
// If the URL has no scheme, https:// is prepended.
// Any trailing slashes are removed.
//
// Examples:
//
//	NormalizeURL("example.atlassian.net") → "https://example.atlassian.net"
//	NormalizeURL("https://example.com/") → "https://example.com"
//	NormalizeURL("http://localhost:8080") → "http://localhost:8080"
func NormalizeURL(u string) string {
	if u == "" {
		return ""
	}

	// Add https:// if no scheme
	if !HasScheme(u) {
		u = "https://" + u
	}

	// Remove trailing slashes
	return TrimTrailingSlashes(u)
}

// HasScheme checks if a URL has an http or https scheme.
func HasScheme(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

// TrimTrailingSlashes removes all trailing slashes from a URL.
func TrimTrailingSlashes(u string) string {
	return strings.TrimRight(u, "/")
}

// IsLoopbackHTTP reports whether u is an http:// URL whose host is localhost
// or a loopback IP address. It is intentionally narrow so CLIs can allow local
// development/proxy endpoints without allowing arbitrary cleartext URLs.
func IsLoopbackHTTP(u string) bool {
	parsed, err := neturl.Parse(u)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" {
		return true
	}
	addr, err := netip.ParseAddr(host)
	return err == nil && addr.IsLoopback()
}
