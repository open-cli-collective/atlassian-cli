package initcmd

import (
	"bufio"
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
		Stdin:   strings.NewReader(""),
	}
}

func TestRunInit_NonInteractive_WithVerify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/myself")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accountId": "123", "displayName": "Test User", "emailAddress": "test@example.com"}`))
	}))
	defer server.Close()

	// Provide "y" for overwrite prompt in case existing config is detected
	opts := newTestRootOptions()
	opts.Stdin = strings.NewReader("y\n")

	err := runInit(opts, server.URL, "test@example.com", "token123", false)
	require.NoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	assert.Contains(t, stdout, "Connected to")
	assert.Contains(t, stdout, "Authenticated as")
	assert.Contains(t, stdout, "Configuration saved")
}

func TestRunInit_NonInteractive_NoVerify(t *testing.T) {
	// Provide "y" for overwrite prompt in case existing config is detected
	opts := newTestRootOptions()
	opts.Stdin = strings.NewReader("y\n")

	err := runInit(opts, "https://test.atlassian.net", "test@example.com", "token123", true)
	require.NoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	assert.Contains(t, stdout, "Configuration saved")
}

func TestRunInit_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message": "Unauthorized"}`))
	}))
	defer server.Close()

	// Provide "y" for overwrite prompt in case existing config is detected
	opts := newTestRootOptions()
	opts.Stdin = strings.NewReader("y\n")

	err := runInit(opts, server.URL, "test@example.com", "bad-token", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestPromptYesNo(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultYes bool
		want       bool
	}{
		{"yes lowercase", "y\n", false, true},
		{"yes full word", "yes\n", false, true},
		{"no lowercase", "n\n", false, false},
		{"empty with default no", "\n", false, false},
		{"empty with default yes", "\n", true, true},
		{"random input", "maybe\n", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got, err := promptYesNo(reader, "", tt.defaultYes)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPromptRequired(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid input", "hello\n", "hello", false},
		{"with leading/trailing space", "  hello  \n", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got, err := promptRequired(reader, "Test")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
