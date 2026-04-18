package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

// stubMoveServer answers the bulk-move happy path (GetIssue, MoveIssues,
// GetMoveTaskStatus). It also echoes the projects/issuetypes refresh calls
// that the resolver will attempt on a no-match, keeping the cache stable
// across refresh-retry. Callers control the echo set via `targetTypes`.
func stubMoveServer(t *testing.T, sourceType string, targetTypes []api.IssueType, captured *[]byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/PROJ-1") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(api.Issue{
				Key: "PROJ-1",
				Fields: api.IssueFields{
					Project:   &api.Project{Key: "PROJ"},
					IssueType: &api.IssueType{ID: "10000", Name: sourceType},
				},
			})
		case r.URL.Path == "/rest/api/3/bulk/issues/move" && r.Method == http.MethodPost:
			*captured, _ = io.ReadAll(r.Body)
			_ = json.NewEncoder(w).Encode(api.MoveIssuesResponse{TaskID: "task-1"})
		case strings.HasPrefix(r.URL.Path, "/rest/api/3/bulk/queue/") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(api.MoveTaskStatus{
				TaskID:   "task-1",
				Status:   "COMPLETE",
				Progress: 100,
				Result:   &api.MoveTaskResult{Successful: []string{"PROJ-1"}},
			})
		// Refresh endpoints: echo the seeded fixtures so refresh-retry
		// doesn't destabilize the cache state under test.
		case r.URL.Path == "/rest/api/3/project" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode([]api.Project{
				{Key: "TARGET", Name: "Target"}, {Key: "PROJ", Name: "Source"},
			})
		case strings.HasPrefix(r.URL.Path, "/rest/api/3/project/TARGET") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(struct {
				IssueTypes []api.IssueType `json:"issueTypes"`
			}{IssueTypes: targetTypes})
		case strings.HasPrefix(r.URL.Path, "/rest/api/3/project/PROJ") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(struct {
				IssueTypes []api.IssueType `json:"issueTypes"`
			}{IssueTypes: []api.IssueType{}})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestRunMove_ExplicitTypeResolvesViaCache(t *testing.T) {
	seedCacheForIssues(t)
	// Override: TARGET has Task=10050 so we can verify the resolved ID
	// flows into the move request.
	testutil.RequireNoError(t, cache.WriteResource("issuetypes", "24h", map[string][]api.IssueType{
		"TARGET": {{ID: "10050", Name: "Task"}},
	}))
	testutil.RequireNoError(t, cache.WriteResource("projects", "24h", []api.Project{
		{Key: "TARGET", Name: "Target"}, {Key: "PROJ", Name: "Source"},
	}))

	targetTypes := []api.IssueType{{ID: "10050", Name: "Task"}}
	var moveBody []byte
	server := stubMoveServer(t, "Task", targetTypes, &moveBody)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runMove(context.Background(), opts, []string{"PROJ-1"}, "TARGET", "Task", false, true)
	testutil.RequireNoError(t, err)

	var req api.MoveIssuesRequest
	testutil.RequireNoError(t, json.Unmarshal(moveBody, &req))
	_, ok := req.TargetToSourcesMapping["TARGET,10050"]
	testutil.True(t, ok, "expected mapping for TARGET,10050")
}

func TestRunMove_DefaultsFromSourceTypeViaCache(t *testing.T) {
	// --to-type omitted. Source issue is a Task; target has Task in its
	// cached issuetypes, so the default must pick it up via the resolver.
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("issuetypes", "24h", map[string][]api.IssueType{
		"TARGET": {
			{ID: "10060", Name: "Story"},
			{ID: "10061", Name: "Task"},
		},
	}))
	testutil.RequireNoError(t, cache.WriteResource("projects", "24h", []api.Project{
		{Key: "TARGET", Name: "Target"}, {Key: "PROJ", Name: "Source"},
	}))

	targetTypes := []api.IssueType{
		{ID: "10060", Name: "Story"},
		{ID: "10061", Name: "Task"},
	}
	var moveBody []byte
	server := stubMoveServer(t, "Task", targetTypes, &moveBody)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runMove(context.Background(), opts, []string{"PROJ-1"}, "TARGET", "", false, true)
	testutil.RequireNoError(t, err)

	var req api.MoveIssuesRequest
	testutil.RequireNoError(t, json.Unmarshal(moveBody, &req))
	_, ok := req.TargetToSourcesMapping["TARGET,10061"]
	testutil.True(t, ok, "expected default to resolve to Task ID 10061")
}

