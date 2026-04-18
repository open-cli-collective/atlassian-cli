package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestListResolutions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/resolution")
		_ = json.NewEncoder(w).Encode([]Resolution{
			{ID: "1", Name: "Fixed", Description: "Work has been completed"},
			{ID: "2", Name: "Won't Do", Description: "Rejected"},
			{ID: "3", Name: "Duplicate"},
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

	resolutions, err := client.ListResolutions(context.Background())
	testutil.RequireNoError(t, err)
	testutil.Len(t, resolutions, 3)
	testutil.Equal(t, resolutions[0].ID, "1")
	testutil.Equal(t, resolutions[0].Name, "Fixed")
	testutil.Equal(t, resolutions[0].Description, "Work has been completed")
	testutil.Equal(t, resolutions[2].Description, "")
}
