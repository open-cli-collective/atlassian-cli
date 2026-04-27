package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestNewCheckCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newCheckCmd(opts)

	testutil.Equal(t, cmd.Use, "check <issue-key>")
	testutil.NotNil(t, cmd.Flags().Lookup("require"))
	testutil.NotNil(t, cmd.Flags().Lookup("warn"))
}

func TestFieldResolver_ResolvesByID(t *testing.T) {
	t.Parallel()
	r := newFieldResolver([]api.Field{
		{ID: "summary", Name: "Summary"},
		{ID: "customfield_10035", Name: "Story Points", Custom: true},
	})

	id, display, ok := r.resolve("customfield_10035")
	testutil.Equal(t, ok, true)
	testutil.Equal(t, id, "customfield_10035")
	testutil.Equal(t, display, "Story Points")
}

func TestFieldResolver_ResolvesByDisplayNameCaseInsensitive(t *testing.T) {
	t.Parallel()
	r := newFieldResolver([]api.Field{
		{ID: "customfield_10035", Name: "Story Points", Custom: true},
	})

	id, display, ok := r.resolve("story points")
	testutil.Equal(t, ok, true)
	testutil.Equal(t, id, "customfield_10035")
	testutil.Equal(t, display, "Story Points")

	id, display, ok = r.resolve("STORY POINTS")
	testutil.Equal(t, ok, true)
	testutil.Equal(t, id, "customfield_10035")
	testutil.Equal(t, display, "Story Points")
}

func TestFieldResolver_FallsBackToLiteralCustomfieldID(t *testing.T) {
	t.Parallel()
	r := newFieldResolver(nil) // empty schema

	id, display, ok := r.resolve("customfield_99999")
	testutil.Equal(t, ok, true)
	testutil.Equal(t, id, "customfield_99999")
	testutil.Equal(t, display, "customfield_99999")
}

func TestFieldResolver_UnknownNameDoesNotResolve(t *testing.T) {
	t.Parallel()
	r := newFieldResolver([]api.Field{
		{ID: "summary", Name: "Summary"},
	})

	_, _, ok := r.resolve("totally-unknown")
	testutil.Equal(t, ok, false)
}

func standardFields() []api.Field {
	return []api.Field{
		{ID: "summary", Name: "Summary"},
		{ID: "description", Name: "Description"},
		{ID: "assignee", Name: "Assignee"},
		{ID: "priority", Name: "Priority"},
		{ID: "labels", Name: "Labels"},
		{ID: "customfield_10035", Name: "Story Points", Custom: true},
		{ID: "customfield_10020", Name: "Sprint", Custom: true},
	}
}

func TestBuildCheckResults_RequiredMissingTriggersFailure(t *testing.T) {
	t.Parallel()
	populated := map[string]api.IssueFieldEntry{
		"summary": {ID: "summary", Name: "Summary", Value: "ok"},
	}
	results, missing := buildCheckResults(
		[]string{"Story Points", "Summary"},
		nil,
		standardFields(),
		populated,
		false,
	)
	testutil.Equal(t, missing, 1)
	testutil.Equal(t, len(results), 2)

	byField := map[string]checkResult{}
	for _, r := range results {
		byField[r.display] = r
	}
	testutil.Equal(t, byField["Summary"].status, statusOK)
	testutil.Equal(t, byField["Story Points"].status, statusMissing)
	testutil.Equal(t, byField["Story Points"].level, levelRequired)
}

func TestBuildCheckResults_WarnMissingDoesNotFail(t *testing.T) {
	t.Parallel()
	populated := map[string]api.IssueFieldEntry{}
	results, missing := buildCheckResults(
		nil,
		[]string{"Story Points"},
		standardFields(),
		populated,
		false,
	)
	testutil.Equal(t, missing, 0) // warn-only never fails
	testutil.Equal(t, len(results), 1)
	testutil.Equal(t, results[0].level, levelWarn)
	testutil.Equal(t, results[0].status, statusMissing)
}

