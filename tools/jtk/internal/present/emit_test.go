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
	if msg.Stream != present.StreamStderr {
		t.Errorf("Stream = %v, want Stderr", msg.Stream)
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
	opts, stdout, stderr := newTestOpts()
	err := EmitIDsWithPaginationToken(opts, []string{"MON-1", "MON-2"}, true, "25")
	if err != nil {
		t.Fatalf("EmitIDsWithPaginationToken: %v", err)
	}
	if stdout.String() != "MON-1\nMON-2\n" {
		t.Errorf("stdout = %q", stdout.String())
	}
	if stderr.String() != "More results available (next: 25)\n" {
		t.Errorf("stderr = %q", stderr.String())
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

func TestPaginationMessageSectionWithToken_Canonical(t *testing.T) {
	t.Parallel()
	msg := paginationMessageSectionWithToken("token")
	if msg.Kind != present.MessageInfo {
		t.Errorf("kind: got %v, want MessageInfo", msg.Kind)
	}
	if msg.Stream != present.StreamStderr {
		t.Errorf("stream: got %v, want StreamStderr", msg.Stream)
	}
	if msg.Message != "More results available (next: token)" {
		t.Errorf("message: got %q", msg.Message)
	}
}

func TestPaginationOnlyModel(t *testing.T) {
	t.Parallel()
	model := PaginationOnlyModel("tok123")
	if len(model.Sections) != 1 {
		t.Fatalf("want 1 section, got %d", len(model.Sections))
	}
	msg, ok := model.Sections[0].(*present.MessageSection)
	if !ok {
		t.Fatalf("want *MessageSection, got %T", model.Sections[0])
	}
	if msg.Stream != present.StreamStderr {
		t.Errorf("want StreamStderr, got %v", msg.Stream)
	}
	if !strings.Contains(msg.Message, "tok123") {
		t.Errorf("want token in message, got %q", msg.Message)
	}
}

func TestValidateMax(t *testing.T) {
	t.Parallel()
	for _, maxResults := range []int{0, -1} {
		if err := ValidateMax(maxResults); err == nil {
			t.Errorf("ValidateMax(%d) = nil", maxResults)
		}
	}
	if err := ValidateMax(1); err != nil {
		t.Errorf("ValidateMax(1) = %v", err)
	}
}

func TestValidateMaxAtMost(t *testing.T) {
	t.Parallel()
	for _, maxResults := range []int{0, 101} {
		if err := ValidateMaxAtMost(maxResults, 100); err == nil {
			t.Errorf("ValidateMaxAtMost(%d, 100) = nil", maxResults)
		}
	}
	if err := ValidateMaxAtMost(100, 100); err != nil {
		t.Errorf("ValidateMaxAtMost(100, 100) = %v", err)
	}
}
