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

// BearerAuthHeader returns the HTTP Bearer Authentication header value
// for use with Atlassian Cloud APIs via the api.atlassian.com gateway.
//
// Service accounts with scoped API tokens must use Bearer authentication
// instead of Basic authentication.
//
// The returned string is in the format "Bearer <token>"
// and can be used directly as the value for the Authorization header.
func BearerAuthHeader(apiToken string) string {
	return "Bearer " + apiToken
}
