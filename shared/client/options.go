package client

import (
	"io"
	"os"
	"strings"
	"time"
)

// DefaultTimeout is the default HTTP request timeout.
const DefaultTimeout = 60 * time.Second

// GatewayBaseURL is the Atlassian API gateway base URL used for bearer auth
// with scoped API tokens (service accounts).
const GatewayBaseURL = "https://api.atlassian.com"

// GatewayBaseURLFromEnv returns a gateway base URL using tool-specific
// precedence, then the shared ATLASSIAN_GATEWAY_BASE_URL override, then
// the Atlassian Cloud gateway default.
func GatewayBaseURLFromEnv(primaryEnv string) string {
	for _, name := range []string{primaryEnv, "ATLASSIAN_GATEWAY_BASE_URL"} {
		if name == "" {
			continue
		}
		if v := strings.TrimRight(strings.TrimSpace(os.Getenv(name)), "/"); v != "" {
			return v
		}
	}
	return GatewayBaseURL
}

// Options configures client behavior.
type Options struct {
	// Timeout for HTTP requests. Defaults to 60 seconds if not set.
	Timeout time.Duration

	// Verbose enables request/response logging.
	Verbose bool

	// VerboseOut is the writer for verbose output. Defaults to os.Stderr.
	VerboseOut io.Writer

	// AuthHeader overrides the default Basic auth header when set.
	// Use auth.BearerAuthHeader() for service accounts with scoped tokens.
	// When empty, New() computes BasicAuthHeader(email, apiToken) as before.
	AuthHeader string

	// SkipAuthHeader suppresses the Authorization header entirely.
	// Use this for proxy auth, where a trusted proxy injects credentials.
	SkipAuthHeader bool
}

// timeoutOrDefault returns the configured timeout or the default.
func (o *Options) timeoutOrDefault() time.Duration {
	if o == nil || o.Timeout == 0 {
		return DefaultTimeout
	}
	return o.Timeout
}
