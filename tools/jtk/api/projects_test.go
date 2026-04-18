package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestSearchProjects(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				testutil.Equal(t, r.URL.Path, "/rest/api/3/project/search")
				if tt.query != "" {
					testutil.Equal(t, r.URL.Query().Get("query"), tt.query)
				}
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client, err := New(ClientConfig{
				URL:      "https://test.atlassian.net",
				Email:    "test@example.com",
				APIToken: "test-token",
			})
			testutil.RequireNoError(t, err)
			client.BaseURL = server.URL + "/rest/api/3"

			result, err := client.SearchProjects(context.Background(), tt.query, 0, 50, "")
			if tt.wantErr {
				testutil.Error(t, err)
				return
			}

			testutil.RequireNoError(t, err)
			testutil.Len(t, result.Values, tt.wantCount)
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
				testutil.RequireNoError(t, err)
				_, err = client.GetProject(context.Background(), "", "")
				testutil.Error(t, err)
				return
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				testutil.Equal(t, r.URL.Path, "/rest/api/3/project/"+tt.keyOrID)
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client, err := New(ClientConfig{
				URL:      "https://test.atlassian.net",
				Email:    "test@example.com",
				APIToken: "test-token",
			})
			testutil.RequireNoError(t, err)
			client.BaseURL = server.URL + "/rest/api/3"

			project, err := client.GetProject(context.Background(), tt.keyOrID, "")
			if tt.wantErr {
				testutil.Error(t, err)
				return
			}

			testutil.RequireNoError(t, err)
			testutil.Equal(t, project.Key, tt.wantKey)
			testutil.Equal(t, project.Lead.DisplayName, "John Smith")
		})
	}
}

func TestSearchProjects_PassesExpandThroughUnmodified(t *testing.T) {
	t.Parallel()
	// The API layer does not know about `--extended`; it takes an expand
	// string verbatim. Two branches: empty means no expand param on the
	// URL; non-empty shows up exactly as passed. Callers map their
	// rendering intent to a concrete expand.
	cases := []struct {
		name             string
		expand           string
		wantParam        string
		wantParamPresent bool
	}{
		{"empty omits param", "", "", false},
		{"lead only", "lead", "lead", true},
		{"full set", ProjectListExpand, ProjectListExpand, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var captured string
			var present bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				captured = r.URL.Query().Get("expand")
				_, present = r.URL.Query()["expand"]
				_, _ = w.Write([]byte(`{"values":[],"isLast":true}`))
			}))
			defer server.Close()

			client, err := New(ClientConfig{URL: "https://test.atlassian.net", Email: "t@t.com", APIToken: "x"})
			testutil.RequireNoError(t, err)
			client.BaseURL = server.URL + "/rest/api/3"

			_, err = client.SearchProjects(context.Background(), "", 0, 50, tc.expand)
			testutil.RequireNoError(t, err)
			testutil.Equal(t, captured, tc.wantParam)
			testutil.Equal(t, present, tc.wantParamPresent)
		})
	}
}

func TestGetProject_PassesExpandThroughUnmodified(t *testing.T) {
	t.Parallel()
	// GetProject takes an expand string verbatim: empty string sends no
	// expand param; non-empty shows up exactly. Callers (e.g. `projects
	// get` default, `projects get --id`, `issues types`) own the decision.
	cases := []struct {
		name             string
		expand           string
		wantParam        string
		wantParamPresent bool
	}{
		{"empty omits param (id path)", "", "", false},
		{"narrow single key (issues types)", "issueTypes", "issueTypes", true},
		{"full get expansion", ProjectGetExpand, ProjectGetExpand, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var captured string
			var present bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				captured = r.URL.Query().Get("expand")
				_, present = r.URL.Query()["expand"]
				_, _ = w.Write([]byte(`{"id":"10001","key":"TST","name":"Test"}`))
			}))
			defer server.Close()

			client, err := New(ClientConfig{URL: "https://test.atlassian.net", Email: "t@t.com", APIToken: "x"})
			testutil.RequireNoError(t, err)
			client.BaseURL = server.URL + "/rest/api/3"

			_, err = client.GetProject(context.Background(), "TST", tc.expand)
			testutil.RequireNoError(t, err)
			testutil.Equal(t, captured, tc.wantParam)
			testutil.Equal(t, present, tc.wantParamPresent)
		})
	}
}

func TestCreateProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/project")

		var req CreateProjectRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.Equal(t, req.Key, "TST")
		testutil.Equal(t, req.Name, "Test Project")
		testutil.Equal(t, req.ProjectTypeKey, "software")
		testutil.Equal(t, req.LeadAccountID, "abc123")

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ProjectDetail{
			ID:             json.Number("10001"),
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
	testutil.RequireNoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	project, err := client.CreateProject(context.Background(), &CreateProjectRequest{
		Key:            "TST",
		Name:           "Test Project",
		ProjectTypeKey: "software",
		LeadAccountID:  "abc123",
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, project.Key, "TST")
	testutil.Equal(t, project.Name, "Test Project")
}

func TestCreateProject_NumericID(t *testing.T) {
	// Jira's create endpoint returns the ID as a number, not a string.
	// This verifies we can parse both shapes.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		// Raw JSON with numeric id (the actual API response shape)
		_, _ = w.Write([]byte(`{"id": 10031, "key": "NEW", "name": "New Project"}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	testutil.RequireNoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	project, err := client.CreateProject(context.Background(), &CreateProjectRequest{
		Key:            "NEW",
		Name:           "New Project",
		ProjectTypeKey: "software",
		LeadAccountID:  "abc123",
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, project.Key, "NEW")
	testutil.Equal(t, project.ID.String(), "10031")
}

func TestUpdateProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPut)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/project/TST")

		var req UpdateProjectRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.Equal(t, req.Name, "Updated Name")

		_ = json.NewEncoder(w).Encode(ProjectDetail{
			ID:   json.Number("10001"),
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
	testutil.RequireNoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	project, err := client.UpdateProject(context.Background(), "TST", &UpdateProjectRequest{
		Name: "Updated Name",
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, project.Name, "Updated Name")
}

func TestUpdateProject_EmptyKey(t *testing.T) {
	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	testutil.RequireNoError(t, err)

	_, err = client.UpdateProject(context.Background(), "", &UpdateProjectRequest{Name: "test"})
	testutil.Error(t, err)
}

func TestDeleteProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodDelete)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/project/TST")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	testutil.RequireNoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	err = client.DeleteProject(context.Background(), "TST")
	testutil.NoError(t, err)
}

func TestDeleteProject_EmptyKey(t *testing.T) {
	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	testutil.RequireNoError(t, err)

	err = client.DeleteProject(context.Background(), "")
	testutil.Error(t, err)
}

func TestRestoreProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/project/TST/restore")
		_ = json.NewEncoder(w).Encode(ProjectDetail{
			ID:   json.Number("10001"),
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
	testutil.RequireNoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	project, err := client.RestoreProject(context.Background(), "TST")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, project.Key, "TST")
}

func TestRestoreProject_EmptyKey(t *testing.T) {
	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	testutil.RequireNoError(t, err)

	_, err = client.RestoreProject(context.Background(), "")
	testutil.Error(t, err)
}

func TestListProjectTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/project/type")
		_ = json.NewEncoder(w).Encode([]ProjectType{
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
	testutil.RequireNoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	types, err := client.ListProjectTypes(context.Background())
	testutil.RequireNoError(t, err)
	testutil.Len(t, types, 3)
	testutil.Equal(t, types[0].Key, "software")
}
