package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
)

// captureJQLServer captures the JQL from the search request body so tests
// can assert that --sprint resolves to a numeric ID before hitting the API.
func captureJQLServer(t *testing.T, jqlOut *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/search/jql") {
			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(body, &payload)
			if v, ok := payload["jql"].(string); ok {
				*jqlOut = v
			}
			_ = json.NewEncoder(w).Encode(api.JQLSearchResult{IsLast: true})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestRunList_SprintNameResolvesToID(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, seedSprints(map[int][]api.Sprint{
		23: {{ID: 125, Name: "MON Sprint 70", State: "active"}},
	}))

	var jql string
	server := captureJQLServer(t, &jql)
	defer server.Close()

	opts, _, _ := newListOpts(t, server)
	err := runList(context.Background(), opts, "PROJ", "MON Sprint 70", 25, "", false, "")
	testutil.RequireNoError(t, err)
	if !strings.Contains(jql, "sprint = 125") {
		t.Fatalf("expected JQL to contain 'sprint = 125', got: %q", jql)
	}
}

func TestRunList_SprintNumericPassThrough(t *testing.T) {
	seedCacheForIssues(t)
	testutil.RequireNoError(t, seedSprints(map[int][]api.Sprint{
		23: {{ID: 125, Name: "MON Sprint 70"}},
	}))

	var jql string
	server := captureJQLServer(t, &jql)
	defer server.Close()

	opts, _, _ := newListOpts(t, server)
	err := runList(context.Background(), opts, "PROJ", "999", 25, "", false, "")
	testutil.RequireNoError(t, err)
	if !strings.Contains(jql, "sprint = 999") {
		t.Fatalf("expected JQL to contain 'sprint = 999', got: %q", jql)
	}
}

func TestRunList_SprintCurrentUsesOpenSprints(t *testing.T) {
	seedCacheForIssues(t)

	var jql string
	server := captureJQLServer(t, &jql)
	defer server.Close()

	opts, _, _ := newListOpts(t, server)
	err := runList(context.Background(), opts, "PROJ", "current", 25, "", false, "")
	testutil.RequireNoError(t, err)
	if !strings.Contains(jql, "openSprints()") {
		t.Fatalf("expected JQL to contain 'openSprints()', got: %q", jql)
	}
}

// seedSprints writes a sprints cache envelope. Pairs with the isolated
// cache set up by seedCacheForIssues.
func seedSprints(byBoard map[int][]api.Sprint) error {
	return cache.WriteResource("sprints", "24h", byBoard)
}

// listResultServer returns a fixed set of issues with configurable IsLast.
// `keys` drives which issue keys the mock returns.
func listResultServer(t *testing.T, keys []string, isLast bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		issues := make([]api.Issue, len(keys))
		for i, k := range keys {
			issues[i] = api.Issue{
				Key: k,
				Fields: api.IssueFields{
					Summary:   "summary for " + k,
					Status:    &api.Status{Name: "Open"},
					IssueType: &api.IssueType{Name: "Task"},
				},
			}
		}
		result := api.JQLSearchResult{Issues: issues, IsLast: isLast}
		if !isLast {
			result.NextPageToken = "next-token"
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	}))
}

func newListOpts(t *testing.T, server *httptest.Server) (*root.Options, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	// runList resolves --project through the cache, which dereferences
	// cache.InstanceKey(). On CI (no JIRA_URL, no config file) InstanceKey
	// returns ErrNoInstance and the resolver propagates it. Override the
	// instance key so the resolver gets a clean "cache miss" path instead.
	// Only set the instance key here, NOT the cache root — tests that seed
	// cache data via seedCacheForIssues set their own tempdir root; a second
	// SetRootForTest here would blow that away.
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "e@x", APIToken: "t"})
	testutil.RequireNoError(t, err)
	var stdout, stderr bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)
	return opts, &stdout, &stderr
}

