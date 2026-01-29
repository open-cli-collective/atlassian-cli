package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestAPIError_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		json           string
		wantMessage    string
		wantErrMsgs    []string
		wantErrMap     map[string]string
		wantErrList    []string
		wantStatusCode int
	}{
		{
			name:        "Jira format with errorMessages",
			json:        `{"statusCode": 400, "errorMessages": ["Issue does not exist", "Another error"], "errors": {"field1": "invalid value"}}`,
			wantErrMsgs: []string{"Issue does not exist", "Another error"},
			wantErrMap:  map[string]string{"field1": "invalid value"},
		},
		{
			name:        "Confluence format with message and errors array",
			json:        `{"statusCode": 404, "message": "Page not found", "errors": ["Error 1", "Error 2"]}`,
			wantMessage: "Page not found",
			wantErrList: []string{"Error 1", "Error 2"},
		},
		{
			name:        "Simple message only",
			json:        `{"statusCode": 500, "message": "Internal server error"}`,
			wantMessage: "Internal server error",
		},
		{
			name: "Empty errors",
			json: `{"statusCode": 401}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var apiErr APIError
			err := json.Unmarshal([]byte(tt.json), &apiErr)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if apiErr.Message != tt.wantMessage {
				t.Errorf("Message = %v, want %v", apiErr.Message, tt.wantMessage)
			}

			if len(apiErr.ErrorMessages) != len(tt.wantErrMsgs) {
				t.Errorf("ErrorMessages length = %d, want %d", len(apiErr.ErrorMessages), len(tt.wantErrMsgs))
			}

			if len(apiErr.Errors) != len(tt.wantErrMap) {
				t.Errorf("Errors length = %d, want %d", len(apiErr.Errors), len(tt.wantErrMap))
			}

			if len(apiErr.ErrorList) != len(tt.wantErrList) {
				t.Errorf("ErrorList length = %d, want %d", len(apiErr.ErrorList), len(tt.wantErrList))
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		apiErr   APIError
		contains []string
	}{
		{
			name: "message only",
			apiErr: APIError{
				StatusCode: 400,
				Message:    "Bad request",
			},
			contains: []string{"Bad request"},
		},
		{
			name: "error messages",
			apiErr: APIError{
				StatusCode:    400,
				ErrorMessages: []string{"Error 1", "Error 2"},
			},
			contains: []string{"Error 1", "Error 2"},
		},
		{
			name: "field errors",
			apiErr: APIError{
				StatusCode: 400,
				Errors:     map[string]string{"name": "is required"},
			},
			contains: []string{"name:", "is required"},
		},
		{
			name: "error list",
			apiErr: APIError{
				StatusCode: 404,
				ErrorList:  []string{"Page not found"},
			},
			contains: []string{"Page not found"},
		},
		{
			name: "empty error",
			apiErr: APIError{
				StatusCode: 500,
			},
			contains: []string{"API error (status 500)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.apiErr.Error()
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("Error() = %q, should contain %q", got, want)
				}
			}
		})
	}
}

func TestParseAPIError(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		body        []byte
		wantErr     error
		wantWrapped bool
	}{
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       nil,
			wantErr:    ErrUnauthorized,
		},
		{
			name:        "401 with details",
			statusCode:  http.StatusUnauthorized,
			body:        []byte(`{"message": "Invalid API token"}`),
			wantErr:     ErrUnauthorized,
			wantWrapped: true,
		},
		{
			name:       "403 forbidden",
			statusCode: http.StatusForbidden,
			body:       nil,
			wantErr:    ErrForbidden,
		},
		{
			name:       "404 not found",
			statusCode: http.StatusNotFound,
			body:       nil,
			wantErr:    ErrNotFound,
		},
		{
			name:       "400 bad request",
			statusCode: http.StatusBadRequest,
			body:       nil,
			wantErr:    ErrBadRequest,
		},
		{
			name:       "429 rate limited",
			statusCode: http.StatusTooManyRequests,
			body:       nil,
			wantErr:    ErrRateLimited,
		},
		{
			name:       "500 server error",
			statusCode: http.StatusInternalServerError,
			body:       nil,
			wantErr:    ErrServerError,
		},
		{
			name:       "502 also server error",
			statusCode: http.StatusBadGateway,
			body:       nil,
			wantErr:    ErrServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseAPIError(tt.statusCode, tt.body)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ParseAPIError() = %v, want %v", err, tt.wantErr)
			}

			if tt.wantWrapped {
				// Should have additional details
				unwrapped := errors.Unwrap(err)
				if unwrapped != nil {
					t.Logf("Unwrapped error: %v", unwrapped)
				}
			}
		})
	}
}

func TestParseAPIError_ReturnsAPIError(t *testing.T) {
	// For 4xx errors without sentinel, should return APIError
	body := []byte(`{"message": "Custom error", "errorMessages": ["Detail 1"]}`)
	err := ParseAPIError(422, body)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("Expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 422 {
		t.Errorf("StatusCode = %d, want 422", apiErr.StatusCode)
	}
}

func TestIsHelpers(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		isFunc    func(error) bool
		wantMatch bool
	}{
		{"IsNotFound with ErrNotFound", ErrNotFound, IsNotFound, true},
		{"IsNotFound with wrapped", fmt.Errorf("wrap: %w", ErrNotFound), IsNotFound, true},
		{"IsNotFound with other", ErrUnauthorized, IsNotFound, false},

		{"IsUnauthorized with ErrUnauthorized", ErrUnauthorized, IsUnauthorized, true},
		{"IsUnauthorized with wrapped", fmt.Errorf("wrap: %w", ErrUnauthorized), IsUnauthorized, true},
		{"IsUnauthorized with other", ErrNotFound, IsUnauthorized, false},

		{"IsForbidden with ErrForbidden", ErrForbidden, IsForbidden, true},
		{"IsForbidden with wrapped", fmt.Errorf("wrap: %w", ErrForbidden), IsForbidden, true},
		{"IsForbidden with other", ErrNotFound, IsForbidden, false},

		{"IsBadRequest with ErrBadRequest", ErrBadRequest, IsBadRequest, true},
		{"IsRateLimited with ErrRateLimited", ErrRateLimited, IsRateLimited, true},
		{"IsServerError with ErrServerError", ErrServerError, IsServerError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.isFunc(tt.err); got != tt.wantMatch {
				t.Errorf("%s = %v, want %v", tt.name, got, tt.wantMatch)
			}
		})
	}
}

func TestParseAPIError_WithComplexJiraResponse(t *testing.T) {
	body := []byte(`{
		"errorMessages": ["Issue Does Not Exist"],
		"errors": {
			"project": "project is required",
			"summary": "summary is required"
		}
	}`)

	err := ParseAPIError(http.StatusBadRequest, body)

	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("Expected ErrBadRequest, got %v", err)
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "Issue Does Not Exist") {
		t.Errorf("Error should contain 'Issue Does Not Exist', got: %s", errStr)
	}
}

func TestParseAPIError_WithComplexConfluenceResponse(t *testing.T) {
	body := []byte(`{
		"statusCode": 404,
		"message": "No content found with id: 12345",
		"errors": ["Content not found", "May have been deleted"]
	}`)

	err := ParseAPIError(http.StatusNotFound, body)

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "No content found") {
		t.Errorf("Error should contain 'No content found', got: %s", errStr)
	}
}
