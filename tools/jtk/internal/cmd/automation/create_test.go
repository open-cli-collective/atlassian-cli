package automation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunCreate(t *testing.T) {
	t.Run("successful create", func(t *testing.T) {
		var receivedBody json.RawMessage

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
				return
			}

			if r.Method == http.MethodPost {
				_ = json.NewDecoder(r.Body).Decode(&receivedBody)
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":99,"ruleKey":"new-uuid","name":"Test Rule"}`))
				return
			}

			w.WriteHeader(http.StatusMethodNotAllowed)
		}))
		defer server.Close()

		client, err := api.New(api.ClientConfig{
			URL:      server.URL,
			Email:    "test@example.com",
			APIToken: "token",
		})
		require.NoError(t, err)

		var stdout, stderr bytes.Buffer
		opts := &root.Options{
			Output: "table",
			Stdout: &stdout,
			Stderr: &stderr,
		}
		opts.SetAPIClient(client)

		// Write test JSON to temp file
		dir := t.TempDir()
		filePath := filepath.Join(dir, "rule.json")
		err = os.WriteFile(filePath, []byte(`{"name":"Test Rule","state":"DISABLED"}`), 0644)
		require.NoError(t, err)

		err = runCreate(opts, filePath)
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "Test Rule")
		assert.Contains(t, stdout.String(), "new-uuid")
		assert.JSONEq(t, `{"name":"Test Rule","state":"DISABLED"}`, string(receivedBody))
	})

	t.Run("invalid JSON file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "bad.json")
		err := os.WriteFile(filePath, []byte(`not valid json`), 0644)
		require.NoError(t, err)

		var stdout, stderr bytes.Buffer
		opts := &root.Options{
			Output: "table",
			Stdout: &stdout,
			Stderr: &stderr,
		}

		err = runCreate(opts, filePath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not contain valid JSON")
	})

	t.Run("file not found", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		opts := &root.Options{
			Output: "table",
			Stdout: &stdout,
			Stderr: &stderr,
		}

		err := runCreate(opts, "/nonexistent/path/rule.json")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read file")
	})
}
