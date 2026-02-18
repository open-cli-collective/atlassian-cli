package dashboards

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.DashboardsResponse{
			Total: 1,
			Dashboards: []api.Dashboard{
				{ID: "10001", Name: "Sprint Board", Owner: &api.User{DisplayName: "Alice"}},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(opts, "", 50)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Sprint Board")
}

func TestRunList_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Sprint", r.URL.Query().Get("dashboardName"))
		json.NewEncoder(w).Encode(api.DashboardSearchResponse{
			Total: 1,
			Values: []api.Dashboard{
				{ID: "10002", Name: "Sprint Board"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(opts, "Sprint", 50)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Sprint Board")
}

func TestRunGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/dashboard/10001":
			json.NewEncoder(w).Encode(api.Dashboard{
				ID:   "10001",
				Name: "My Dashboard",
			})
		case "/rest/api/3/dashboard/10001/gadget":
			json.NewEncoder(w).Encode(api.DashboardGadgetsResponse{
				Gadgets: []api.DashboardGadget{
					{ID: 1, Title: "Filter Results"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(opts, "10001")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "My Dashboard")
	assert.Contains(t, stdout.String(), "Filter Results")
}

func TestRunCreate(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		json.NewEncoder(w).Encode(api.Dashboard{ID: "10099", Name: "New Board"})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(opts, "New Board", "Description")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Created")

	var req api.CreateDashboardRequest
	err = json.Unmarshal(capturedBody, &req)
	require.NoError(t, err)
	assert.Equal(t, "New Board", req.Name)
	assert.Equal(t, "Description", req.Description)
}

func TestRunDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/dashboard/10001", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runDelete(opts, "10001")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Deleted")
}

func TestRunGadgetsList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.DashboardGadgetsResponse{
			Gadgets: []api.DashboardGadget{
				{ID: 1, Title: "Filter Results", ModuleID: "com.atlassian.jira.gadgets:filter-results-gadget"},
				{ID: 2, Title: "Pie Chart", ModuleID: "com.atlassian.jira.gadgets:pie-chart-gadget"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGadgetsList(opts, "10001")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Filter Results")
	assert.Contains(t, stdout.String(), "Pie Chart")
}

func TestRunGadgetsRemove(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/dashboard/10001/gadget/42", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGadgetsRemove(opts, "10001", 42)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Removed")
}
