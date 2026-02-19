package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestGetDashboards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/dashboard")
		_ = json.NewEncoder(w).Encode(DashboardsResponse{
			Total: 1,
			Dashboards: []Dashboard{
				{ID: "10001", Name: "My Dashboard"},
			},
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	result, err := client.GetDashboards(0, 50)
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Dashboards, 1)
	testutil.Equal(t, result.Dashboards[0].Name, "My Dashboard")
}

func TestSearchDashboards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/dashboard/search")
		testutil.Equal(t, r.URL.Query().Get("dashboardName"), "Sprint")
		_ = json.NewEncoder(w).Encode(DashboardSearchResponse{
			Total: 1,
			Values: []Dashboard{
				{ID: "10002", Name: "Sprint Board"},
			},
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	result, err := client.SearchDashboards("Sprint", 50)
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Values, 1)
	testutil.Equal(t, result.Values[0].Name, "Sprint Board")
}

func TestGetDashboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/dashboard/10001")
		_ = json.NewEncoder(w).Encode(Dashboard{
			ID:   "10001",
			Name: "My Dashboard",
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	dash, err := client.GetDashboard("10001")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, dash.Name, "My Dashboard")
}

func TestGetDashboard_EmptyID(t *testing.T) {
	_, err := (&Client{}).GetDashboard("")
	testutil.Error(t, err)
}

func TestCreateDashboard(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, "POST")
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Dashboard{ID: "10099", Name: "New Board"})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	dash, err := client.CreateDashboard(CreateDashboardRequest{
		Name:             "New Board",
		EditPermissions:  []SharePerm{{Type: "global"}},
		SharePermissions: []SharePerm{{Type: "global"}},
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, dash.ID, "10099")
	testutil.Equal(t, dash.Name, "New Board")

	var req CreateDashboardRequest
	err = json.Unmarshal(capturedBody, &req)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, req.Name, "New Board")
}

func TestDeleteDashboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/dashboard/10001")
		testutil.Equal(t, r.Method, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	err = client.DeleteDashboard("10001")
	testutil.RequireNoError(t, err)
}

func TestDeleteDashboard_EmptyID(t *testing.T) {
	testutil.Error(t, (&Client{}).DeleteDashboard(""))
}

func TestGetDashboardGadgets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/dashboard/10001/gadget")
		_ = json.NewEncoder(w).Encode(DashboardGadgetsResponse{
			Gadgets: []DashboardGadget{
				{ID: 1, Title: "Filter Results"},
				{ID: 2, Title: "Pie Chart"},
			},
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	result, err := client.GetDashboardGadgets("10001")
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Gadgets, 2)
	testutil.Equal(t, result.Gadgets[0].Title, "Filter Results")
}

func TestRemoveDashboardGadget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/dashboard/10001/gadget/42")
		testutil.Equal(t, r.Method, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	err = client.RemoveDashboardGadget("10001", 42)
	testutil.RequireNoError(t, err)
}
