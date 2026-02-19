package links

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestNewListCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newListCmd(opts)

	testutil.Equal(t, cmd.Use, "list <issue-key>")
	testutil.Equal(t, cmd.Short, "List links on an issue")
}

func TestRunList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fields": map[string]any{
				"issuelinks": []map[string]any{
					{
						"id":   "10001",
						"type": map[string]string{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
						"outwardIssue": map[string]any{
							"key":    "PROJ-456",
							"fields": map[string]string{"summary": "Blocked issue"},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(opts, "PROJ-123")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "PROJ-456")
	testutil.Contains(t, stdout.String(), "Blocks")
	// OutwardIssue is set → current issue is the inward side → show inward direction
	testutil.Contains(t, stdout.String(), "is blocked by")
}

func TestRunList_NoLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fields": map[string]any{
				"issuelinks": []any{},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runList(opts, "PROJ-123")
	testutil.RequireNoError(t, err)
}

func TestRunCreate(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/issueLinkType":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issueLinkTypes": []map[string]string{
					{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
				},
			})
		case "/rest/api/3/issueLink":
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(opts, "PROJ-123", "PROJ-456", "Blocks")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Created")

	var req api.CreateIssueLinkRequest
	err = json.Unmarshal(capturedBody, &req)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, req.Type.Name, "Blocks")
	testutil.Equal(t, req.OutwardIssue.Key, "PROJ-123")
	testutil.Equal(t, req.InwardIssue.Key, "PROJ-456")
}

func TestRunCreate_InvalidType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issueLinkTypes": []map[string]string{
				{"id": "1", "name": "Blocks"},
				{"id": "2", "name": "Relates"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(opts, "A", "B", "InvalidType")
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "not found")
	testutil.Contains(t, err.Error(), "Blocks")
}

func TestRunDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issueLink/10001")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runDelete(opts, "10001")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Deleted")
}

func TestRunTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issueLinkTypes": []map[string]string{
				{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
				{"id": "2", "name": "Relates", "inward": "relates to", "outward": "relates to"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runTypes(opts)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Blocks")
	testutil.Contains(t, stdout.String(), "Relates")
}
