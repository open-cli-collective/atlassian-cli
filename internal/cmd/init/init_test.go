package init

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rianjs/confluence-cli/internal/config"
)

func TestVerifyConnection_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		assert.Equal(t, "/api/v2/spaces", r.URL.Path)
		assert.Equal(t, "1", r.URL.Query().Get("limit"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		// Verify basic auth is present
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok, "basic auth should be present")
		assert.Equal(t, "test@example.com", user)
		assert.Equal(t, "test-token", pass)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "test-token",
	}

	err := verifyConnection(cfg)
	assert.NoError(t, err)
}

func TestVerifyConnection_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message": "Unauthorized"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		URL:      server.URL,
		Email:    "bad@example.com",
		APIToken: "wrong-token",
	}

	err := verifyConnection(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
	assert.Contains(t, err.Error(), "email and API token")
}

func TestVerifyConnection_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "Forbidden"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token-no-perms",
	}

	err := verifyConnection(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
	assert.Contains(t, err.Error(), "permissions")
}

func TestVerifyConnection_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Config{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "test-token",
	}

	err := verifyConnection(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 500")
}

func TestVerifyConnection_NetworkError(t *testing.T) {
	cfg := &config.Config{
		URL:      "http://localhost:99999", // Non-existent server
		Email:    "test@example.com",
		APIToken: "test-token",
	}

	err := verifyConnection(cfg)
	require.Error(t, err)
	// Should fail to connect
}

func TestVerifyConnection_StatusCodes(t *testing.T) {
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			cfg := &config.Config{
				URL:      server.URL,
				Email:    "test@example.com",
				APIToken: "test-token",
			}

			err := verifyConnection(cfg)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigFilePermissions(t *testing.T) {
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
	require.NoError(t, err)

	// Check the file permissions
	info, err := os.Stat(configPath)
	require.NoError(t, err)

	// On Unix, permissions should be 0600 (user read/write only)
	// The exact mode includes the file type bits, so we mask with 0777
	perm := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0600), perm, "config file should have 0600 permissions")
}

func TestConfigFilePermissions_DirectoryCreation(t *testing.T) {
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
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Verify directory was created
	dirInfo, err := os.Stat(filepath.Dir(configPath))
	require.NoError(t, err)
	assert.True(t, dirInfo.IsDir())
}

func TestNewCmdInit_Flags(t *testing.T) {
	cmd := NewCmdInit()

	// Verify command structure
	assert.Equal(t, "init", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Verify flags exist
	urlFlag := cmd.Flags().Lookup("url")
	require.NotNil(t, urlFlag)
	assert.Equal(t, "", urlFlag.DefValue)

	emailFlag := cmd.Flags().Lookup("email")
	require.NotNil(t, emailFlag)
	assert.Equal(t, "", emailFlag.DefValue)

	noVerifyFlag := cmd.Flags().Lookup("no-verify")
	require.NotNil(t, noVerifyFlag)
	assert.Equal(t, "false", noVerifyFlag.DefValue)
}
