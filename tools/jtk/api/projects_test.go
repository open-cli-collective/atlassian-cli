package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchProjects(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		response   string
		statusCode int
		wantErr    bool
		wantCount  int
	}{
		{
			name:  "successful search",
			query: "test",
			response: `{
				"maxResults": 50,
				"startAt": 0,
				"total": 2,
				"isLast": true,
				"values": [
					{"id": "10001", "key": "TST", "name": "Test Project", "projectTypeKey": "software"},
					{"id": "10002", "key": "TST2", "name": "Test Project 2", "projectTypeKey": "business"}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:       "empty results",
			query:      "nonexistent",
			response:   `{"maxResults": 50, "startAt": 0, "total": 0, "isLast": true, "values": []}`,
			statusCode: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "server error",
			query:      "test",
			response:   `{"errorMessages":["Internal error"]}`,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/rest/api/3/project/search", r.URL.Path)
				if tt.query != "" {
					assert.Equal(t, tt.query, r.URL.Query().Get("query"))
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client, err := New(ClientConfig{
				URL:      "https://test.atlassian.net",
				Email:    "test@example.com",
				APIToken: "test-token",
			})
			require.NoError(t, err)
			client.BaseURL = server.URL + "/rest/api/3"

			result, err := client.SearchProjects(tt.query, 0, 50)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Values, tt.wantCount)
		})
	}
}

func TestGetProject(t *testing.T) {
	tests := []struct {
		name       string
		keyOrID    string
		response   string
		statusCode int
		wantErr    bool
		wantKey    string
	}{
		{
			name:    "successful get",
			keyOrID: "TST",
			response: `{
				"id": "10001",
				"key": "TST",
				"name": "Test Project",
				"projectTypeKey": "software",
				"lead": {"accountId": "abc123", "displayName": "John Smith"}
			}`,
			statusCode: http.StatusOK,
			wantKey:    "TST",
		},
		{
			name:       "not found",
			keyOrID:    "NOPE",
			response:   `{"errorMessages":["No project could be found with key 'NOPE'"]}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:    "empty key",
			keyOrID: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.keyOrID == "" {
				client, err := New(ClientConfig{
					URL:      "https://test.atlassian.net",
					Email:    "test@example.com",
					APIToken: "test-token",
				})
				require.NoError(t, err)
				_, err = client.GetProject("")
				assert.Error(t, err)
				return
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/rest/api/3/project/"+tt.keyOrID, r.URL.Path)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client, err := New(ClientConfig{
				URL:      "https://test.atlassian.net",
				Email:    "test@example.com",
				APIToken: "test-token",
			})
			require.NoError(t, err)
			client.BaseURL = server.URL + "/rest/api/3"

			project, err := client.GetProject(tt.keyOrID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantKey, project.Key)
			assert.Equal(t, "John Smith", project.Lead.DisplayName)
		})
	}
}

func TestCreateProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/api/3/project", r.URL.Path)

		var req CreateProjectRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "TST", req.Key)
		assert.Equal(t, "Test Project", req.Name)
		assert.Equal(t, "software", req.ProjectTypeKey)
		assert.Equal(t, "abc123", req.LeadAccountID)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ProjectDetail{
			ID:             "10001",
			Key:            "TST",
			Name:           "Test Project",
			ProjectTypeKey: "software",
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	require.NoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	project, err := client.CreateProject(&CreateProjectRequest{
		Key:            "TST",
		Name:           "Test Project",
		ProjectTypeKey: "software",
		LeadAccountID:  "abc123",
	})
	require.NoError(t, err)
	assert.Equal(t, "TST", project.Key)
	assert.Equal(t, "Test Project", project.Name)
}

func TestUpdateProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/rest/api/3/project/TST", r.URL.Path)

		var req UpdateProjectRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", req.Name)

		json.NewEncoder(w).Encode(ProjectDetail{
			ID:   "10001",
			Key:  "TST",
			Name: "Updated Name",
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	require.NoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	project, err := client.UpdateProject("TST", &UpdateProjectRequest{
		Name: "Updated Name",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", project.Name)
}

func TestUpdateProject_EmptyKey(t *testing.T) {
	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	require.NoError(t, err)

	_, err = client.UpdateProject("", &UpdateProjectRequest{Name: "test"})
	assert.Error(t, err)
}

func TestDeleteProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/rest/api/3/project/TST", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	require.NoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	err = client.DeleteProject("TST")
	assert.NoError(t, err)
}

func TestDeleteProject_EmptyKey(t *testing.T) {
	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	require.NoError(t, err)

	err = client.DeleteProject("")
	assert.Error(t, err)
}

func TestRestoreProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/api/3/project/TST/restore", r.URL.Path)
		json.NewEncoder(w).Encode(ProjectDetail{
			ID:   "10001",
			Key:  "TST",
			Name: "Test Project",
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	require.NoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	project, err := client.RestoreProject("TST")
	require.NoError(t, err)
	assert.Equal(t, "TST", project.Key)
}

func TestRestoreProject_EmptyKey(t *testing.T) {
	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	require.NoError(t, err)

	_, err = client.RestoreProject("")
	assert.Error(t, err)
}

func TestListProjectTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/project/type", r.URL.Path)
		json.NewEncoder(w).Encode([]ProjectType{
			{Key: "software", FormattedKey: "Software"},
			{Key: "business", FormattedKey: "Business"},
			{Key: "service_desk", FormattedKey: "Service Desk"},
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	require.NoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	types, err := client.ListProjectTypes()
	require.NoError(t, err)
	assert.Len(t, types, 3)
	assert.Equal(t, "software", types[0].Key)
}
