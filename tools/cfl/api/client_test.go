package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestNewClient(t *testing.T) {
	t.Parallel()
	client := NewClient("https://example.atlassian.net/wiki", "user@example.com", "token123")

	testutil.NotNil(t, client)
	testutil.Equal(t, "https://example.atlassian.net/wiki", client.GetBaseURL())
	testutil.Contains(t, client.GetAuthHeader(), "Basic ")
}

func TestClient_AuthHeader(t *testing.T) {
	t.Parallel()
	var capturedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "mytoken")
	_, err := client.Get(context.Background(), "/test")
	testutil.RequireNoError(t, err)

	// Verify Basic auth header
	testutil.True(t, strings.HasPrefix(capturedAuth, "Basic "))
	encoded := strings.TrimPrefix(capturedAuth, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "user@example.com:mytoken", string(decoded))
}

func TestClient_Headers(t *testing.T) {
	t.Parallel()
	var capturedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "mytoken")
	_, err := client.Get(context.Background(), "/test")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, "application/json", capturedHeaders.Get("Accept"))
	testutil.Equal(t, "application/json", capturedHeaders.Get("Content-Type"))
}

func TestClient_ErrorResponse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{
			name:           "401 unauthorized",
			statusCode:     401,
			responseBody:   `{"message": "Authentication failed"}`,
			expectedErrMsg: "Authentication failed",
		},
		{
			name:           "403 forbidden",
			statusCode:     403,
			responseBody:   `{"message": "Access denied"}`,
			expectedErrMsg: "Access denied",
		},
		{
			name:           "404 not found",
			statusCode:     404,
			responseBody:   `{"message": "Page not found"}`,
			expectedErrMsg: "Page not found",
		},
		{
			name:           "500 server error",
			statusCode:     500,
			responseBody:   `{"message": "Internal server error"}`,
			expectedErrMsg: "Internal server error",
		},
		{
			name:           "error with errors array",
			statusCode:     400,
			responseBody:   `{"message": "Bad request", "errors": ["Invalid title", "Missing body"]}`,
			expectedErrMsg: "Invalid title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, "user@example.com", "token")
			_, err := client.Get(context.Background(), "/test")

			testutil.RequireError(t, err)
			testutil.Contains(t, err.Error(), tt.expectedErrMsg)
		})
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		// Slow response
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Get(ctx, "/test")
	testutil.RequireError(t, err)
}

func TestClient_URLConstruction(t *testing.T) {
	t.Parallel()
	var capturedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")

	tests := []struct {
		inputPath    string
		expectedPath string
	}{
		{"/api/v2/spaces", "/api/v2/spaces"},
		{"api/v2/spaces", "/api/v2/spaces"},
	}

	for _, tt := range tests {
		_, err := client.Get(context.Background(), tt.inputPath)
		testutil.RequireNoError(t, err)
		testutil.Equal(t, tt.expectedPath, capturedPath)
	}
}
