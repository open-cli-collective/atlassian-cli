package client

import (
	"io"
	"time"
)

// DefaultTimeout is the default HTTP request timeout.
const DefaultTimeout = 60 * time.Second

// GatewayBaseURL is the Atlassian API gateway base URL used for bearer auth
// with scoped API tokens (service accounts).
const GatewayBaseURL = "https://api.atlassian.com"

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
}

// timeoutOrDefault returns the configured timeout or the default.
func (o *Options) timeoutOrDefault() time.Duration {
	if o == nil || o.Timeout == 0 {
		return DefaultTimeout
	}
	return o.Timeout
}
