package dashboards

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newListTestServer(t *testing.T, dashboards []api.Dashboard, gadgets map[string][]api.DashboardGadget) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/rest/api/3/dashboard" && r.URL.Query().Get("dashboardName") == "" {
			_ = json.NewEncoder(w).Encode(api.DashboardsResponse{
				Total:      len(dashboards),
				Dashboards: dashboards,
			})
			return
		}

		for id, gs := range gadgets {
			if r.URL.Path == "/rest/api/3/dashboard/"+id+"/gadget" {
				_ = json.NewEncoder(w).Encode(api.DashboardGadgetsResponse{Gadgets: gs})
				return
			}
		}

		// Default: empty gadgets for any dashboard gadget request
		if strings.Contains(r.URL.Path, "/gadget") {
			_ = json.NewEncoder(w).Encode(api.DashboardGadgetsResponse{})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestRunList(t *testing.T) {
	dashboards := []api.Dashboard{
		{ID: "10001", Name: "Sprint Board", Owner: &api.User{DisplayName: "Alice"}},
	}
	server := newListTestServer(t, dashboards, nil)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Sprint Board")
}

func TestRunList_ColumnOrder(t *testing.T) {
	dashboards := []api.Dashboard{
		{ID: "10001", Name: "Sprint Board", Owner: &api.User{DisplayName: "Alice"}, IsFavourite: true},
	}
	gadgets := map[string][]api.DashboardGadget{
		"10001": {{ID: 1, Title: "G1"}, {ID: 2, Title: "G2"}},
	}
	server := newListTestServer(t, dashboards, gadgets)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")

	header := lines[0]
	idIdx := strings.Index(header, "ID")
	gadgetsIdx := strings.Index(header, "GADGETS")
	ownerIdx := strings.Index(header, "OWNER")
	favIdx := strings.Index(header, "FAVOURITE")
	nameIdx := strings.Index(header, "NAME")
	testutil.True(t, idIdx < gadgetsIdx, "ID before GADGETS")
	testutil.True(t, gadgetsIdx < ownerIdx, "GADGETS before OWNER")
	testutil.True(t, ownerIdx < favIdx, "OWNER before FAVOURITE")
	testutil.True(t, favIdx < nameIdx, "FAVOURITE before NAME")

	testutil.Contains(t, out, "10001")
	testutil.Contains(t, out, "2")
	testutil.Contains(t, out, "Alice")
	testutil.Contains(t, out, "Sprint Board")
}

func TestRunList_IDOnly(t *testing.T) {
	dashboards := []api.Dashboard{
		{ID: "10001", Name: "Sprint Board"},
		{ID: "10002", Name: "Incidents"},
	}
	server := newListTestServer(t, dashboards, nil)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, stdout.String(), "10001\n10002\n")
}

func TestRunList_IDOnly_Empty(t *testing.T) {
	server := newListTestServer(t, nil, nil)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)
	if stdout.String() != "" {
		t.Errorf("--id with empty results should emit nothing, got %q", stdout.String())
	}
}

func TestRunList_Empty(t *testing.T) {
	server := newListTestServer(t, nil, nil)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No dashboards found")
}

func TestRunList_Extended(t *testing.T) {
	dashboards := []api.Dashboard{
		{
			ID:          "10001",
			Name:        "Sprint Board",
			Owner:       &api.User{DisplayName: "Alice"},
			IsFavourite: true,
			Popularity:  7,
			SharePerm:   []api.SharePerm{{Type: "group", Group: &api.SharePermGroup{Name: "developers"}}},
		},
	}
	gadgets := map[string][]api.DashboardGadget{
		"10001": {{ID: 1}, {ID: 2}, {ID: 3}},
	}
	server := newListTestServer(t, dashboards, gadgets)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "RANK")
	testutil.Contains(t, out, "PERMISSIONS")
	testutil.Contains(t, out, "group:developers")
	testutil.Contains(t, out, "7")
}

func TestRunList_GadgetFetchFails(t *testing.T) {
	dashboards := []api.Dashboard{
		{ID: "10001", Name: "Board"},
	}
	// Server that returns dashboards but 500s on gadget requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/rest/api/3/dashboard" {
			_ = json.NewEncoder(w).Encode(api.DashboardsResponse{Total: 1, Dashboards: dashboards})
			return
		}
		if strings.Contains(r.URL.Path, "/gadget") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "Board")
	// Gadget fetch failed → GADGETS column (index 1) should show "-" (unknown), not "0".
	// Agent-style table uses " | " as column separator.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	cols := strings.Split(lines[1], " | ")
	testutil.True(t, len(cols) >= 2, "expected at least 2 columns in data row")
	testutil.Equal(t, cols[1], "-")
}

