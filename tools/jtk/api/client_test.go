package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cfg         ClientConfig
		wantErr     bool
		wantErrIs   error
		wantURL     string
		wantBaseURL string
	}{
		{
			name: "valid config with full URL",
			cfg: ClientConfig{
				URL:      "https://example.atlassian.net",
				Email:    "user@example.com",
				APIToken: "token123",
			},
			wantErr:     false,
			wantURL:     "https://example.atlassian.net",
			wantBaseURL: "https://example.atlassian.net/rest/api/3",
		},
		{
			name: "valid config with self-hosted URL",
			cfg: ClientConfig{
				URL:      "https://jira.internal.corp.com",
				Email:    "user@example.com",
				APIToken: "token123",
			},
			wantErr:     false,
			wantURL:     "https://jira.internal.corp.com",
			wantBaseURL: "https://jira.internal.corp.com/rest/api/3",
		},
		{
			name: "URL without scheme",
			cfg: ClientConfig{
				URL:      "example.atlassian.net",
				Email:    "user@example.com",
				APIToken: "token123",
			},
			wantErr:     false,
			wantURL:     "https://example.atlassian.net",
			wantBaseURL: "https://example.atlassian.net/rest/api/3",
		},
		{
			name: "URL with trailing slash",
			cfg: ClientConfig{
				URL:      "https://example.atlassian.net/",
				Email:    "user@example.com",
				APIToken: "token123",
			},
			wantErr:     false,
			wantURL:     "https://example.atlassian.net",
			wantBaseURL: "https://example.atlassian.net/rest/api/3",
		},
		{
			name: "missing URL",
			cfg: ClientConfig{
				Email:    "user@example.com",
				APIToken: "token123",
			},
			wantErr:   true,
			wantErrIs: ErrURLRequired,
		},
		{
			name: "missing email",
			cfg: ClientConfig{
				URL:      "https://example.atlassian.net",
				APIToken: "token123",
			},
			wantErr:   true,
			wantErrIs: ErrEmailRequired,
		},
		{
			name: "missing api token",
			cfg: ClientConfig{
				URL:   "https://example.atlassian.net",
				Email: "user@example.com",
			},
			wantErr:   true,
			wantErrIs: ErrAPITokenRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.cfg)
			if tt.wantErr {
				testutil.Error(t, err)
				testutil.Nil(t, client)
				if tt.wantErrIs != nil {
					if !errors.Is(err, tt.wantErrIs) {
						t.Errorf("got error %v, want %v", err, tt.wantErrIs)
					}
				}
			} else {
				testutil.RequireNoError(t, err)
				testutil.NotNil(t, client)
				testutil.Equal(t, client.URL, tt.wantURL)
				testutil.Equal(t, client.BaseURL, tt.wantBaseURL)
				testutil.Equal(t, client.AgileURL, tt.wantURL+"/rest/agile/1.0")
				// Auth header should be set
				testutil.Contains(t, client.GetAuthHeader(), "Basic ")
			}
		})
	}
}

func TestClient_get(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		wantErr        bool
	}{
		{
			name:           "successful GET",
			responseStatus: http.StatusOK,
			responseBody:   `{"key": "value"}`,
			wantErr:        false,
		},
		{
			name:           "unauthorized",
			responseStatus: http.StatusUnauthorized,
			responseBody:   `{"errorMessages": ["Unauthorized"]}`,
			wantErr:        true,
		},
		{
			name:           "not found",
			responseStatus: http.StatusNotFound,
			responseBody:   `{"errorMessages": ["Issue not found"]}`,
			wantErr:        true,
		},
		{
			name:           "server error",
			responseStatus: http.StatusInternalServerError,
			responseBody:   `{"errorMessages": ["Internal error"]}`,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify auth header is present
				testutil.NotEmpty(t, r.Header.Get("Authorization"))
				testutil.Equal(t, r.Header.Get("Content-Type"), "application/json")
				testutil.Equal(t, r.Header.Get("Accept"), "application/json")

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client, err := New(ClientConfig{
				URL:      server.URL,
				Email:    "user@example.com",
				APIToken: "token",
			})
			testutil.RequireNoError(t, err)

			body, err := client.Get(context.Background(), server.URL+"/test")

			if tt.wantErr {
				testutil.Error(t, err)
			} else {
				testutil.RequireNoError(t, err)
				testutil.Equal(t, string(body), tt.responseBody)
			}
		})
	}
}

func TestClient_post_withBody(t *testing.T) {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedBody)
		testutil.RequireNoError(t, err)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "user@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	requestBody := map[string]any{
		"summary": "Test issue",
		"priority": map[string]string{
			"name": "High",
		},
	}

	_, err = client.Post(context.Background(), server.URL+"/test", requestBody)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, receivedBody["summary"], "Test issue")
	priority := receivedBody["priority"].(map[string]any)
	testutil.Equal(t, priority["name"], "High")
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name   string
		base   string
		params map[string]string
		want   string
	}{
		{
			name:   "no params",
			base:   "https://example.com/api",
			params: nil,
			want:   "https://example.com/api",
		},
		{
			name:   "empty params",
			base:   "https://example.com/api",
			params: map[string]string{},
			want:   "https://example.com/api",
		},
		{
			name: "single param",
			base: "https://example.com/api",
			params: map[string]string{
				"jql": "project = TEST",
			},
			want: "https://example.com/api?jql=project+%3D+TEST",
		},
		{
			name: "multiple params",
			base: "https://example.com/api",
			params: map[string]string{
				"startAt":    "0",
				"maxResults": "50",
			},
			want: "https://example.com/api?maxResults=50&startAt=0",
		},
		{
			name: "skip empty values",
			base: "https://example.com/api",
			params: map[string]string{
				"jql":    "project = TEST",
				"fields": "",
			},
			want: "https://example.com/api?jql=project+%3D+TEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildURL(tt.base, tt.params)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func TestClient_IssueURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		issueKey string
		want     string
	}{
		{
			name:     "cloud URL",
			url:      "https://mycompany.atlassian.net",
			issueKey: "PROJ-123",
			want:     "https://mycompany.atlassian.net/browse/PROJ-123",
		},
		{
			name:     "self-hosted URL",
			url:      "https://jira.internal.corp.com",
			issueKey: "PROJ-456",
			want:     "https://jira.internal.corp.com/browse/PROJ-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{URL: tt.url}
			testutil.Equal(t, client.IssueURL(tt.issueKey), tt.want)
		})
	}
}

func TestClient_VerboseMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "user@example.com",
		APIToken: "token",
		Verbose:  true,
	})
	testutil.RequireNoError(t, err)

	// Just verify it doesn't panic with verbose mode
	_, err = client.Get(context.Background(), "/test")
	testutil.RequireNoError(t, err)
}
