// Package url provides URL normalization utilities for Atlassian CLI tools.
package url

import "strings"

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
