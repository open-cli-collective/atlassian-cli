// Package version provides build-time version information for Atlassian CLI tools.
//
// The variables in this package are set at build time via ldflags:
//
//	go build -ldflags "-X github.com/open-cli-collective/atlassian-go/version.Version=1.0.0 \
//	                   -X github.com/open-cli-collective/atlassian-go/version.Commit=abc123 \
//	                   -X github.com/open-cli-collective/atlassian-go/version.BuildDate=2024-01-01"
package version

// Build information, set via ldflags
var (
	// Version is the semantic version of the CLI
	Version = "dev"

	// Commit is the git commit hash
	Commit = "none"

	// BuildDate is the date the binary was built
	BuildDate = "unknown"
)

// Info returns the version string
func Info() string {
	return Version
}

// Full returns the full version information including commit and build date
func Full() string {
	return Version + " (" + Commit + ") built " + BuildDate
}
