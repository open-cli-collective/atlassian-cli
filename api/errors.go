package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Sentinel errors
var (
	ErrNotFound           = errors.New("resource not found")
	ErrUnauthorized       = errors.New("unauthorized: check your credentials")
	ErrForbidden          = errors.New("forbidden: insufficient permissions")
	ErrBadRequest         = errors.New("bad request")
	ErrRateLimited        = errors.New("rate limited: too many requests")
	ErrServerError        = errors.New("server error")
	ErrURLRequired        = errors.New("URL is required")
	ErrEmailRequired      = errors.New("email is required")
	ErrAPITokenRequired   = errors.New("API token is required")
	ErrIssueKeyRequired   = errors.New("issue key is required")
	ErrProjectKeyRequired = errors.New("project key is required")
)

// APIError represents an error response from the Jira API
type APIError struct {
	StatusCode    int
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
}

func (e *APIError) Error() string {
	var parts []string

	if len(e.ErrorMessages) > 0 {
		parts = append(parts, e.ErrorMessages...)
	}

	for field, msg := range e.Errors {
		parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("API error (status %d)", e.StatusCode)
	}

	return strings.Join(parts, "; ")
}

// ParseAPIError parses an error response from the Jira API
func ParseAPIError(resp *http.Response, body []byte) error {
	apiErr := &APIError{StatusCode: resp.StatusCode}

	if len(body) > 0 {
		_ = json.Unmarshal(body, apiErr)
	}

	// Return sentinel errors for common status codes
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		if apiErr.Error() != fmt.Sprintf("API error (status %d)", resp.StatusCode) {
			return fmt.Errorf("%w: %s", ErrUnauthorized, apiErr.Error())
		}
		return ErrUnauthorized
	case http.StatusForbidden:
		if apiErr.Error() != fmt.Sprintf("API error (status %d)", resp.StatusCode) {
			return fmt.Errorf("%w: %s", ErrForbidden, apiErr.Error())
		}
		return ErrForbidden
	case http.StatusNotFound:
		if apiErr.Error() != fmt.Sprintf("API error (status %d)", resp.StatusCode) {
			return fmt.Errorf("%w: %s", ErrNotFound, apiErr.Error())
		}
		return ErrNotFound
	case http.StatusBadRequest:
		if apiErr.Error() != fmt.Sprintf("API error (status %d)", resp.StatusCode) {
			return fmt.Errorf("%w: %s", ErrBadRequest, apiErr.Error())
		}
		return ErrBadRequest
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		if resp.StatusCode >= 500 {
			return fmt.Errorf("%w: %s", ErrServerError, apiErr.Error())
		}
		return apiErr
	}
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsUnauthorized checks if an error is an unauthorized error
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsForbidden checks if an error is a forbidden error
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}