func TestRunList_IDTakesPrecedenceOverExtended(t *testing.T) {
	dashboards := []api.Dashboard{
		{ID: "10001", Name: "Board", Popularity: 5},
	}
	server := newListTestServer(t, dashboards, nil)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true, Extended: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "10001\n")
}

func TestRunList_ExtendedJSON(t *testing.T) {
	dashboards := []api.Dashboard{
		{ID: "10001", Name: "Board", Popularity: 5},
	}
	server := newListTestServer(t, dashboards, nil)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)

	// JSON mode returns raw data regardless of --extended
	testutil.Contains(t, stdout.String(), `"id"`)
	testutil.NotContains(t, stdout.String(), "RANK")
}

func TestRunList_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/rest/api/3/dashboard/search" {
			testutil.Equal(t, r.URL.Query().Get("dashboardName"), "Sprint")
			_ = json.NewEncoder(w).Encode(api.DashboardSearchResponse{
				Total: 1,
				Values: []api.Dashboard{
					{ID: "10002", Name: "Sprint Board"},
				},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/gadget") {
			_ = json.NewEncoder(w).Encode(api.DashboardGadgetsResponse{})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "Sprint", 50)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Sprint Board")
}

func TestRunGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
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

	err = runGet(context.Background(), opts, "10001")
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

	out := stdout.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	testutil.Contains(t, lines[0], "ID")
	testutil.Contains(t, lines[0], "NAME")
	testutil.Contains(t, lines[0], "GADGETS")
	testutil.Contains(t, out, "New Board")
	testutil.Contains(t, out, "0")

	var req api.CreateDashboardRequest
	err = json.Unmarshal(capturedBody, &req)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, req.Name, "New Board")
	testutil.Equal(t, req.Description, "Description")
}

func TestRunCreate_IDOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Dashboard{ID: "10099", Name: "New Board"})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runCreate(opts, "New Board", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "10099\n")
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
	testutil.Equal(t, stdout.String(), "Deleted dashboard 10001\n")
}

func TestRunDelete_JSONOutputEmitsText(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runDelete(opts, "10001")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted dashboard 10001\n")
	testutil.Equal(t, stderr.String(), "")
}

func TestRunGadgetsList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.DashboardGadgetsResponse{
			Gadgets: []api.DashboardGadget{
				{ID: 1, Title: "Filter Results", ModuleID: "filter-results-gadget", Position: api.DashboardGadgetPos{Row: 0, Column: 0}},
				{ID: 2, Title: "Pie Chart", ModuleID: "pie-chart-gadget", Position: api.DashboardGadgetPos{Row: 1, Column: 0}},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGadgetsList(context.Background(), opts, "10001")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Filter Results")
	testutil.Contains(t, stdout.String(), "Pie Chart")
}

func TestRunGadgetsList_ColumnOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.DashboardGadgetsResponse{
			Gadgets: []api.DashboardGadget{
				{ID: 1, Title: "Burndown", ModuleID: "sprint-burndown-gadget", Position: api.DashboardGadgetPos{Row: 0, Column: 0}},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGadgetsList(context.Background(), opts, "10001")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")

	header := lines[0]
	idIdx := strings.Index(header, "ID")
	posIdx := strings.Index(header, "POSITION")
	titleIdx := strings.Index(header, "TITLE")
	typeIdx := strings.Index(header, "TYPE")
	testutil.True(t, idIdx < posIdx, "ID before POSITION")
	testutil.True(t, posIdx < titleIdx, "POSITION before TITLE")
	testutil.True(t, titleIdx < typeIdx, "TITLE before TYPE")
	testutil.NotContains(t, header, "MODULE")
	testutil.Contains(t, out, "0,0")
}

func TestRunGadgetsList_IDOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.DashboardGadgetsResponse{
			Gadgets: []api.DashboardGadget{
				{ID: 1, Title: "Gadget A"},
				{ID: 2, Title: "Gadget B"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runGadgetsList(context.Background(), opts, "10001")
	testutil.RequireNoError(t, err)

	testutil.Equal(t, stdout.String(), "1\n2\n")
}

func TestRunGadgetsList_IDOnly_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.DashboardGadgetsResponse{})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runGadgetsList(context.Background(), opts, "10001")
	testutil.RequireNoError(t, err)
	if stdout.String() != "" {
		t.Errorf("--id with empty results should emit nothing, got %q", stdout.String())
	}
}

func TestRunGadgetsList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.DashboardGadgetsResponse{})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGadgetsList(context.Background(), opts, "10001")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No gadgets on dashboard 10001")
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
	testutil.Equal(t, stdout.String(), "Removed gadget 42 from dashboard 10001\n")
}

