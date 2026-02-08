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
	t.Run("strips server-assigned fields", func(t *testing.T) {
		var receivedBody map[string]interface{}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
				return
			}

			if r.Method == http.MethodPost {
				_ = json.NewDecoder(r.Body).Decode(&receivedBody)
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":99,"ruleUuid":"new-uuid-456","name":"Test Rule"}`))
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

		// Write test JSON with server-assigned fields that should be stripped
		dir := t.TempDir()
		filePath := filepath.Join(dir, "rule.json")
		inputJSON := `{
			"uuid": "existing-uuid",
			"id": 42,
			"ruleKey": "old-rule-key",
			"created": "2024-01-01T00:00:00Z",
			"updated": "2024-06-01T00:00:00Z",
			"name": "Test Rule",
			"state": "DISABLED"
		}`
		err = os.WriteFile(filePath, []byte(inputJSON), 0644)
		require.NoError(t, err)

		err = runCreate(opts, filePath)
		require.NoError(t, err)

		// Verify server-assigned fields were stripped
		assert.Nil(t, receivedBody["uuid"], "uuid should be stripped")
		assert.Nil(t, receivedBody["id"], "id should be stripped")
		assert.Nil(t, receivedBody["ruleKey"], "ruleKey should be stripped")
		assert.Nil(t, receivedBody["created"], "created should be stripped")
		assert.Nil(t, receivedBody["updated"], "updated should be stripped")

		// Verify non-server fields are preserved
		assert.Equal(t, "Test Rule", receivedBody["name"])
		assert.Equal(t, "DISABLED", receivedBody["state"])

		// Verify output shows the new UUID from response
		assert.Contains(t, stdout.String(), "Test Rule")
		assert.Contains(t, stdout.String(), "new-uuid-456")
	})

	t.Run("response with ruleUuid field", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
				return
			}

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"ruleUuid":"rule-uuid-789","name":"New Rule"}`))
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

		dir := t.TempDir()
		filePath := filepath.Join(dir, "rule.json")
		err = os.WriteFile(filePath, []byte(`{"name":"New Rule","state":"DISABLED"}`), 0644)
		require.NoError(t, err)

		err = runCreate(opts, filePath)
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "rule-uuid-789")
	})

	t.Run("response prefers uuid over ruleUuid", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
				return
			}

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"uuid":"preferred-uuid","ruleUuid":"fallback-uuid","name":"Both UUIDs"}`))
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

		dir := t.TempDir()
		filePath := filepath.Join(dir, "rule.json")
		err = os.WriteFile(filePath, []byte(`{"name":"Both UUIDs","state":"DISABLED"}`), 0644)
		require.NoError(t, err)

		err = runCreate(opts, filePath)
		require.NoError(t, err)
		assert.Contains(t, stdout.String(), "preferred-uuid")
		assert.NotContains(t, stdout.String(), "fallback-uuid")
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
