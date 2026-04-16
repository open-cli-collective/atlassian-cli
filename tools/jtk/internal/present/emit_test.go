package present

import (
	"bytes"
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newTestOpts() (*root.Options, *bytes.Buffer, *bytes.Buffer) {
	var stdout, stderr bytes.Buffer
	return &root.Options{Stdout: &stdout, Stderr: &stderr}, &stdout, &stderr
}

func TestEmit_SplitsStreams(t *testing.T) {
	t.Parallel()
	opts, stdout, stderr := newTestOpts()

	model := &present.OutputModel{
		Sections: []present.Section{
			&present.DetailSection{Fields: []present.Field{{Label: "ID", Value: "1"}}},
			&present.MessageSection{Kind: present.MessageInfo, Message: "diag", Stream: present.StreamStderr},
		},
	}

	if err := Emit(opts, model); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	wantStdout := "ID: 1\n"
	wantStderr := "diag\n"
	if stdout.String() != wantStdout {
		t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout.String(), wantStdout)
	}
	if stderr.String() != wantStderr {
		t.Errorf("stderr:\ngot:  %q\nwant: %q", stderr.String(), wantStderr)
	}
}

func TestEmitIDs_OnePerLine(t *testing.T) {
	t.Parallel()
	opts, stdout, stderr := newTestOpts()

	if err := EmitIDs(opts, "MON-1", "MON-2", "MON-3"); err != nil {
		t.Fatalf("EmitIDs returned error: %v", err)
	}

	want := "MON-1\nMON-2\nMON-3\n"
	if stdout.String() != want {
		t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestEmitIDs_EmptyEmitsNothing(t *testing.T) {
	t.Parallel()
	opts, stdout, stderr := newTestOpts()

	if err := EmitIDs(opts); err != nil {
		t.Fatalf("EmitIDs returned error: %v", err)
	}

	if stdout.String() != "" {
		t.Errorf("stdout should be empty, got: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestEmitIDsWithPagination_HasMoreAppendsContinuation(t *testing.T) {
	t.Parallel()
	opts, stdout, stderr := newTestOpts()

	if err := EmitIDsWithPagination(opts, []string{"MON-1", "MON-2"}, true); err != nil {
		t.Fatalf("EmitIDsWithPagination returned error: %v", err)
	}

	want := "MON-1\nMON-2\nMore results available (use --next-page-token to fetch next page)\n"
	if stdout.String() != want {
		t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestEmitIDsWithPagination_NoMoreOmitsContinuation(t *testing.T) {
	t.Parallel()
	opts, stdout, _ := newTestOpts()

	if err := EmitIDsWithPagination(opts, []string{"MON-1"}, false); err != nil {
		t.Fatalf("EmitIDsWithPagination returned error: %v", err)
	}

	want := "MON-1\n"
	if stdout.String() != want {
		t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestEmitIDsWithPagination_EmptyAndNoMore(t *testing.T) {
	t.Parallel()
	opts, stdout, stderr := newTestOpts()

	if err := EmitIDsWithPagination(opts, nil, false); err != nil {
		t.Fatalf("EmitIDsWithPagination returned error: %v", err)
	}

	if stdout.String() != "" {
		t.Errorf("stdout should be empty, got: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("stderr should be empty, got: %q", stderr.String())
	}
}

func TestEmitIDsWithPagination_EmptyButHasMore(t *testing.T) {
	t.Parallel()
	// Edge case: zero results on this page but more pages exist. Emit only
	// the continuation line so the caller can keep paging.
	opts, stdout, _ := newTestOpts()

	if err := EmitIDsWithPagination(opts, nil, true); err != nil {
		t.Fatalf("EmitIDsWithPagination returned error: %v", err)
	}

	want := "More results available (use --next-page-token to fetch next page)\n"
	if stdout.String() != want {
		t.Errorf("stdout:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}
