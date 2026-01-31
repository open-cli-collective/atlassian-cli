package configcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{
			name:  "normal token",
			token: "abcd1234567890wxyz",
			want:  "abcd********wxyz",
		},
		{
			name:  "short token",
			token: "abc",
			want:  "********",
		},
		{
			name:  "exactly 8 chars",
			token: "12345678",
			want:  "********",
		},
		{
			name:  "9 chars",
			token: "123456789",
			want:  "1234********6789",
		},
		{
			name:  "empty token",
			token: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskToken(tt.token)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetValueAndSource(t *testing.T) {
	tests := []struct {
		name       string
		envValue   string
		fileValue  string
		envVarName string
		wantValue  string
		wantSource string
	}{
		{
			name:       "env takes precedence",
			envValue:   "from-env",
			fileValue:  "from-file",
			envVarName: "CFL_URL",
			wantValue:  "from-env",
			wantSource: "CFL_URL",
		},
		{
			name:       "file used when env empty",
			envValue:   "",
			fileValue:  "from-file",
			envVarName: "CFL_URL",
			wantValue:  "from-file",
			wantSource: "config",
		},
		{
			name:       "not set when both empty",
			envValue:   "",
			fileValue:  "",
			envVarName: "CFL_URL",
			wantValue:  "",
			wantSource: "not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotSource := getValueAndSource(tt.envValue, tt.fileValue, tt.envVarName)
			assert.Equal(t, tt.wantValue, gotValue)
			assert.Equal(t, tt.wantSource, gotSource)
		})
	}
}

func TestFormatValueWithSource(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		source string
		want   string
	}{
		{
			name:   "value with source",
			value:  "https://example.com",
			source: "config",
			want:   "https://example.com  (source: config)",
		},
		{
			name:   "empty value",
			value:  "",
			source: "not set",
			want:   "(source: not set)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValueWithSource(tt.value, tt.source)
			assert.Equal(t, tt.want, got)
		})
	}
}
