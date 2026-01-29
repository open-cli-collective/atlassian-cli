package prompt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirm(t *testing.T) {
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
			got, err := Confirm(strings.NewReader(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfirmOrForce(t *testing.T) {
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
			got, err := ConfirmOrForce(tt.force, strings.NewReader(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
