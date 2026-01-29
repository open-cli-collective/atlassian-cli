package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	sharederrors "github.com/open-cli-collective/atlassian-go/errors"
	"github.com/stretchr/testify/assert"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name   string
		apiErr *APIError
		want   string
	}{
		{
			name: "with error messages",
			apiErr: &APIError{
				StatusCode:    400,
				ErrorMessages: []string{"Field is required"},
			},
			want: "Field is required",
		},
		{
			name: "with field errors",
			apiErr: &APIError{
				StatusCode: 400,
				Errors: map[string]string{
					"summary": "Summary is required",
				},
			},
			want: "summary: Summary is required",
		},
		{
			name: "with both",
			apiErr: &APIError{
				StatusCode:    400,
				ErrorMessages: []string{"Bad request"},
				Errors: map[string]string{
					"summary": "Required",
				},
			},
			want: "Bad request; summary: Required",
		},
		{
			name: "empty - just status",
			apiErr: &APIError{
				StatusCode: 500,
			},
			want: "API error (status 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.apiErr.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseAPIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    error
		wantMsg    string
	}{
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{}`,
			wantErr:    sharederrors.ErrUnauthorized,
		},
		{
			name:       "401 with message",
			statusCode: http.StatusUnauthorized,
			body:       `{"errorMessages": ["Bad credentials"]}`,
			wantErr:    sharederrors.ErrUnauthorized,
			wantMsg:    "Bad credentials",
		},
		{
			name:       "403 forbidden",
			statusCode: http.StatusForbidden,
			body:       `{}`,
			wantErr:    sharederrors.ErrForbidden,
		},
		{
			name:       "404 not found",
			statusCode: http.StatusNotFound,
			body:       `{}`,
			wantErr:    sharederrors.ErrNotFound,
		},
		{
			name:       "404 with message",
			statusCode: http.StatusNotFound,
			body:       `{"errorMessages": ["Issue Does Not Exist"]}`,
			wantErr:    sharederrors.ErrNotFound,
			wantMsg:    "Issue Does Not Exist",
		},
		{
			name:       "400 bad request",
			statusCode: http.StatusBadRequest,
			body:       `{}`,
			wantErr:    sharederrors.ErrBadRequest,
		},
		{
			name:       "429 rate limited",
			statusCode: http.StatusTooManyRequests,
			body:       `{}`,
			wantErr:    sharederrors.ErrRateLimited,
		},
		{
			name:       "500 server error",
			statusCode: http.StatusInternalServerError,
			body:       `{}`,
			wantErr:    sharederrors.ErrServerError,
		},
		{
			name:       "502 server error",
			statusCode: http.StatusBadGateway,
			body:       `{}`,
			wantErr:    sharederrors.ErrServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use shared ParseAPIError which takes (statusCode, body)
			err := sharederrors.ParseAPIError(tt.statusCode, []byte(tt.body))
			assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)

			if tt.wantMsg != "" {
				assert.Contains(t, err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestParseAPIError_418_NonStandard(t *testing.T) {
	// Test a non-standard status code that isn't explicitly handled
	body := `{"errorMessages": ["I'm a teapot"]}`

	err := sharederrors.ParseAPIError(418, []byte(body))

	// Should return an APIError, not a sentinel error
	var apiErr *APIError
	assert.True(t, errors.As(err, &apiErr))
	assert.Equal(t, 418, apiErr.StatusCode)
	assert.Contains(t, err.Error(), "I'm a teapot")
}

func TestIsNotFound(t *testing.T) {
	assert.True(t, sharederrors.IsNotFound(sharederrors.ErrNotFound))
	assert.True(t, sharederrors.IsNotFound(fmt.Errorf("wrapped: %w", sharederrors.ErrNotFound)))
	assert.False(t, sharederrors.IsNotFound(sharederrors.ErrUnauthorized))
	assert.False(t, sharederrors.IsNotFound(nil))
}

func TestIsUnauthorized(t *testing.T) {
	assert.True(t, sharederrors.IsUnauthorized(sharederrors.ErrUnauthorized))
	assert.False(t, sharederrors.IsUnauthorized(sharederrors.ErrNotFound))
	assert.False(t, sharederrors.IsUnauthorized(nil))
}

func TestIsForbidden(t *testing.T) {
	assert.True(t, sharederrors.IsForbidden(sharederrors.ErrForbidden))
	assert.False(t, sharederrors.IsForbidden(sharederrors.ErrNotFound))
	assert.False(t, sharederrors.IsForbidden(nil))
}
