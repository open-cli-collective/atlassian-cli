package init

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

func TestVerifyConnection_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		testutil.Equal(t, "/api/v2/spaces", r.URL.Path)
		testutil.Equal(t, "1", r.URL.Query().Get("limit"))
		testutil.Equal(t, "application/json", r.Header.Get("Accept"))

		// Verify basic auth is present
		user, pass, ok := r.BasicAuth()
		testutil.True(t, ok, "basic auth should be present")
		testutil.Equal(t, "test@example.com", user)
		testutil.Equal(t, "test-token", pass)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "test-token",
	}

	err := verifyConnection(context.Background(), cfg)
	testutil.NoError(t, err)
}

func TestVerifyConnection_Unauthorized(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message": "Unauthorized"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		URL:      server.URL,
		Email:    "bad@example.com",
		APIToken: "wrong-token",
	}

	err := verifyConnection(context.Background(), cfg)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "authentication failed")
	testutil.Contains(t, err.Error(), "email and API token")
}

func TestVerifyConnection_Forbidden(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message": "Forbidden"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token-no-perms",
	}

	err := verifyConnection(context.Background(), cfg)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "access denied")
	testutil.Contains(t, err.Error(), "permissions")
}

func TestVerifyConnection_ServerError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Config{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "test-token",
	}

	err := verifyConnection(context.Background(), cfg)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "unexpected status code: 500")
}

func TestVerifyConnection_NetworkError(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		URL:      "http://localhost:99999", // Non-existent server
		Email:    "test@example.com",
		APIToken: "test-token",
	}

	err := verifyConnection(context.Background(), cfg)
	testutil.RequireError(t, err)
	// Should fail to connect
}

func TestVerifyConnection_StatusCodes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		errContain string
	}{
		{
			name:       "200 OK",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
			errContain: "authentication failed",
		},
		{
			name:       "403 Forbidden",
			statusCode: http.StatusForbidden,
			wantErr:    true,
			errContain: "access denied",
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			wantErr:    true,
			errContain: "unexpected status code: 404",
		},
		{
			name:       "502 Bad Gateway",
			statusCode: http.StatusBadGateway,
			wantErr:    true,
			errContain: "unexpected status code: 502",
		},
		{
			name:       "503 Service Unavailable",
			statusCode: http.StatusServiceUnavailable,
			wantErr:    true,
			errContain: "unexpected status code: 503",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			cfg := &config.Config{
				URL:      server.URL,
				Email:    "test@example.com",
				APIToken: "test-token",
			}

			err := verifyConnection(context.Background(), cfg)
			if tt.wantErr {
				testutil.RequireError(t, err)
				testutil.Contains(t, err.Error(), tt.errContain)
			} else {
				testutil.NoError(t, err)
			}
		})
	}
}

func TestConfigFilePermissions(t *testing.T) {
	t.Parallel()
	// Create a temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	cfg := config.Config{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "secret-token",
	}

	// Save the config
	err := cfg.Save(configPath)
	testutil.RequireNoError(t, err)

	// Check the file permissions
	info, err := os.Stat(configPath)
	testutil.RequireNoError(t, err)

	// On Unix, permissions should be 0600 (user read/write only)
	// The exact mode includes the file type bits, so we mask with 0777
	perm := info.Mode().Perm()
	testutil.Equal(t, perm, os.FileMode(0600))
}

func TestConfigFilePermissions_DirectoryCreation(t *testing.T) {
	t.Parallel()
	// Create a temp directory with nested path
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "deeply", "config.yml")

	cfg := config.Config{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "secret-token",
	}

	// Save should create the directory structure
	err := cfg.Save(configPath)
	testutil.RequireNoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	testutil.RequireNoError(t, err)

	// Verify directory was created
	dirInfo, err := os.Stat(filepath.Dir(configPath))
	testutil.RequireNoError(t, err)
	testutil.True(t, dirInfo.IsDir())
}

func TestInitCommand_Flags(t *testing.T) {
	t.Parallel()
	// Create root command with init registered
	rootCmd := &cobra.Command{
		Use:   "cfl",
		Short: "Test CLI",
	}

	opts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	Register(rootCmd, opts)

	// Find the init command
	initCmd, _, err := rootCmd.Find([]string{"init"})
	testutil.RequireNoError(t, err)

	// Verify command structure
	testutil.Equal(t, "init", initCmd.Use)
	testutil.NotEmpty(t, initCmd.Short)
	testutil.NotEmpty(t, initCmd.Long)

	// Verify flags exist
	urlFlag := initCmd.Flags().Lookup("url")
	testutil.NotNil(t, urlFlag)
	testutil.Equal(t, "", urlFlag.DefValue)

	emailFlag := initCmd.Flags().Lookup("email")
	testutil.NotNil(t, emailFlag)
	testutil.Equal(t, "", emailFlag.DefValue)

	noVerifyFlag := initCmd.Flags().Lookup("no-verify")
	testutil.NotNil(t, noVerifyFlag)
	testutil.Equal(t, "false", noVerifyFlag.DefValue)
}