func TestRunList_DefaultPaginationOnStdout(t *testing.T) {
	t.Parallel()
	server := listResultServer(t, []string{"TEST-1", "TEST-2"}, false)
	defer server.Close()

	opts, stdout, stderr := newListOpts(t, server)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	if !strings.Contains(stdout.String(), "TEST-1") {
		t.Errorf("stdout missing issue key: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "More results available") {
		t.Errorf("pagination hint should be on stdout, got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), "More results available") {
		t.Errorf("pagination hint should NOT be on stderr: %q", stderr.String())
	}
}

func TestRunList_IDOnlyEmitsKeysOnePerLine(t *testing.T) {
	t.Parallel()
	server := listResultServer(t, []string{"TEST-1", "TEST-2", "TEST-3"}, true)
	defer server.Close()

	opts, stdout, stderr := newListOpts(t, server)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	want := "TEST-1\nTEST-2\nTEST-3\n"
	if stdout.String() != want {
		t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestRunList_IDOnlyWithMoreResultsAppendsContinuation(t *testing.T) {
	t.Parallel()
	server := listResultServer(t, []string{"TEST-1", "TEST-2"}, false)
	defer server.Close()

	opts, stdout, _ := newListOpts(t, server)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	want := "TEST-1\nTEST-2\nMore results available (next: next-token)\n"
	if stdout.String() != want {
		t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRunList_EmptyDefault_NoIssuesFoundOnStdout(t *testing.T) {
	t.Parallel()
	server := listResultServer(t, nil, true)
	defer server.Close()

	opts, stdout, stderr := newListOpts(t, server)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	if !strings.Contains(stdout.String(), "No issues found") {
		t.Errorf("expected 'No issues found' on stdout, got: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestRunList_EmptyWithMoreResults_EmitsOnlyPaginationHint(t *testing.T) {
	t.Parallel()
	// Empty page with IsLast=false (more pages exist). The continuation hint
	// alone reaches stdout so agents keep paging; the "No issues found"
	// message is suppressed because the result set is not actually empty —
	// only this page is. Emitting both would self-contradict.
	server := listResultServer(t, nil, false)
	defer server.Close()

	opts, stdout, stderr := newListOpts(t, server)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	if !strings.Contains(stdout.String(), "More results available") {
		t.Errorf("pagination hint should appear on stdout; got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "No issues found") {
		t.Errorf("'No issues found' must not co-occur with pagination hint; got %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestRunList_EmptyWithIDOnly_EmitsNothing(t *testing.T) {
	t.Parallel()
	server := listResultServer(t, nil, true)
	defer server.Close()

	opts, stdout, stderr := newListOpts(t, server)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "")
	testutil.RequireNoError(t, err)

	if stdout.String() != "" {
		t.Errorf("stdout should be empty under --id with zero results, got: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

// capturingServer records each inbound request body plus the path and
// responds with a canned issues payload. Tests introspect requests to verify
// fetch-optimization behavior (which Fields were sent to the Search API,
// whether GetFields was called, etc.).
type capturingServer struct {
	server         *httptest.Server
	searchCaptured *api.SearchRequest
	fieldsCalls    int
}

func newCapturingServer(t *testing.T, keys []string, isLast bool, fieldsResp []api.Field) *capturingServer {
	t.Helper()
	cs := &capturingServer{searchCaptured: &api.SearchRequest{}}
	cs.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/field") {
			cs.fieldsCalls++
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(fieldsResp)
			return
		}
		if strings.Contains(r.URL.Path, "/search") {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, cs.searchCaptured)
			issues := make([]api.Issue, len(keys))
			for i, k := range keys {
				issues[i] = api.Issue{
					Key: k,
					Fields: api.IssueFields{
						Summary:   "summary for " + k,
						Status:    &api.Status{Name: "Open"},
						IssueType: &api.IssueType{Name: "Task"},
						Assignee:  &api.User{DisplayName: "Alice"},
					},
				}
			}
			result := api.JQLSearchResult{Issues: issues, IsLast: isLast}
			if !isLast {
				result.NextPageToken = "next-token"
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(result)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	return cs
}

func newOptsFor(t *testing.T, cs *capturingServer) (*root.Options, *bytes.Buffer, *bytes.Buffer) {
	return newListOpts(t, cs.server)
}

func TestRunList_Fields_HeaderAliases_ProjectsTable(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, nil)
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "SUMMARY,STATUS")
	testutil.RequireNoError(t, err)

	// Header row in the pipe-delimited agent output should be KEY | SUMMARY | STATUS.
	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) == 0 {
		t.Fatalf("empty output")
	}
	if lines[0] != "KEY | SUMMARY | STATUS" {
		t.Errorf("header mismatch: got %q", lines[0])
	}
	if cs.fieldsCalls != 0 {
		t.Errorf("header aliases must not trigger GetFields; got %d calls", cs.fieldsCalls)
	}
	// Derived fetch: identity KEY contributes nothing; SUMMARY→summary, STATUS→status.
	got := cs.searchCaptured.Fields
	if len(got) != 2 || got[0] != "status" || got[1] != "summary" {
		t.Errorf("fetch set: got %v; want [status summary]", got)
	}
}

// Projection must coexist with pagination state: when a --fields projection
// runs against a multi-page result (hasMore=true), ProjectTable rewrites the
// TableSection but the pagination hint section must survive untouched. A
// regression that stripped the hint would only surface at runtime against a
// multi-page paginated table result.
func TestRunList_Fields_Projection_PreservesPaginationHint(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, false, nil) // isLast=false → hasMore=true
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "SUMMARY,STATUS")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "KEY | SUMMARY | STATUS")
	testutil.Contains(t, out, "next: next-token")
}

func TestRunList_Fields_JiraFieldIDs_ProjectsTable(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, nil)
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "summary,assignee")
	testutil.RequireNoError(t, err)

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if lines[0] != "KEY | SUMMARY | ASSIGNEE" {
		t.Errorf("header mismatch: got %q", lines[0])
	}
}

func TestRunList_Fields_HumanName_TriggersFieldsFetch(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, []api.Field{
		{ID: "issuetype", Name: "Issue Type"},
	})
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "Issue Type")
	testutil.RequireNoError(t, err)

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if lines[0] != "KEY | TYPE" {
		t.Errorf("header mismatch: got %q", lines[0])
	}
	if cs.fieldsCalls != 1 {
		t.Errorf("human-name resolution must trigger GetFields exactly once; got %d", cs.fieldsCalls)
	}
}

func TestRunList_Fields_UnknownToken_Errors(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, []api.Field{})
	defer cs.server.Close()

	opts, _, _ := newOptsFor(t, cs)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "bogus")
	var ufe *projection.UnknownFieldError
	if !errors.As(err, &ufe) {
		t.Fatalf("expected UnknownFieldError, got %v", err)
	}
}

func TestRunList_Fields_UnrenderedField_ByHumanName_Errors(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, []api.Field{
		{ID: "customfield_99999", Name: "Phantom"},
	})
	defer cs.server.Close()

	opts, _, _ := newOptsFor(t, cs)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "Phantom")
	var ure *projection.UnrenderedFieldError
	if !errors.As(err, &ure) {
		t.Fatalf("expected UnrenderedFieldError, got %v", err)
	}
	testutil.Equal(t, "Phantom", ure.JiraName)
	testutil.Equal(t, "issues list", ure.Command)
}

