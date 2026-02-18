package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDashboards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/dashboard", r.URL.Path)
		json.NewEncoder(w).Encode(DashboardsResponse{
			Total: 1,
			Dashboards: []Dashboard{
				{ID: "10001", Name: "My Dashboard"},
			},
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	result, err := client.GetDashboards(0, 50)
	require.NoError(t, err)
	require.Len(t, result.Dashboards, 1)
	assert.Equal(t, "My Dashboard", result.Dashboards[0].Name)
}

func TestSearchDashboards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/dashboard/search", r.URL.Path)
		assert.Equal(t, "Sprint", r.URL.Query().Get("dashboardName"))
		json.NewEncoder(w).Encode(DashboardSearchResponse{
			Total: 1,
			Values: []Dashboard{
				{ID: "10002", Name: "Sprint Board"},
			},
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	result, err := client.SearchDashboards("Sprint", 50)
	require.NoError(t, err)
	require.Len(t, result.Values, 1)
	assert.Equal(t, "Sprint Board", result.Values[0].Name)
}

func TestGetDashboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/dashboard/10001", r.URL.Path)
		json.NewEncoder(w).Encode(Dashboard{
			ID:   "10001",
			Name: "My Dashboard",
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	dash, err := client.GetDashboard("10001")
	require.NoError(t, err)
	assert.Equal(t, "My Dashboard", dash.Name)
}

func TestGetDashboard_EmptyID(t *testing.T) {
	_, err := (&Client{}).GetDashboard("")
	assert.Error(t, err)
}

func TestCreateDashboard(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Dashboard{ID: "10099", Name: "New Board"})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	dash, err := client.CreateDashboard(CreateDashboardRequest{
		Name:             "New Board",
		EditPermissions:  []SharePerm{{Type: "global"}},
		SharePermissions: []SharePerm{{Type: "global"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "10099", dash.ID)
	assert.Equal(t, "New Board", dash.Name)

	var req CreateDashboardRequest
	err = json.Unmarshal(capturedBody, &req)
	require.NoError(t, err)
	assert.Equal(t, "New Board", req.Name)
}

func TestDeleteDashboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/dashboard/10001", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	err = client.DeleteDashboard("10001")
	require.NoError(t, err)
}

func TestDeleteDashboard_EmptyID(t *testing.T) {
	assert.Error(t, (&Client{}).DeleteDashboard(""))
}

func TestGetDashboardGadgets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/dashboard/10001/gadget", r.URL.Path)
		json.NewEncoder(w).Encode(DashboardGadgetsResponse{
			Gadgets: []DashboardGadget{
				{ID: 1, Title: "Filter Results"},
				{ID: 2, Title: "Pie Chart"},
			},
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	result, err := client.GetDashboardGadgets("10001")
	require.NoError(t, err)
	require.Len(t, result.Gadgets, 2)
	assert.Equal(t, "Filter Results", result.Gadgets[0].Title)
}

func TestRemoveDashboardGadget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/dashboard/10001/gadget/42", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	err = client.RemoveDashboardGadget("10001", 42)
	require.NoError(t, err)
}
