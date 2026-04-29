package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestMoveIssuesToBacklog(t *testing.T) {
	t.Parallel()

	var capturedPath string
	var capturedBody map[string]any

	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := client.MoveIssuesToBacklog(context.Background(), []string{"PROJ-1", "PROJ-2"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, capturedPath, "/rest/agile/1.0/backlog/issue")

	issues, ok := capturedBody["issues"].([]any)
	testutil.True(t, ok)
	testutil.Len(t, issues, 2)
	testutil.Equal(t, issues[0].(string), "PROJ-1")
	testutil.Equal(t, issues[1].(string), "PROJ-2")
}

func TestMoveIssuesToBacklog_Error(t *testing.T) {
	t.Parallel()

	client, server := newTestClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errorMessages":["server error"]}`))
	}))
	defer server.Close()

	err := client.MoveIssuesToBacklog(context.Background(), []string{"PROJ-1"})
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "moving issues to backlog")
}
