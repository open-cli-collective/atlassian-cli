package dashboards

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.DashboardsResponse{
			Total: 1,
			Dashboards: []api.Dashboard{
				{ID: "10001", Name: "Sprint Board", Owner: &api.User{DisplayName: "Alice"}},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(opts, "", 50)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Sprint Board")
}

func TestRunList_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Query().Get("dashboardName"), "Sprint")
		_ = json.NewEncoder(w).Encode(api.DashboardSearchResponse{
			Total: 1,
			Values: []api.Dashboard{
				{ID: "10002", Name: "Sprint Board"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(opts, "Sprint", 50)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Sprint Board")
}

func TestRunGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/dashboard/10001":
			_ = json.NewEncoder(w).Encode(api.Dashboard{
				ID:   "10001",
				Name: "My Dashboard",
			})
		case "/rest/api/3/dashboard/10001/gadget":
			_ = json.NewEncoder(w).Encode(api.DashboardGadgetsResponse{
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
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(opts, "10001")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "My Dashboard")
	testutil.Contains(t, stdout.String(), "Filter Results")
}

func TestRunCreate(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(api.Dashboard{ID: "10099", Name: "New Board"})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(opts, "New Board", "Description")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Created")

	var req api.CreateDashboardRequest
	err = json.Unmarshal(capturedBody, &req)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, req.Name, "New Board")
	testutil.Equal(t, req.Description, "Description")
}

func TestRunDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/dashboard/10001")
		testutil.Equal(t, r.Method, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runDelete(opts, "10001")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Deleted")
}

func TestRunGadgetsList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.DashboardGadgetsResponse{
			Gadgets: []api.DashboardGadget{
				{ID: 1, Title: "Filter Results", ModuleID: "com.atlassian.jira.gadgets:filter-results-gadget"},
				{ID: 2, Title: "Pie Chart", ModuleID: "com.atlassian.jira.gadgets:pie-chart-gadget"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGadgetsList(opts, "10001")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Filter Results")
	testutil.Contains(t, stdout.String(), "Pie Chart")
}

func TestRunGadgetsRemove(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/dashboard/10001/gadget/42")
		testutil.Equal(t, r.Method, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGadgetsRemove(opts, "10001", 42)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Removed")
}