func TestRunList_Fields_UnrenderedField_ByFieldID_Errors(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, []api.Field{
		{ID: "customfield_99999", Name: "Phantom"},
	})
	defer cs.server.Close()

	opts, _, _ := newOptsFor(t, cs)
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "customfield_99999")
	var ure *projection.UnrenderedFieldError
	if !errors.As(err, &ure) {
		t.Fatalf("expected UnrenderedFieldError, got %v", err)
	}
	testutil.Equal(t, "Phantom", ure.JiraName)
	testutil.Contains(t, err.Error(), "Phantom")
}

func TestRunList_Fields_WithJSON_Errors(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, nil)
	defer cs.server.Close()

	opts, _, _ := newOptsFor(t, cs)
	opts.Output = "json"
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "SUMMARY")
	if err == nil {
		t.Fatalf("expected error when --fields combined with --output json")
	}
	testutil.Contains(t, err.Error(), "not supported with --output json")
}

func TestRunList_FieldsWithIDOnly_IDWins(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1", "TEST-2"}, true, nil)
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "SUMMARY")
	testutil.RequireNoError(t, err)

	want := "TEST-1\nTEST-2\n"
	if stdout.String() != want {
		t.Errorf("stdout: got %q, want %q", stdout.String(), want)
	}
}

// Under --id, projection.Resolve is skipped entirely. A human-name --fields
// token would normally trigger a GetFields() call; --id must suppress it.
func TestRunList_IDOnly_SkipsFieldsResolution(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, []api.Field{
		{ID: "issuetype", Name: "Issue Type"},
	})
	defer cs.server.Close()

	opts, _, _ := newOptsFor(t, cs)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "Issue Type")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 0, cs.fieldsCalls)
}

// Under --id, even an unknown --fields token must not fail — --id bypasses
// projection entirely. Without this short-circuit, `--id --fields bogus`
// would error even though --id would have discarded the projection anyway.
func TestRunList_IDOnly_BypassesFieldsValidation(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, []api.Field{})
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "bogus")
	testutil.RequireNoError(t, err)
	if stdout.String() != "TEST-1\n" {
		t.Errorf("expected bare key, got %q", stdout.String())
	}
}

// Under --id, the JSON + --fields rejection also must not fire. --id produces
// plain identifiers, not JSON, so the conflict is moot.
func TestRunList_IDOnly_BypassesJSONFieldsRejection(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, nil)
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	opts.IDOnly = true
	opts.Output = "json"
	err := runList(context.Background(), opts, "TEST", "", 25, "", false, "SUMMARY")
	testutil.RequireNoError(t, err)
	if stdout.String() != "TEST-1\n" {
		t.Errorf("expected bare key, got %q", stdout.String())
	}
}

