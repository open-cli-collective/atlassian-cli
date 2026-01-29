// Package errors provides error types for Atlassian API responses.
package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Sentinel errors for common HTTP status codes.
var (
	ErrNotFound     = errors.New("resource not found")
	ErrUnauthorized = errors.New("unauthorized: check your credentials")
	ErrForbidden    = errors.New("forbidden: insufficient permissions")
	ErrBadRequest   = errors.New("bad request")
	ErrRateLimited  = errors.New("rate limited: too many requests")
	ErrServerError  = errors.New("server error")
)

// APIError represents an error response from an Atlassian API.
// It supports both Jira and Confluence error response formats.
type APIError struct {
	StatusCode    int               `json:"statusCode"`
	Message       string            `json:"message,omitempty"`
	ErrorMessages []string          `json:"errorMessages,omitempty"`
	Errors        map[string]string `json:"errors,omitempty"`
	ErrorList     []string          `json:"-"` // Confluence uses "errors" as array
}

// UnmarshalJSON handles both Jira and Confluence error formats.
// Jira uses: {"errorMessages": [...], "errors": {"field": "msg"}}
// Confluence uses: {"message": "...", "errors": [...]}
func (e *APIError) UnmarshalJSON(data []byte) error {
	// First, try to unmarshal with standard fields
	type alias APIError
	aux := &struct {
		*alias
		ErrorsRaw json.RawMessage `json:"errors,omitempty"`
	}{
		alias: (*alias)(e),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Handle "errors" field which can be object or array
	if len(aux.ErrorsRaw) > 0 {
		// Try as object (Jira format)
		var errMap map[string]string
		if err := json.Unmarshal(aux.ErrorsRaw, &errMap); err == nil {
			e.Errors = errMap
			return nil
		}

		// Try as array (Confluence format)
		var errList []string
		if err := json.Unmarshal(aux.ErrorsRaw, &errList); err == nil {
			e.ErrorList = errList
			return nil
		}
	}

	return nil
}

// Error returns a human-readable error message.
func (e *APIError) Error() string {
	var parts []string

	// Add main message if present
	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	// Add error messages (Jira format)
	parts = append(parts, e.ErrorMessages...)

	// Add field-specific errors (Jira format)
	for field, msg := range e.Errors {
		parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
	}

	// Add error list (Confluence format)
	parts = append(parts, e.ErrorList...)

	if len(parts) == 0 {
		return fmt.Sprintf("API error (status %d)", e.StatusCode)
	}

	return strings.Join(parts, "; ")
}

// ParseAPIError parses an HTTP response body into an appropriate error type.
// It returns sentinel errors for common status codes, wrapping the APIError
// details when additional information is available.
func ParseAPIError(statusCode int, body []byte) error {
	apiErr := &APIError{StatusCode: statusCode}

	if len(body) > 0 {
		_ = json.Unmarshal(body, apiErr)
	}

	// Determine the base sentinel error
	var sentinel error
	switch statusCode {
	case http.StatusUnauthorized:
		sentinel = ErrUnauthorized
	case http.StatusForbidden:
		sentinel = ErrForbidden
	case http.StatusNotFound:
		sentinel = ErrNotFound
	case http.StatusBadRequest:
		sentinel = ErrBadRequest
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		if statusCode >= 500 {
			sentinel = ErrServerError
		} else {
			// For other 4xx errors, just return the APIError
			return apiErr
		}
	}

	// If we have additional details, wrap the sentinel error
	details := apiErr.Error()
	genericMsg := fmt.Sprintf("API error (status %d)", statusCode)
	if details != genericMsg {
		return fmt.Errorf("%w: %s", sentinel, details)
	}

	return sentinel
}

// IsNotFound returns true if the error is or wraps ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsUnauthorized returns true if the error is or wraps ErrUnauthorized.
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsForbidden returns true if the error is or wraps ErrForbidden.
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsBadRequest returns true if the error is or wraps ErrBadRequest.
func IsBadRequest(err error) bool {
	return errors.Is(err, ErrBadRequest)
}

// IsRateLimited returns true if the error is or wraps ErrRateLimited.
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}

// IsServerError returns true if the error is or wraps ErrServerError.
func IsServerError(err error) bool {
	return errors.Is(err, ErrServerError)
}
