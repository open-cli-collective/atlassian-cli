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