func TestBuildCheckResults_DefaultsSilentlySkipUnknownFields(t *testing.T) {
	t.Parallel()
	// Schema lacks Sprint + Story Points (e.g., a non-Agile project).
	fields := []api.Field{
		{ID: "summary", Name: "Summary"},
		{ID: "assignee", Name: "Assignee"},
	}
	populated := map[string]api.IssueFieldEntry{
		"summary": {ID: "summary", Name: "Summary", Value: "ok"},
	}

	results, missing := buildCheckResults(
		nil,
		defaultWarnFields,
		fields,
		populated,
		true, // useDefaults — fields not on schema get dropped, not flagged
	)
	testutil.Equal(t, missing, 0)
	for _, r := range results {
		// Only Summary + Assignee from the default list should resolve here.
		if r.display != "Summary" && r.display != "Assignee" {
			t.Errorf("unexpected default field surfaced when not on schema: %q", r.display)
		}
	}
}

func TestBuildCheckResults_ExplicitUnknownFieldSurfacesAsMissing(t *testing.T) {
	t.Parallel()
	// User explicitly named "Story Points" but schema doesn't have it. With
	// useDefaults=false, we surface this so a typo isn't silent.
	fields := []api.Field{
		{ID: "summary", Name: "Summary"},
	}
	results, missing := buildCheckResults(
		[]string{"Story Points"},
		nil,
		fields,
		map[string]api.IssueFieldEntry{},
		false,
	)
	testutil.Equal(t, missing, 1)
	testutil.Equal(t, len(results), 1)
	testutil.Equal(t, results[0].status, statusMissing)
	testutil.Contains(t, results[0].value, "unknown field")
}

func TestBuildCheckResults_OrdersRequiredFirstThenAlphabetical(t *testing.T) {
	t.Parallel()
	populated := map[string]api.IssueFieldEntry{}
	results, _ := buildCheckResults(
		[]string{"Summary"},
		[]string{"Assignee", "Story Points"},
		standardFields(),
		populated,
		false,
	)
	// Expected order: REQUIRED Summary, WARN Assignee, WARN Story Points.
	testutil.Equal(t, len(results), 3)
	testutil.Equal(t, results[0].display, "Summary")
	testutil.Equal(t, results[0].level, levelRequired)
	testutil.Equal(t, results[1].display, "Assignee")
	testutil.Equal(t, results[2].display, "Story Points")
}

// --- end-to-end runCheck tests through an httptest server ---
// These tests are non-parallel: cache.SetRootForTest /
// SetInstanceKeyForTest (called via seedCacheForIssues) are process-globals
// that race with t.Parallel() tests writing them. Same convention as
// fields_test.go.

func newCheckTestServer(t *testing.T, issue api.Issue) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/field" {
			t.Fatal("live /field must not be called when cache is fresh")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(issue)
	}))
}

func TestRunCheck_RequiredMissingReturnsError(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("fields", "24h", standardFields()))

	issue := api.Issue{
		Key: "TEST-1",
		Fields: api.IssueFields{
			Summary: "has summary",
			// no Story Points
		},
	}

	server := newCheckTestServer(t, issue)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCheck(context.Background(), opts, "TEST-1", []string{"Story Points"}, nil)
	if err == nil {
		t.Fatal("expected runCheck to return an error when a required field is missing")
	}
	if !strings.Contains(err.Error(), "1 required field") {
		t.Fatalf("error should describe missing required field count, got: %v", err)
	}
	testutil.Contains(t, stdout.String(), "Story Points")
	testutil.Contains(t, stdout.String(), statusMissing)
}

func TestRunCheck_DefaultWarnListNeverErrors(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, cache.WriteResource("fields", "24h", standardFields()))

	issue := api.Issue{
		Key: "TEST-2",
		Fields: api.IssueFields{
			Summary: "has summary",
		},
	}

	server := newCheckTestServer(t, issue)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCheck(context.Background(), opts, "TEST-2", nil, nil)
	testutil.RequireNoError(t, err) // default-mode is warn-only; no required → no error
	testutil.Contains(t, stdout.String(), "Story Points")
	testutil.Contains(t, stdout.String(), "Sprint")
	testutil.Contains(t, stdout.String(), levelWarn)
}
