package client

import (
	"io"
	"time"
)

// DefaultTimeout is the default HTTP request timeout.
const DefaultTimeout = 30 * time.Second

// Options configures client behavior.
type Options struct {
	// Timeout for HTTP requests. Defaults to 30 seconds if not set.
	Timeout time.Duration

	// Verbose enables request/response logging.
	Verbose bool

	// VerboseOut is the writer for verbose output. Defaults to os.Stderr.
	VerboseOut io.Writer
}

// timeoutOrDefault returns the configured timeout or the default.
func (o *Options) timeoutOrDefault() time.Duration {
	if o == nil || o.Timeout == 0 {
		return DefaultTimeout
	}
	return o.Timeout
}
