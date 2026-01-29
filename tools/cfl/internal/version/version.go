// Package version provides build-time version information.
package version

// These variables are set at build time via ldflags.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)
