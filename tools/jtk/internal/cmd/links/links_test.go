package links

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestNewListCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newListCmd(opts)

	assert.Equal(t, "list <issue-key>", cmd.Use)
	assert.Equal(t, "List links on an issue", cmd.Short)
}

func TestRunList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"fields": map[string]interface{}{
				"issuelinks": []map[string]interface{}{
					{
						"id":   "10001",
						"type": map[string]string{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
						"outwardIssue": map[string]interface{}{
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(opts, "PROJ-123")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "PROJ-456")
	assert.Contains(t, stdout.String(), "Blocks")
	// OutwardIssue is set → current issue is the inward side → show inward direction
	assert.Contains(t, stdout.String(), "is blocked by")
}

func TestRunList_NoLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"fields": map[string]interface{}{
				"issuelinks": []interface{}{},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runList(opts, "PROJ-123")
	require.NoError(t, err)
}

func TestRunCreate(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/issueLinkType":
			json.NewEncoder(w).Encode(map[string]interface{}{
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(opts, "PROJ-123", "PROJ-456", "Blocks")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Created")

	var req api.CreateIssueLinkRequest
	err = json.Unmarshal(capturedBody, &req)
	require.NoError(t, err)
	assert.Equal(t, "Blocks", req.Type.Name)
	assert.Equal(t, "PROJ-123", req.OutwardIssue.Key)
	assert.Equal(t, "PROJ-456", req.InwardIssue.Key)
}

func TestRunCreate_InvalidType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issueLinkTypes": []map[string]string{
				{"id": "1", "name": "Blocks"},
				{"id": "2", "name": "Relates"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(opts, "A", "B", "InvalidType")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "Blocks")
}

func TestRunDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issueLink/10001", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runDelete(opts, "10001")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Deleted")
}

func TestRunTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issueLinkTypes": []map[string]string{
				{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
				{"id": "2", "name": "Relates", "inward": "relates to", "outward": "relates to"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runTypes(opts)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Blocks")
	assert.Contains(t, stdout.String(), "Relates")
}
