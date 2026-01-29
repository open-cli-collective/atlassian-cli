// Package auth provides authentication utilities for Atlassian APIs.
package auth

import (
	"encoding/base64"
	"fmt"
)

// BasicAuthHeader returns the HTTP Basic Authentication header value
// for use with Atlassian Cloud APIs.
//
// The returned string is in the format "Basic <base64-encoded-credentials>"
// and can be used directly as the value for the Authorization header.
func BasicAuthHeader(email, apiToken string) string {
	creds := fmt.Sprintf("%s:%s", email, apiToken)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
}
