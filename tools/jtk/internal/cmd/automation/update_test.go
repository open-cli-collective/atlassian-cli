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

func TestNewUpdateCmd_FileFlagShorthand(t *testing.T) {
	t.Parallel()
	cmd := newUpdateCmd(&root.Options{})

	fileFlag := cmd.Flags().Lookup("file")
	testutil.NotNil(t, fileFlag)
	testutil.Equal(t, fileFlag.Shorthand, "F")

	testutil.Nil(t, cmd.Flags().ShorthandLookup("f"))
	if err := cmd.ParseFlags([]string{"-f", "rule.json"}); err == nil {
		t.Fatalf("expected error parsing legacy -f shorthand, got nil")
	}
}

func TestRunUpdate(t *testing.T) {
	t.Parallel()
	t.Run("invalid JSON file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		filePath := filepath.Join(dir, "bad.json")
		err := os.WriteFile(filePath, []byte(`not valid json`), 0600)
		testutil.RequireNoError(t, err)

		var stdout, stderr bytes.Buffer
		opts := &root.Options{
			Stdout: &stdout,
			Stderr: &stderr,
		}

		err = runUpdate(context.Background(), opts, "12345", filePath)
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "does not contain valid JSON")
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		opts := &root.Options{
			Stdout: &stdout,
			Stderr: &stderr,
		}

		err := runUpdate(context.Background(), opts, "12345", "/nonexistent/path/rule.json")
		testutil.RequireError(t, err)
		testutil.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("emits eventual-consistency advisory to stderr on success", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
				return
			}
			switch r.Method {
			case http.MethodPut:
				_, _ = w.Write([]byte(`{"ruleUuid":"12345"}`))
			case http.MethodGet:
				_ = json.NewEncoder(w).Encode(struct {
					Rule api.AutomationRule `json:"rule"`
				}{Rule: api.AutomationRule{UUID: "12345", Name: "Test Rule", State: "ENABLED"}})
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}))
		defer server.Close()

		client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@example.com", APIToken: "token"})
		testutil.RequireNoError(t, err)

		dir := t.TempDir()
		filePath := filepath.Join(dir, "rule.json")
		testutil.RequireNoError(t, os.WriteFile(filePath, []byte(`{"rule":{"name":"Test Rule"}}`), 0600))

		var stdout, stderr bytes.Buffer
		opts := &root.Options{Stdout: &stdout, Stderr: &stderr}
		opts.SetAPIClient(client)

		err = runUpdate(context.Background(), opts, "12345", filePath)
		testutil.RequireNoError(t, err)
		testutil.Contains(t, stderr.String(), "eventually consistent")
		testutil.Contains(t, stderr.String(), "Re-read after a moment")
	})

	t.Run("no advisory in id-only mode", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/_edge/tenant_info" {
				_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
				return
			}
			_, _ = w.Write([]byte(`{"ruleUuid":"12345"}`))
		}))
		defer server.Close()

		client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@example.com", APIToken: "token"})
		testutil.RequireNoError(t, err)

		dir := t.TempDir()
		filePath := filepath.Join(dir, "rule.json")
		testutil.RequireNoError(t, os.WriteFile(filePath, []byte(`{"rule":{"name":"Test Rule"}}`), 0600))

		var stdout, stderr bytes.Buffer
		opts := &root.Options{Stdout: &stdout, Stderr: &stderr, IDOnly: true}
		opts.SetAPIClient(client)

		err = runUpdate(context.Background(), opts, "12345", filePath)
		testutil.RequireNoError(t, err)
		testutil.Equal(t, stderr.String(), "")
	})
}
