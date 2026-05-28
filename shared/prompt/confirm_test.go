package prompt

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestConfirm(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{
			name:  "lowercase y confirms",
			input: "y\n",
			want:  true,
		},
		{
			name:  "uppercase Y confirms",
			input: "Y\n",
			want:  true,
		},
		{
			name:  "yes does not confirm (only y)",
			input: "yes\n",
			want:  false,
		},
		{
			name:  "n does not confirm",
			input: "n\n",
			want:  false,
		},
		{
			name:  "empty input does not confirm",
			input: "\n",
			want:  false,
		},
		{
			name:  "whitespace around y confirms",
			input: "  y  \n",
			want:  true,
		},
		{
			name:  "EOF without input does not confirm",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Confirm(strings.NewReader(tt.input))
			if tt.wantErr {
				testutil.RequireError(t, err)
				return
			}
			testutil.RequireNoError(t, err)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func TestConfirmOrForce(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		force   bool
		input   string
		want    bool
		wantErr bool
	}{
		{
			name:  "force bypasses confirmation",
			force: true,
			input: "", // Not read when force is true
			want:  true,
		},
		{
			name:  "without force, y confirms",
			force: false,
			input: "y\n",
			want:  true,
		},
		{
			name:  "without force, n does not confirm",
			force: false,
			input: "n\n",
			want:  false,
		},
		{
			name:  "without force, empty does not confirm",
			force: false,
			input: "\n",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ConfirmOrForce(tt.force, strings.NewReader(tt.input))
			if tt.wantErr {
				testutil.RequireError(t, err)
				return
			}
			testutil.RequireNoError(t, err)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func TestConfirmOrFail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		force          bool
		nonInteractive bool
		input          string
		want           bool
		wantErr        error
	}{
		{
			name:           "force wins over non-interactive",
			force:          true,
			nonInteractive: true,
			want:           true,
		},
		{
			name:           "non-interactive without force surfaces sentinel",
			nonInteractive: true,
			wantErr:        ErrConfirmationRequired,
		},
		{
			name:  "interactive y confirms",
			input: "y\n",
			want:  true,
		},
		{
			name:  "interactive n cancels",
			input: "n\n",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ConfirmOrFail(tt.force, tt.nonInteractive, strings.NewReader(tt.input))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want %v, got %v", tt.wantErr, err)
				}
				return
			}
			testutil.RequireNoError(t, err)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func TestWantPrompt(t *testing.T) {
	t.Parallel()

	// Non-TTY reader (strings.Reader isn't *os.File so isTerminal returns false).
	nonTTY := strings.NewReader("")

	if WantPrompt(true, nonTTY) {
		t.Fatal("nonInteractive=true must force WantPrompt=false")
	}
	if WantPrompt(false, nonTTY) {
		t.Fatal("non-TTY stdin must yield WantPrompt=false")
	}

	// Build a real TTY-like *os.File via os.Pipe(); the read end isn't a
	// terminal so isTerminal returns false. We can't easily fabricate a
	// real TTY in a unit test, so assert the false-case from a real *os.File
	// to cover that branch of the type assertion.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer func() { _ = r.Close(); _ = w.Close() }()
	if WantPrompt(false, r) {
		t.Fatal("pipe read end is not a TTY; WantPrompt must be false")
	}
}
