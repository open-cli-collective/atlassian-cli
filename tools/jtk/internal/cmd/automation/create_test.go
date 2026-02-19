package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunCreate(t *testing.T) {
	t.Parallel()
	t.Run("strips server-assigned fields", func(t *testing.T) {
		t.Parallel()
		var receivedBody map[string]any

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
		testutil.RequireNoError(t, err)

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
		err = os.WriteFile(filePath, []byte(inputJSON), 0600)
		testutil.RequireNoError(t, err)

		err = runCreate(context.Background(), opts, filePath)
		testutil.RequireNoError(t, err)

		// Verify server-assigned fields were stripped
		testutil.Nil(t, receivedBody["uuid"])
		testutil.Nil(t, receivedBody["id"])
		testutil.Nil(t, receivedBody["ruleKey"])
		testutil.Nil(t, receivedBody["created"])
		testutil.Nil(t, receivedBody["updated"])

		// Verify non-server fields are preserved
		testutil.Equal(t, receivedBody["name"], "Test Rule")
		testutil.Equal(t, receivedBody["state"], "DISABLED")

		// Verify output shows the new UUID from response
		testutil.Contains(t, stdout.String(), "Test Rule")
		testutil.Contains(t, stdout.String(), "new-uuid-456")
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
		testutil.RequireNoError(t, err)

		var stdout, stderr bytes.Buffer
		opts := &root.Options{
			Output: "table",
			Stdout: &stdout,
			Stderr: &stderr,
		}
		opts.SetAPIClient(client)

		dir := t.TempDir()
		filePath := filepath.Join(dir, "rule.json")
		err = os.WriteFile(filePath, []byte(`{"name":"New Rule","state":"DISABLED"}`), 0600)
		testutil.RequireNoError(t, err)

		err = runCreate(context.Background(), opts, filePath)
		testutil.RequireNoError(t, err)
		testutil.Contains(t, stdout.String(), "rule-uuid-789")
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
		testutil.RequireNoError(t, err)

		var stdout, stderr bytes.Buffer
		opts := &root.Options{
			Output: "table",
			Stdout: &stdout,
			Stderr: &stderr,
		}
		opts.SetAPIClient(client)

		dir := t.TempDir()
		filePath := filepath.Join(dir, "rule.json")
		err = os.WriteFile(filePath, []byte(`{"name":"Both UUIDs","state":"DISABLED"}`), 0600)
		testutil.RequireNoError(t, err)

		err = runCreate(context.Background(), opts, filePath)
		testutil.RequireNoError(t, err)
		testutil.Contains(t, stdout.String(), "preferred-uuid")
		testutil.NotContains(t, stdout.String(), "fallback-uuid")
	})

	t.Run("invalid JSON file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "bad.json")
		err := os.WriteFile(filePath, []byte(`not valid json`), 0600)
		testutil.RequireNoError(t, err)

		var stdout, stderr bytes.Buffer
		opts := &root.Options{
			Output: "table",
			Stdout: &stdout,
			Stderr: &stderr,
		}

		err = runCreate(context.Background(), opts, filePath)
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "does not contain valid JSON")
	})

	t.Run("file not found", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		opts := &root.Options{
			Output: "table",
			Stdout: &stdout,
			Stderr: &stderr,
		}

		err := runCreate(context.Background(), opts, "/nonexistent/path/rule.json")
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "reading file")
	})
}