func TestRunList_Fields_TrumpsAllFieldsForFetch(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, nil)
	defer cs.server.Close()

	opts, _, _ := newOptsFor(t, cs)
	// Both --fields and --all-fields set; --fields must win for fetch.
	err := runList(context.Background(), opts, "TEST", "", 25, "", true, "SUMMARY")
	testutil.RequireNoError(t, err)
	got := cs.searchCaptured.Fields
	if len(got) != 1 || got[0] != "summary" {
		t.Errorf("--fields must drive fetch even when --all-fields is set; got %v", got)
	}
}

func TestRunList_AllFieldsWithoutFields_UsesDefaultSearchFields(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, nil)
	defer cs.server.Close()

	opts, _, _ := newOptsFor(t, cs)
	err := runList(context.Background(), opts, "TEST", "", 25, "", true, "")
	testutil.RequireNoError(t, err)
	got := cs.searchCaptured.Fields
	if len(got) != len(api.DefaultSearchFields) {
		t.Errorf("--all-fields should request DefaultSearchFields; got %d fields, want %d", len(got), len(api.DefaultSearchFields))
	}
}

func TestJqlEscape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain", "Sprint 70", "Sprint 70"},
		{"quote_escaped", `She said "hi"`, `She said \"hi\"`},
		{"backslash_escaped_before_quote", `C:\path`, `C:\\path`},
		// Ordering invariant: backslash is escaped BEFORE quote. If the order
		// reversed, the input `\"` would become `\\"` (unterminated) instead of
		// the correct `\\\"`.
		{"bslash_then_quote_ordering", `\"`, `\\\"`},
		{"both_chars_mixed", `a\b"c\"d`, `a\\b\"c\\\"d`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := jqlEscape(tc.in); got != tc.want {
				t.Errorf("jqlEscape(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestBuildSprintClause_WarnBranches(t *testing.T) {
	seedCacheForIssues(t)
	// Seed two boards with a sprint of the same name on both, so name resolution
	// is ambiguous.
	testutil.RequireNoError(t, seedSprints(map[int][]api.Sprint{
		11: {{ID: 100, Name: "Duplicated Sprint", State: "active"}},
		22: {{ID: 200, Name: "Duplicated Sprint", State: "closed"}},
	}))

	// Hermetic httptest server — prevents any accidental resolver refresh from
	// reaching a real host (CI outbound-blocked envs would otherwise time out).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "e", APIToken: "t"})
	testutil.RequireNoError(t, err)

	t.Run("ambiguous_emits_warning", func(t *testing.T) {
		var buf bytes.Buffer
		clause, err := buildSprintClause(context.Background(), resolve.New(client), "Duplicated Sprint", &buf)
		testutil.RequireNoError(t, err)
		if !strings.Contains(clause, `sprint = "Duplicated Sprint"`) {
			t.Fatalf("want quoted JQL fallback, got %q", clause)
		}
		if !strings.Contains(buf.String(), "matched multiple cached boards") {
			t.Errorf("want ambiguity warning, got: %q", buf.String())
		}
	})

	t.Run("unresolvable_name_always_emits_some_warning", func(t *testing.T) {
		// The resolver's NotFoundError vs "resolver failed" (network) branch
		// depends on refresh reachability — both are valid in this test setup
		// (httptest client isn't wired for a sprints refresh). The invariant
		// the reviewer asked us to pin is that the function never silently
		// falls through to a JQL quoted-name clause: whichever branch fires,
		// the user gets a diagnostic.
		var buf bytes.Buffer
		clause, err := buildSprintClause(context.Background(), resolve.New(client), "Nonexistent Sprint Name", &buf)
		testutil.RequireNoError(t, err)
		if !strings.Contains(clause, `sprint = "Nonexistent Sprint Name"`) {
			t.Fatalf("want quoted JQL fallback, got %q", clause)
		}
		if !strings.Contains(buf.String(), "warning:") {
			t.Errorf("want some warning, got silence: %q", buf.String())
		}
	})

	t.Run("negative_numeric_rejected", func(t *testing.T) {
		_, err := buildSprintClause(context.Background(), resolve.New(client), "-5", nil)
		if err == nil || !strings.Contains(err.Error(), "must be positive") {
			t.Errorf("want positive-only error, got %v", err)
		}
	})

	t.Run("zero_numeric_rejected", func(t *testing.T) {
		_, err := buildSprintClause(context.Background(), resolve.New(client), "0", nil)
		if err == nil || !strings.Contains(err.Error(), "must be positive") {
			t.Errorf("want positive-only error, got %v", err)
		}
	})
}