func TestRunGadgetsRemove_JSONOutputEmitsText(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runGadgetsRemove(opts, "10001", 42)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Removed gadget 42 from dashboard 10001\n")
	testutil.Equal(t, stderr.String(), "")
}

func TestRunGadgetsAdd(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/dashboard/10001/gadget")
		testutil.Equal(t, r.Method, "POST")
		capturedBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(api.DashboardGadget{
			ID:       10124,
			Title:    "Sprint Burndown",
			ModuleID: "sprint-burndown-gadget",
			Position: api.DashboardGadgetPos{Row: 1, Column: 0},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGadgetsAdd(context.Background(), opts, "10001", "sprint-burndown-gadget", "Sprint Burndown", "", "1,0")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	testutil.Contains(t, lines[0], "ID")
	testutil.Contains(t, lines[0], "POSITION")
	testutil.Contains(t, lines[0], "TITLE")
	testutil.Contains(t, lines[0], "TYPE")
	testutil.Contains(t, out, "10124")
	testutil.Contains(t, out, "Sprint Burndown")
	testutil.Contains(t, out, "sprint-burndown-gadget")
	testutil.Contains(t, out, "1,0")

	var req api.AddDashboardGadgetRequest
	err = json.Unmarshal(capturedBody, &req)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, req.ModuleKey, "sprint-burndown-gadget")
	testutil.Equal(t, req.Title, "Sprint Burndown")
	testutil.NotNil(t, req.Position)
	testutil.Equal(t, req.Position.Row, 1)
	testutil.Equal(t, req.Position.Column, 0)
}

func TestRunGadgetsAdd_IDOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.DashboardGadget{
			ID:       10124,
			Title:    "Sprint Burndown",
			ModuleID: "sprint-burndown-gadget",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runGadgetsAdd(context.Background(), opts, "10001", "sprint-burndown-gadget", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "10124\n")
}

func TestRunGadgetsAdd_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.DashboardGadget{
			ID:       10124,
			Title:    "Sprint Burndown",
			ModuleID: "sprint-burndown-gadget",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGadgetsAdd(context.Background(), opts, "10001", "sprint-burndown-gadget", "", "", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), `"id"`)
	testutil.Contains(t, stdout.String(), "10124")
}

func TestRunGadgetsAdd_InvalidPosition(t *testing.T) {
	client, err := api.New(api.ClientConfig{URL: "https://test.atlassian.net", Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	tests := []struct {
		name     string
		position string
		errMsg   string
	}{
		{"no comma", "invalid", "invalid position"},
		{"bad row", "abc,0", "invalid position row"},
		{"bad column", "1,xyz", "invalid position column"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &root.Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
			opts.SetAPIClient(client)
			err := runGadgetsAdd(context.Background(), opts, "10001", "gadget", "", "", tt.position)
			testutil.RequireError(t, err)
			testutil.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestRunGadgetsAdd_NegativePosition(t *testing.T) {
	client, err := api.New(api.ClientConfig{URL: "https://test.atlassian.net", Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGadgetsAdd(context.Background(), opts, "10001", "gadget", "", "", "-1,0")
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "non-negative")
}

func TestRunGadgetsAdd_ZeroPosition(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(api.DashboardGadget{
			ID:       10125,
			Title:    "Gadget",
			ModuleID: "test-gadget",
			Position: api.DashboardGadgetPos{Row: 0, Column: 0},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGadgetsAdd(context.Background(), opts, "10001", "test-gadget", "", "", "0,0")
	testutil.RequireNoError(t, err)

	var req api.AddDashboardGadgetRequest
	err = json.Unmarshal(capturedBody, &req)
	testutil.RequireNoError(t, err)
	testutil.NotNil(t, req.Position)
	testutil.Equal(t, req.Position.Row, 0)
	testutil.Equal(t, req.Position.Column, 0)
}
