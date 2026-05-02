package issues

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func init() {
	pollRetryDelay = 0
}

func TestPollMoveTask_TerminalComplete(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.MoveTaskStatus{Status: "COMPLETE"})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	status, err := pollMoveTask(context.Background(), client, "task-1")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "COMPLETE", status.Status)
}

func TestPollMoveTask_TerminalFailed(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.MoveTaskStatus{Status: "FAILED"})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	status, err := pollMoveTask(context.Background(), client, "task-1")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "FAILED", status.Status)
}

func TestPollMoveTask_TransientNotFound_ThenComplete(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errorMessages":["not found"]}`))
			return
		}
		_ = json.NewEncoder(w).Encode(api.MoveTaskStatus{Status: "COMPLETE"})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	status, err := pollMoveTask(context.Background(), client, "task-1")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "COMPLETE", status.Status)
	testutil.Equal(t, int32(3), calls.Load())
}

func TestPollMoveTask_PersistentNotFound_ReturnsStatusUnavailable(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errorMessages":["not found"]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	_, err := pollMoveTask(context.Background(), client, "task-1")
	testutil.True(t, errors.Is(err, errStatusUnavailable))
	testutil.Equal(t, int32(maxPollNotFoundRetries+1), calls.Load())
}

func TestPollMoveTask_ContextCancellation(t *testing.T) {
	// Not parallel: mutates package-level pollRetryDelay.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errorMessages":["not found"]}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	origDelay := pollRetryDelay
	pollRetryDelay = 5 * time.Second
	defer func() { pollRetryDelay = origDelay }()

	client := newTestClient(t, server.URL)
	_, err := pollMoveTask(ctx, client, "task-1")
	testutil.True(t, errors.Is(err, context.DeadlineExceeded))
}

func newTestClient(t *testing.T, url string) *api.Client {
	t.Helper()
	client, err := api.New(api.ClientConfig{
		URL:      url,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)
	return client
}
