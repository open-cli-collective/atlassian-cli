// Package auth provides authentication utilities for Atlassian APIs.
package auth

import (
	"encoding/base64"
	"errors"
	"fmt"
)

const (
	// AuthMethodBasic is the default authentication method using email:token.
	AuthMethodBasic = "basic"

	// AuthMethodBearer is the authentication method for service accounts with scoped API tokens.
	AuthMethodBearer = "bearer"

	// AuthMethodProxy sends no Authorization header and relies on a local or upstream proxy.
	AuthMethodProxy = "proxy"
)

// ErrInvalidAuthMethod is returned when an unrecognized auth method is provided.
var ErrInvalidAuthMethod = errors.New("invalid auth method: must be \"basic\", \"bearer\", or \"proxy\"")

// ValidateAuthMethod returns nil if method is a recognized auth method, or ErrInvalidAuthMethod otherwise.
func ValidateAuthMethod(method string) error {
	switch method {
	case AuthMethodBasic, AuthMethodBearer, AuthMethodProxy:
		return nil
	default:
		return fmt.Errorf("%w: got %q", ErrInvalidAuthMethod, method)
	}
}

// NormalizeConfig applies auth-method policy to config credential fields.
//
// Empty auth method defaults to basic. Proxy auth sends no CLI-side
// credentials, so direct credential fields are cleared.
func NormalizeConfig(authMethod, email, apiToken, cloudID string) (string, string, string, string) {
	if authMethod == "" {
		authMethod = AuthMethodBasic
	}
	if authMethod == AuthMethodProxy {
		email = ""
		apiToken = ""
		cloudID = ""
	}
	return authMethod, email, apiToken, cloudID
}

// RequireNonInteractiveFields validates the auth fields needed by scripted
// init flows and names the first missing CLI value. toolHint is the
// tool-specific set-credential command shown when the API token is absent.
func RequireNonInteractiveFields(url, authMethod, email, apiToken, cloudID, toolHint string) error {
	if url == "" {
		return errors.New("--non-interactive: missing required value for --url")
	}

	switch authMethod {
	case AuthMethodProxy:
		return nil
	case AuthMethodBearer:
		if cloudID == "" {
			return errors.New("--non-interactive: missing required value for --cloud-id (bearer auth)")
		}
	default:
		if email == "" {
			return errors.New("--non-interactive: missing required value for --email (basic auth)")
		}
	}

	if apiToken == "" {
		return fmt.Errorf("--non-interactive: missing required value for --token-stdin or --token-from-env VAR (or pre-stage with `%s`)", toolHint)
	}
	return nil
}

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
