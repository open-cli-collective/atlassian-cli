package issues

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// Search and List share projection infrastructure and must stay in lockstep.
// These tests exercise --fields semantics through runSearch so drift between
// the two commands surfaces immediately.

func TestRunSearch_Fields_HeaderAliases_ProjectsTable(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, nil)
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	err := runSearch(context.Background(), opts, "project = TEST", 25, "", false, "SUMMARY,STATUS")
	testutil.RequireNoError(t, err)

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if lines[0] != "KEY | SUMMARY | STATUS" {
		t.Errorf("header mismatch: got %q", lines[0])
	}
	if cs.fieldsCalls != 0 {
		t.Errorf("header aliases must not trigger GetFields; got %d calls", cs.fieldsCalls)
	}
}

func TestRunSearch_Fields_HumanName_TriggersFieldsFetch(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, []api.Field{
		{ID: "issuetype", Name: "Issue Type"},
	})
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	err := runSearch(context.Background(), opts, "project = TEST", 25, "", false, "Issue Type")
	testutil.RequireNoError(t, err)

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if lines[0] != "KEY | TYPE" {
		t.Errorf("header mismatch: got %q", lines[0])
	}
	if cs.fieldsCalls != 1 {
		t.Errorf("human-name resolution must trigger GetFields exactly once; got %d", cs.fieldsCalls)
	}
}

func TestRunSearch_Fields_UnknownToken_Errors(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, []api.Field{})
	defer cs.server.Close()

	opts, _, _ := newOptsFor(t, cs)
	err := runSearch(context.Background(), opts, "project = TEST", 25, "", false, "bogus")
	var ufe *projection.UnknownFieldError
	if !errors.As(err, &ufe) {
		t.Fatalf("expected UnknownFieldError, got %v", err)
	}
}

func TestRunSearch_Fields_WithJSON_Errors(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, nil)
	defer cs.server.Close()

	opts, _, _ := newOptsFor(t, cs)
	opts.Output = "json"
	err := runSearch(context.Background(), opts, "project = TEST", 25, "", false, "SUMMARY")
	if err == nil {
		t.Fatalf("expected error when --fields combined with --output json")
	}
	testutil.Contains(t, err.Error(), "not supported with --output json")
}

// Search and List share deriveFetchFields, but assert here directly so a
// future divergence in search's fetch wiring does not hide behind list's
// coverage.
func TestRunSearch_Fields_DerivesFetchSet(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, nil)
	defer cs.server.Close()

	opts, _, _ := newOptsFor(t, cs)
	err := runSearch(context.Background(), opts, "project = TEST", 25, "", false, "SUMMARY,STATUS")
	testutil.RequireNoError(t, err)

	got := cs.searchCaptured.Fields
	// KEY has an empty FieldID, so it never appears in the derived set.
	want := map[string]bool{"summary": true, "status": true}
	if len(got) != len(want) {
		t.Fatalf("fetch set length: got %v, want keys %v", got, want)
	}
	for _, f := range got {
		if !want[f] {
			t.Errorf("unexpected fetch field %q (want only SUMMARY + STATUS IDs)", f)
		}
	}
}

// --id in search is new behavior in this PR. Cover the pagination branch
// so the idOnly emit path is exercised with hasMore=true.
func TestRunSearch_FieldsWithIDOnly_Pagination(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1", "TEST-2"}, false, nil) // isLast=false → hasMore=true
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	opts.IDOnly = true
	err := runSearch(context.Background(), opts, "project = TEST", 25, "", false, "SUMMARY")
	testutil.RequireNoError(t, err)

	// Bare keys plus a pagination hint on stderr; stdout stays parse-friendly.
	got := stdout.String()
	testutil.Contains(t, got, "TEST-1\n")
	testutil.Contains(t, got, "TEST-2\n")
}

// Empty result set under --id must not emit anything on stdout (no header,
// no "No issues found" noise — --id is a machine-friendly mode).
func TestRunSearch_FieldsWithIDOnly_EmptyResults(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{}, true, nil)
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	opts.IDOnly = true
	err := runSearch(context.Background(), opts, "project = TEST", 25, "", false, "SUMMARY")
	testutil.RequireNoError(t, err)
	if stdout.String() != "" {
		t.Errorf("expected empty stdout for --id with no results, got %q", stdout.String())
	}
}

func TestRunSearch_FieldsWithIDOnly_IDWins(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1", "TEST-2"}, true, nil)
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	opts.IDOnly = true
	err := runSearch(context.Background(), opts, "project = TEST", 25, "", false, "SUMMARY")
	testutil.RequireNoError(t, err)

	want := "TEST-1\nTEST-2\n"
	if stdout.String() != want {
		t.Errorf("stdout: got %q, want %q", stdout.String(), want)
	}
}

func TestRunSearch_IDOnly_SkipsFieldsResolution(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, []api.Field{
		{ID: "issuetype", Name: "Issue Type"},
	})
	defer cs.server.Close()

	opts, _, _ := newOptsFor(t, cs)
	opts.IDOnly = true
	err := runSearch(context.Background(), opts, "project = TEST", 25, "", false, "Issue Type")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, 0, cs.fieldsCalls)
}

func TestRunSearch_IDOnly_BypassesFieldsValidation(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, []api.Field{})
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	opts.IDOnly = true
	err := runSearch(context.Background(), opts, "project = TEST", 25, "", false, "bogus")
	testutil.RequireNoError(t, err)
	if stdout.String() != "TEST-1\n" {
		t.Errorf("expected bare key, got %q", stdout.String())
	}
}

func TestRunSearch_IDOnly_BypassesJSONFieldsRejection(t *testing.T) {
	t.Parallel()
	cs := newCapturingServer(t, []string{"TEST-1"}, true, nil)
	defer cs.server.Close()

	opts, stdout, _ := newOptsFor(t, cs)
	opts.IDOnly = true
	opts.Output = "json"
	err := runSearch(context.Background(), opts, "project = TEST", 25, "", false, "SUMMARY")
	testutil.RequireNoError(t, err)
	if stdout.String() != "TEST-1\n" {
		t.Errorf("expected bare key, got %q", stdout.String())
	}
}
