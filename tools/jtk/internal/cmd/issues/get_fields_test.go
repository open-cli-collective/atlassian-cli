package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// capturingGetServer handles both /issue/* and /field requests. Tests
// introspect fieldsCalls to verify whether GetFields was invoked.
type capturingGetServer struct {
	server      *httptest.Server
	fieldsCalls int
	issueCalls  int
}

func newCapturingGetServer(t *testing.T, issue api.Issue, fieldsResp []api.Field) *capturingGetServer {
	t.Helper()
	cs := &capturingGetServer{}
	cs.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/field") {
			cs.fieldsCalls++
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(fieldsResp)
			return
		}
		if strings.Contains(r.URL.Path, "/issue/") {
			cs.issueCalls++
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(issue)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	return cs
}

func newGetOpts(t *testing.T, cs *capturingGetServer) (*root.Options, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: cs.server.URL, Email: "e@x", APIToken: "t"})
	testutil.RequireNoError(t, err)
	var stdout, stderr bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &stderr, Output: "table"}
	opts.SetAPIClient(client)
	return opts, &stdout, &stderr
}

func fullIssue() api.Issue {
	return api.Issue{
		Key: "TEST-1",
		Fields: api.IssueFields{
			Summary:     "summary text",
			Description: &api.Description{Text: "desc body"},
			Status:      &api.Status{Name: "Open"},
			IssueType:   &api.IssueType{Name: "Task"},
			Priority:    &api.Priority{Name: "High"},
			Assignee:    &api.User{DisplayName: "Alice"},
			Project:     &api.Project{Key: "TEST"},
		},
	}
}

func TestRunGet_Fields_ProjectsDetailToSelected(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssue(), nil)
	defer cs.server.Close()

	opts, stdout, _ := newGetOpts(t, cs)
	err := runGet(context.Background(), opts, "TEST-1", false, "Status")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	// Identity (Key) must always appear.
	testutil.Contains(t, output, "Key: TEST-1")
	testutil.Contains(t, output, "Status: Open")
	// Dropped fields must NOT appear.
	if strings.Contains(output, "Assignee") {
		t.Errorf("Assignee should be dropped when --fields=Status: %q", output)
	}
	if strings.Contains(output, "Priority") {
		t.Errorf("Priority should be dropped when --fields=Status: %q", output)
	}
}

func TestRunGet_Fields_URL_SyntheticColumn(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssue(), nil)
	defer cs.server.Close()

	opts, stdout, _ := newGetOpts(t, cs)
	err := runGet(context.Background(), opts, "TEST-1", false, "URL")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Key: TEST-1")
	testutil.Contains(t, output, "URL:")
}

func TestRunGet_Fields_HumanName_TriggersFieldsFetch(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssue(), []api.Field{
		{ID: "issuetype", Name: "Issue Type"},
	})
	defer cs.server.Close()

	opts, stdout, _ := newGetOpts(t, cs)
	err := runGet(context.Background(), opts, "TEST-1", false, "Issue Type")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Key: TEST-1")
	testutil.Contains(t, output, "Type: Task")
	if cs.fieldsCalls != 1 {
		t.Errorf("human-name resolution must trigger GetFields once; got %d", cs.fieldsCalls)
	}
}

func TestRunGet_Fields_UnknownToken_Errors(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssue(), []api.Field{})
	defer cs.server.Close()

	opts, _, _ := newGetOpts(t, cs)
	err := runGet(context.Background(), opts, "TEST-1", false, "bogus")
	var ufe *projection.UnknownFieldError
	if !errors.As(err, &ufe) {
		t.Fatalf("expected UnknownFieldError, got %v", err)
	}
}

func TestRunGet_Fields_UnrenderedField_ByFieldID_Errors(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssue(), []api.Field{
		{ID: "customfield_99999", Name: "Phantom"},
	})
	defer cs.server.Close()

	opts, _, _ := newGetOpts(t, cs)
	err := runGet(context.Background(), opts, "TEST-1", false, "customfield_99999")
	var ure *projection.UnrenderedFieldError
	if !errors.As(err, &ure) {
		t.Fatalf("expected UnrenderedFieldError, got %v", err)
	}
	testutil.Equal(t, "Phantom", ure.JiraName)
	testutil.Equal(t, "issues get", ure.Command)
}

func TestRunGet_Fields_WithJSON_Errors(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssue(), nil)
	defer cs.server.Close()

	opts, _, _ := newGetOpts(t, cs)
	opts.Output = "json"
	err := runGet(context.Background(), opts, "TEST-1", false, "Status")
	if err == nil {
		t.Fatalf("expected error when --fields combined with --output json")
	}
	testutil.Contains(t, err.Error(), "not supported with --output json")
}

