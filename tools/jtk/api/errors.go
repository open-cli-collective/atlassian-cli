package api

import (
	"errors"

	sharederrors "github.com/open-cli-collective/atlassian-go/errors"
)

// Re-export common errors from shared module
var (
	ErrNotFound     = sharederrors.ErrNotFound
	ErrUnauthorized = sharederrors.ErrUnauthorized
	ErrForbidden    = sharederrors.ErrForbidden
	ErrBadRequest   = sharederrors.ErrBadRequest
	ErrRateLimited  = sharederrors.ErrRateLimited
	ErrServerError  = sharederrors.ErrServerError
)

// Jira-specific validation errors
var (
	ErrIssueKeyRequired   = errors.New("issue key is required")
	ErrProjectKeyRequired = errors.New("project key is required")
)

// APIError is an alias for the shared APIError type
type APIError = sharederrors.APIError

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	return sharederrors.IsNotFound(err)
}

// IsUnauthorized checks if an error is an unauthorized error
func IsUnauthorized(err error) bool {
	return sharederrors.IsUnauthorized(err)
}

// IsForbidden checks if an error is a forbidden error
func IsForbidden(err error) bool {
	return sharederrors.IsForbidden(err)
}
