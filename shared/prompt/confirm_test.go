package prompt

import (
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
