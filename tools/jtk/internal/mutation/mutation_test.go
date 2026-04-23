package mutation

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func init() {
	BackoffSchedule = []time.Duration{0, 0, 0, 0}
}

func testOpts() (*root.Options, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return &root.Options{
		Stdout: stdout,
		Stderr: stderr,
	}, stdout, stderr
}

func successModel(msg string) *present.OutputModel {
	return &present.OutputModel{
		Sections: []present.Section{
			&present.MessageSection{
				Kind:    present.MessageSuccess,
				Message: msg,
				Stream:  present.StreamStdout,
			},
		},
	}
}

func fallbackFor(prefix string) func(string) *present.OutputModel {
	return func(id string) *present.OutputModel {
		return successModel(prefix + " " + id)
	}
}

func TestWriteAndPresent_HappyPath(t *testing.T) {
	opts, stdout, stderr := testOpts()
	err := WriteAndPresent(context.Background(), opts, Config{
		Write: func(_ context.Context) (string, error) { return "MON-100", nil },
		Fetch: func(_ context.Context, id string) (*present.OutputModel, error) {
			return successModel("Detail for " + id), nil
		},
		Fallback: fallbackFor("Created"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "Detail for MON-100\n" {
		t.Errorf("stdout = %q, want %q", got, "Detail for MON-100\n")
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestWriteAndPresent_IsFresh_RetryThenAccept(t *testing.T) {
	opts, stdout, _ := testOpts()
	calls := 0
	err := WriteAndPresent(context.Background(), opts, Config{
		Write: func(_ context.Context) (string, error) { return "MON-100", nil },
		Fetch: func(_ context.Context, _ string) (*present.OutputModel, error) {
			calls++
			return successModel("Status: " + map[bool]string{true: "In Development", false: "Backlog"}[calls >= 3]), nil
		},
		IsFresh: func(m *present.OutputModel) bool {
			return ModelContainsStatus(m, "In Development")
		},
		Fallback: fallbackFor("Transitioned"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls < 3 {
		t.Errorf("expected at least 3 fetch calls, got %d", calls)
	}
	if got := stdout.String(); got != "Status: In Development\n" {
		t.Errorf("stdout = %q", got)
	}
}

func TestWriteAndPresent_AllStale_EmitsBestAvailable(t *testing.T) {
	opts, stdout, stderr := testOpts()
	err := WriteAndPresent(context.Background(), opts, Config{
		Write: func(_ context.Context) (string, error) { return "MON-100", nil },
		Fetch: func(_ context.Context, _ string) (*present.OutputModel, error) {
			return successModel("Status: Backlog"), nil
		},
		IsFresh:  func(_ *present.OutputModel) bool { return false },
		Fallback: fallbackFor("Transitioned"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "Status: Backlog\n" {
		t.Errorf("stdout = %q, want best-available stale model", got)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be empty for best-available, got %q", stderr.String())
	}
}

func TestWriteAndPresent_AllFetchErrors_EmitsFallback(t *testing.T) {
	opts, stdout, stderr := testOpts()
	err := WriteAndPresent(context.Background(), opts, Config{
		Write: func(_ context.Context) (string, error) { return "MON-100", nil },
		Fetch: func(_ context.Context, _ string) (*present.OutputModel, error) {
			return nil, errors.New("network error")
		},
		Fallback: fallbackFor("Created"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "Created MON-100\n" {
		t.Errorf("stdout = %q", got)
	}
	if got := stderr.String(); got == "" {
		t.Error("expected advisory on stderr")
	}
}

func TestWriteAndPresent_WriteFails(t *testing.T) {
	opts, _, _ := testOpts()
	fetchCalled := false
	err := WriteAndPresent(context.Background(), opts, Config{
		Write: func(_ context.Context) (string, error) { return "", errors.New("write failed") },
		Fetch: func(_ context.Context, _ string) (*present.OutputModel, error) {
			fetchCalled = true
			return successModel("should not happen"), nil
		},
		Fallback: fallbackFor("nope"),
	})
	if err == nil || err.Error() != "write failed" {
		t.Errorf("expected write error, got %v", err)
	}
	if fetchCalled {
		t.Error("Fetch should not be called when Write fails")
	}
}

func TestWriteAndPresent_IDOnly(t *testing.T) {
	opts, stdout, _ := testOpts()
	opts.IDOnly = true
	fetchCalled := false
	err := WriteAndPresent(context.Background(), opts, Config{
		Write: func(_ context.Context) (string, error) { return "MON-100", nil },
		Fetch: func(_ context.Context, _ string) (*present.OutputModel, error) {
			fetchCalled = true
			return successModel("detail"), nil
		},
		Fallback: fallbackFor("nope"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if fetchCalled {
		t.Error("Fetch should not be called in --id mode")
	}
	if got := stdout.String(); got != "MON-100\n" {
		t.Errorf("stdout = %q, want %q", got, "MON-100\n")
	}
}

func TestWriteAndPresent_IsFreshNil_AcceptsFirstFetch(t *testing.T) {
	opts, stdout, _ := testOpts()
	calls := 0
	err := WriteAndPresent(context.Background(), opts, Config{
		Write: func(_ context.Context) (string, error) { return "MON-100", nil },
		Fetch: func(_ context.Context, _ string) (*present.OutputModel, error) {
			calls++
			return successModel("first fetch"), nil
		},
		IsFresh:  nil,
		Fallback: fallbackFor("nope"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("expected 1 fetch call (no freshness check), got %d", calls)
	}
	if got := stdout.String(); got != "first fetch\n" {
		t.Errorf("stdout = %q", got)
	}
}

func TestWriteAndPresent_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts, stdout, stderr := testOpts()
	err := WriteAndPresent(ctx, opts, Config{
		Write: func(_ context.Context) (string, error) { return "MON-100", nil },
		Fetch: func(_ context.Context, _ string) (*present.OutputModel, error) {
			return nil, context.Canceled
		},
		Fallback: fallbackFor("Created"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if stdout.Len() == 0 {
		t.Error("expected fallback output on stdout")
	}
	if stderr.Len() == 0 {
		t.Error("expected advisory on stderr")
	}
}

func TestModelContainsStatus(t *testing.T) {
	model := successModel("Status: In Development   Type: SDLC   Priority: Medium   Points: 5")
	if !ModelContainsStatus(model, "In Development") {
		t.Error("expected true for matching status")
	}
	if ModelContainsStatus(model, "Backlog") {
		t.Error("expected false for non-matching status")
	}
	if ModelContainsStatus(model, "In") {
		t.Error("expected false for prefix collision: 'In' should not match 'In Development'")
	}
}

func TestModelContainsStatus_EmptyModel(t *testing.T) {
	model := &present.OutputModel{}
	if ModelContainsStatus(model, "anything") {
		t.Error("expected false for empty model")
	}
}

func TestModelContainsField_EndOfLine(t *testing.T) {
	model := successModel("Assignee: Aaron Wong")
	if !ModelContainsField(model, "Assignee: ", "Aaron Wong") {
		t.Error("expected true for value at end of line")
	}
	if ModelContainsField(model, "Assignee: ", "Aaron") {
		t.Error("expected false for prefix collision at end of line")
	}
}