func TestRunMove_SourceTypeMissingFallsBackToCachedNonSubtask(t *testing.T) {
	// Source is Epic but target project has no Epic. Resolver misses;
	// defaultCachedIssueType should return Task (first non-subtask).
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("issuetypes", "24h", map[string][]api.IssueType{
		"TARGET": {
			{ID: "10070", Name: "Sub-task", Subtask: true},
			{ID: "10071", Name: "Task"},
			{ID: "10072", Name: "Story"},
		},
	}))
	testutil.RequireNoError(t, cache.WriteResource("projects", "24h", []api.Project{
		{Key: "TARGET", Name: "Target"}, {Key: "PROJ", Name: "Source"},
	}))

	targetTypes := []api.IssueType{
		{ID: "10070", Name: "Sub-task", Subtask: true},
		{ID: "10071", Name: "Task"},
		{ID: "10072", Name: "Story"},
	}
	var moveBody []byte
	server := stubMoveServer(t, "Epic", targetTypes, &moveBody)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runMove(context.Background(), opts, []string{"PROJ-1"}, "TARGET", "", false, true)
	testutil.RequireNoError(t, err)

	var req api.MoveIssuesRequest
	testutil.RequireNoError(t, json.Unmarshal(moveBody, &req))
	// First non-subtask in cached order = Task (10071).
	_, ok := req.TargetToSourcesMapping["TARGET,10071"]
	testutil.True(t, ok, "expected default fallback to first cached non-subtask")
}

func TestRunMove_NoCachedIssueTypesPromptsToSpecifyType(t *testing.T) {
	// --to-type omitted. Source is Epic, target project has NO cached
	// issuetypes at all. Resolver should surface an actionable error
	// rather than silently fetching from the API.
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("issuetypes", "24h", map[string][]api.IssueType{
		// Only PROJ's types; TARGET absent.
		"PROJ": {{ID: "1", Name: "Epic"}},
	}))
	testutil.RequireNoError(t, cache.WriteResource("projects", "24h", []api.Project{
		{Key: "TARGET", Name: "Target"}, {Key: "PROJ", Name: "Source"},
	}))

	// The test server must not be hit for GetProjectIssueTypes — the whole
	// point is that we no longer do a live fallback.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/api/3/issue/PROJ-1") {
			_ = json.NewEncoder(w).Encode(api.Issue{
				Key: "PROJ-1",
				Fields: api.IssueFields{
					Project:   &api.Project{Key: "PROJ"},
					IssueType: &api.IssueType{ID: "1", Name: "Epic"},
				},
			})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/rest/api/3/project/TARGET") {
			// Seed the resolver's refresh-retry with no matching type so
			// the resolver reports NotFound, forcing the cached-fallback
			// path.
			_ = json.NewEncoder(w).Encode(struct {
				IssueTypes []api.IssueType `json:"issueTypes"`
			}{IssueTypes: []api.IssueType{}})
			return
		}
		// Explicitly fail if the old live GetProjectIssueTypes path runs.
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runMove(context.Background(), opts, []string{"PROJ-1"}, "TARGET", "", false, true)
	if err == nil {
		t.Fatalf("expected error when no cached types available for target project")
	}
	if !strings.Contains(err.Error(), "--to-type") {
		t.Fatalf("expected error to suggest --to-type, got: %v", err)
	}
}
