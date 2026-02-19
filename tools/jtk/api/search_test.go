package api //nolint:revive // package name is intentional

import (
	"context"
	"net/http"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestSearchAll_CancelledContext(t *testing.T) {
	t.Parallel()
	client, server := newTestClientWithServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.SearchAll(ctx, "project = TEST", 100)
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "searching all issues")
}
