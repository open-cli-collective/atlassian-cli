package api //nolint:revive // package name is intentional

import (
	"context"
	"net/http"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestGetFilter(t *testing.T) {
	t.Parallel()
	var receivedPath string

	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"10026","name":"MON Aggregate"}`))
	}))
	defer server.Close()

	f, err := client.GetFilter(context.Background(), "10026")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, f.ID, "10026")
	testutil.Equal(t, f.Name, "MON Aggregate")
	testutil.Equal(t, receivedPath, "/rest/api/3/filter/10026")
}

func TestGetFilter_NotFound(t *testing.T) {
	t.Parallel()
	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errorMessages":["The selected filter is not available to you"]}`))
	}))
	defer server.Close()

	_, err := client.GetFilter(context.Background(), "99999")
	testutil.Error(t, err)
}
