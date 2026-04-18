package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestListPriorities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/priority")
		_ = json.NewEncoder(w).Encode([]Priority{
			{ID: "1", Name: "Highest"},
			{ID: "2", Name: "High"},
			{ID: "3", Name: "Medium"},
			{ID: "4", Name: "Low"},
			{ID: "5", Name: "Lowest"},
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

	priorities, err := client.ListPriorities(context.Background())
	testutil.RequireNoError(t, err)
	testutil.Len(t, priorities, 5)
	testutil.Equal(t, priorities[0].ID, "1")
	testutil.Equal(t, priorities[0].Name, "Highest")
	testutil.Equal(t, priorities[4].Name, "Lowest")
}
