package issues

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunDelete_SingleIssue(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, []string{"PROJ-123"}, true)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted PROJ-123\n")
	testutil.Equal(t, stderr.String(), "")
}

func TestRunDelete_MultipleIssues(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, []string{"PROJ-1", "PROJ-2", "PROJ-3"}, true)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted PROJ-1\nDeleted PROJ-2\nDeleted PROJ-3\n")
	testutil.Equal(t, stderr.String(), "")
}

func TestRunDelete_PartialFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue/PROJ-2" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errorMessages":["Issue does not exist"]}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, []string{"PROJ-1", "PROJ-2", "PROJ-3"}, true)
	testutil.Error(t, err)
	if !errors.Is(err, root.ErrAlreadyReported) {
		t.Fatalf("expected ErrAlreadyReported, got %v", err)
	}
	testutil.Equal(t, stdout.String(), "Deleted PROJ-1\nDeleted PROJ-3\n")
	testutil.Contains(t, stderr.String(), "Failed to delete PROJ-2")
}

func TestRunDelete_PromptDeclined(t *testing.T) {
	t.Parallel()

	client, err := api.New(api.ClientConfig{URL: "https://test.atlassian.net", Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  bytes.NewBufferString("n\n"),
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, []string{"PROJ-123"}, false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deletion cancelled.\n")
	testutil.Contains(t, stderr.String(), "permanently delete issue PROJ-123")
}

func TestRunDelete_BatchPromptDeclined(t *testing.T) {
	t.Parallel()

	client, err := api.New(api.ClientConfig{URL: "https://test.atlassian.net", Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  bytes.NewBufferString("n\n"),
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, []string{"PROJ-1", "PROJ-2", "PROJ-3"}, false)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stderr.String(), "3 issues")
	testutil.Equal(t, stdout.String(), "Deletion cancelled.\n")
}

func TestRunDelete_PromptAccepted(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  bytes.NewBufferString("y\n"),
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, []string{"PROJ-123"}, false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted PROJ-123\n")
}

func TestRunDelete_BatchPromptAccepted(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  bytes.NewBufferString("y\n"),
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, []string{"PROJ-1", "PROJ-2"}, false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted PROJ-1\nDeleted PROJ-2\n")
}

func TestRunDelete_AllFailures(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errorMessages":["Issue does not exist"]}`))
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, []string{"PROJ-1", "PROJ-2"}, true)
	testutil.Error(t, err)
	if !errors.Is(err, root.ErrAlreadyReported) {
		t.Fatalf("expected ErrAlreadyReported, got %v", err)
	}
	testutil.Equal(t, stdout.String(), "")
	testutil.Contains(t, stderr.String(), "Failed to delete PROJ-1")
	testutil.Contains(t, stderr.String(), "Failed to delete PROJ-2")
}
