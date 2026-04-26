package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunArchive_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue/archive" && r.Method == "PUT" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.ArchiveResult{NumberUpdated: 1})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runArchive(context.Background(), opts, []string{"TEST-1"})
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Archived TEST-1")
}

func TestRunArchive_PartialFailure(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue/archive" && r.Method == "PUT" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.ArchiveResult{
				NumberUpdated: 1,
				Errors: map[string]api.ArchiveError{
					"PERMISSION_DENIED": {
						Count:          1,
						IssueIdsOrKeys: []string{"TEST-2"},
						Message:        "Permission denied",
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runArchive(context.Background(), opts, []string{"TEST-1", "TEST-2"})
	testutil.NotNil(t, err)
	testutil.Contains(t, stdout.String(), "Archived TEST-1")
	testutil.Contains(t, stderr.String(), "Permission denied")
}

func TestRunArchive_IDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue/archive" && r.Method == "PUT" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.ArchiveResult{NumberUpdated: 2})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runArchive(context.Background(), opts, []string{"TEST-1", "TEST-2"})
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "TEST-1")
	testutil.Contains(t, stdout.String(), "TEST-2")
}
