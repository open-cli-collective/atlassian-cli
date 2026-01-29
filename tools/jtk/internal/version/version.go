package version

// Build information, set via ldflags
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// Info returns version information as a string
func Info() string {
	return Version
}

// Full returns full version information
func Full() string {
	return Version + " (" + Commit + ") built " + BuildDate
}