func TestRunGet_FieldsWithIDOnly_IDWins(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssue(), nil)
	defer cs.server.Close()

	opts, stdout, _ := newGetOpts(t, cs)
	opts.IDOnly = true
	err := runGet(context.Background(), opts, "TEST-1", false, "Status")
	testutil.RequireNoError(t, err)

	// Bare key on stdout; nothing else.
	if stdout.String() != "TEST-1\n" {
		t.Errorf("expected bare key, got %q", stdout.String())
	}
}

// Under --id, projection.Resolve is skipped entirely. A human-name --fields
// token would normally trigger a GetFields() call; --id must suppress it.
func TestRunGet_IDOnly_SkipsFieldsResolution(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssue(), []api.Field{
		{ID: "issuetype", Name: "Issue Type"},
	})
	defer cs.server.Close()

	opts, _, _ := newGetOpts(t, cs)
	opts.IDOnly = true
	err := runGet(context.Background(), opts, "TEST-1", false, "Issue Type")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 0, cs.fieldsCalls)
}

func TestRunGet_IDOnly_BypassesFieldsValidation(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssue(), []api.Field{})
	defer cs.server.Close()

	opts, stdout, _ := newGetOpts(t, cs)
	opts.IDOnly = true
	err := runGet(context.Background(), opts, "TEST-1", false, "bogus")
	testutil.RequireNoError(t, err)
	if stdout.String() != "TEST-1\n" {
		t.Errorf("expected bare key, got %q", stdout.String())
	}
}

func TestRunGet_IDOnly_BypassesJSONFieldsRejection(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssue(), nil)
	defer cs.server.Close()

	opts, stdout, _ := newGetOpts(t, cs)
	opts.IDOnly = true
	opts.Output = "json"
	err := runGet(context.Background(), opts, "TEST-1", false, "Status")
	testutil.RequireNoError(t, err)
	if stdout.String() != "TEST-1\n" {
		t.Errorf("expected bare key, got %q", stdout.String())
	}
}

// fullIssueWithLongDescription returns a TEST-1 fixture whose Description
// exceeds the 200-char truncation threshold so the body-vs-fulltext
// interaction is observable.
func fullIssueWithLongDescription() api.Issue {
	issue := fullIssue()
	issue.Fields.Description = &api.Description{Text: strings.Repeat("D", 300)}
	return issue
}

// AC1 (issues get): description suppressed by --fields when not selected.
// Long description text MUST be absent from output.
func TestRunGet_Fields_SuppressesDescription_WhenNotSelected(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssueWithLongDescription(), nil)
	defer cs.server.Close()

	opts, stdout, _ := newGetOpts(t, cs)
	err := runGet(context.Background(), opts, "TEST-1", false, "Summary,Status")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Key: TEST-1")
	testutil.Contains(t, output, "Status: Open")
	if strings.Contains(output, "Description") {
		t.Errorf("Description label should not appear: %q", output)
	}
	if strings.Contains(output, "DDDD") {
		t.Errorf("Description body text leaked into output: %q", output)
	}
}

// AC1 (issues get): description selected without --fulltext is truncated.
func TestRunGet_Fields_Description_TruncatedWithoutFullText(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssueWithLongDescription(), nil)
	defer cs.server.Close()

	opts, stdout, _ := newGetOpts(t, cs)
	err := runGet(context.Background(), opts, "TEST-1", false, "Description")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Description:")
	testutil.Contains(t, output, "[truncated, use --fulltext for complete text]")
}

// AC1+AC3 (issues get): description selected WITH --fulltext shows full body
// and no truncation marker.
func TestRunGet_Fields_Description_FullTextWhenSelected(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssueWithLongDescription(), nil)
	defer cs.server.Close()

	opts, stdout, _ := newGetOpts(t, cs)
	err := runGet(context.Background(), opts, "TEST-1", true, "Description")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Description:")
	testutil.Contains(t, output, strings.Repeat("D", 300))
	testutil.NotContains(t, output, "[truncated")
}

// AC3 (issues get): --fulltext is a no-op when Description is not selected.
// Output must not contain description text even though noTruncate=true.
func TestRunGet_Fields_FullTextNoOp_WhenDescriptionNotSelected(t *testing.T) {
	t.Parallel()
	cs := newCapturingGetServer(t, fullIssueWithLongDescription(), nil)
	defer cs.server.Close()

	opts, stdout, _ := newGetOpts(t, cs)
	err := runGet(context.Background(), opts, "TEST-1", true, "Summary")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Summary:")
	if strings.Contains(output, "Description") {
		t.Errorf("Description label should not appear: %q", output)
	}
	if strings.Contains(output, "DDDD") {
		t.Errorf("Description body text leaked even though Description not selected: %q", output)
	}
}
