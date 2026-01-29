package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ClientConfig
		wantErr     bool
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
			wantErr: true,
		},
		{
			name: "missing email",
			cfg: ClientConfig{
				URL:      "https://example.atlassian.net",
				APIToken: "token123",
			},
			wantErr: true,
		},
		{
			name: "missing api token",
			cfg: ClientConfig{
				URL:   "https://example.atlassian.net",
				Email: "user@example.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.wantURL, client.URL)
				assert.Equal(t, tt.wantBaseURL, client.BaseURL)
				assert.Equal(t, tt.wantURL+"/rest/agile/1.0", client.AgileURL)
				// Auth header should be set
				assert.Contains(t, client.GetAuthHeader(), "Basic ")
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
				assert.NotEmpty(t, r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "application/json", r.Header.Get("Accept"))

				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client, err := New(ClientConfig{
				URL:      server.URL,
				Email:    "user@example.com",
				APIToken: "token",
			})
			require.NoError(t, err)

			body, err := client.get(server.URL + "/test")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.responseBody, string(body))
			}
		})
	}
}

func TestClient_post_withBody(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedBody)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "user@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	requestBody := map[string]interface{}{
		"summary": "Test issue",
		"priority": map[string]string{
			"name": "High",
		},
	}

	_, err = client.post(server.URL+"/test", requestBody)
	require.NoError(t, err)

	assert.Equal(t, "Test issue", receivedBody["summary"])
	priority := receivedBody["priority"].(map[string]interface{})
	assert.Equal(t, "High", priority["name"])
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
			assert.Equal(t, tt.want, got)
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
			assert.Equal(t, tt.want, client.IssueURL(tt.issueKey))
		})
	}
}

func TestClient_VerboseMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "user@example.com",
		APIToken: "token",
		Verbose:  true,
	})
	require.NoError(t, err)

	// Just verify it doesn't panic with verbose mode
	_, err = client.Get(context.Background(), "/test")
	require.NoError(t, err)
}
