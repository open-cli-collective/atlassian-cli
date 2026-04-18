package present

import (
	"bytes"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newTestOpts() (*root.Options, *bytes.Buffer, *bytes.Buffer) {
	var stdout, stderr bytes.Buffer
	return &root.Options{Stdout: &stdout, Stderr: &stderr}, &stdout, &stderr
}

func TestParseStartAtToken(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		want    int
		wantErr string
	}{
		{"empty", "", 0, ""},
		{"zero", "0", 0, ""},
		{"positive", "25", 25, ""},
		{"non-numeric", "abc", 0, "invalid --next-page-token"},
		{"negative", "-1", 0, "invalid --next-page-token"},
		{"float-like", "2.5", 0, "invalid --next-page-token"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseStartAtToken(tc.input)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tc.want {
					t.Errorf("got %d, want %d", got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestAppendPaginationHintWithToken_EmbedsToken(t *testing.T) {
	t.Parallel()
	sections := AppendPaginationHintWithToken(nil, true, "eyJzdGFydEF0IjoxMH0")
	if len(sections) != 1 {
		t.Fatalf("sections = %d, want 1", len(sections))
	}
	msg, ok := sections[0].(*present.MessageSection)
	if !ok {
		t.Fatalf("expected MessageSection, got %T", sections[0])
	}
	want := "More results available (next: eyJzdGFydEF0IjoxMH0)"
	if msg.Message != want {
		t.Errorf("Message = %q, want %q", msg.Message, want)
	}
	if msg.Stream != present.StreamStdout {
		t.Errorf("Stream = %v, want Stdout", msg.Stream)
	}
}

func TestAppendPaginationHintWithToken_EmptyTokenFallsBackToLegacyWording(t *testing.T) {
	t.Parallel()
	sections := AppendPaginationHintWithToken(nil, true, "")
	msg := sections[0].(*present.MessageSection)
	if msg.Message != paginationHint {
		t.Errorf("empty-token fallback = %q, want legacy wording %q", msg.Message, paginationHint)
	}
}

func TestAppendPaginationHintWithToken_NoMoreReturnsUnchanged(t *testing.T) {
	t.Parallel()
	base := []present.Section{&present.MessageSection{Kind: present.MessageInfo, Message: "only"}}
	got := AppendPaginationHintWithToken(base, false, "anything")
	if len(got) != 1 {
		t.Errorf("hasMore=false should not append, got len=%d", len(got))
	}
}

func TestEmitIDsWithPaginationToken_EmitsTokenInContinuationLine(t *testing.T) {
	t.Parallel()
	opts, stdout, _ := newTestOpts()
	err := EmitIDsWithPaginationToken(opts, []string{"MON-1", "MON-2"}, true, "25")
	if err != nil {
		t.Fatalf("EmitIDsWithPaginationToken: %v", err)
	}
	want := "MON-1\nMON-2\nMore results available (next: 25)\n"
	if stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestEmitIDsWithPaginationToken_NoMoreOmitsContinuation(t *testing.T) {
	t.Parallel()
	opts, stdout, _ := newTestOpts()
	err := EmitIDsWithPaginationToken(opts, []string{"MON-1"}, false, "unused")
	if err != nil {
		t.Fatalf("EmitIDsWithPaginationToken: %v", err)
	}
	if stdout.String() != "MON-1\n" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "MON-1\n")
	}
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

	if err := EmitIDs(opts, []string{"MON-1", "MON-2", "MON-3"}); err != nil {
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

	if err := EmitIDs(opts, nil); err != nil {
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

func TestPaginationMessageSection_Canonical(t *testing.T) {
	t.Parallel()
	// Every pagination call site funnels through this helper; drift would
	// de-sync wording, kind, or stream across the three migrated commands.
	msg := paginationMessageSection()
	if msg.Kind != present.MessageInfo {
		t.Errorf("kind: got %v, want MessageInfo", msg.Kind)
	}
	if msg.Stream != present.StreamStdout {
		t.Errorf("stream: got %v, want StreamStdout", msg.Stream)
	}
	if msg.Message != paginationHint {
		t.Errorf("message: got %q, want %q", msg.Message, paginationHint)
	}
}

func TestAppendPaginationHint(t *testing.T) {
	t.Parallel()
	base := []present.Section{
		&present.TableSection{Headers: []string{"K"}, Rows: []present.Row{{Cells: []string{"v"}}}},
	}

	same := AppendPaginationHint(base, false)
	if len(same) != 1 {
		t.Errorf("no-op when hasMore=false: got %d sections, want 1", len(same))
	}

	withHint := AppendPaginationHint(base, true)
	if len(withHint) != 2 {
		t.Fatalf("hasMore=true: got %d sections, want 2", len(withHint))
	}
	msg, ok := withHint[1].(*present.MessageSection)
	if !ok {
		t.Fatalf("second section should be *MessageSection, got %T", withHint[1])
	}
	if msg.Stream != present.StreamStdout || msg.Message != paginationHint {
		t.Errorf("appended section mismatch: stream=%v msg=%q", msg.Stream, msg.Message)
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
