package configcmd

import (
	"bytes"
	"testing"

	"github.com/open-cli-collective/atlassian-go/credtest"
	"github.com/open-cli-collective/atlassian-go/keyring"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// config show must report token PRESENCE + keyring metadata and never
// the token value (or any slice of it), even with a token configured.
func TestRunShow_TokenPresenceNoLeak(t *testing.T) {
	credtest.Hermetic(t)
	credtest.SeedToken(t, "SUPER-SECRET-show-token")

	out, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	opts := &root.Options{Output: "table", NoColor: true, Stdout: out, Stderr: errBuf}
	testutil.RequireNoError(t, runShow(opts))

	combined := out.String() + errBuf.String()
	testutil.NotContains(t, combined, "SUPER-SECRET-show-token")
	testutil.NotContains(t, combined, "SUPER") // no prefix slice either
	testutil.Contains(t, combined, "configured")
	testutil.Contains(t, combined, "Keyring Ref")
	testutil.Contains(t, combined, keyring.Ref)
}

func TestGetValueAndSource(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			gotValue, gotSource := getValueAndSource(tt.envValue, tt.fileValue, tt.envVarName)
			testutil.Equal(t, tt.wantValue, gotValue)
			testutil.Equal(t, tt.wantSource, gotSource)
		})
	}
}

func TestFormatValueWithSource(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			got := formatValueWithSource(tt.value, tt.source)
			testutil.Equal(t, tt.want, got)
		})
	}
}
