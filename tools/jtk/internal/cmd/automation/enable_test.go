package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newAutomationTestServer(t *testing.T, rule api.AutomationRule) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(rule)
	}))
}

func TestRunSetState_AlreadyEnabled(t *testing.T) {
	t.Parallel()
	rule := api.AutomationRule{
		ID:    json.Number("42"),
		Name:  "Test Rule",
		State: "ENABLED",
	}

	server := newAutomationTestServer(t, rule)
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

	err = runSetState(context.Background(), opts, "42", true)
	testutil.RequireNoError(t, err)
	// No-change messages route to stderr (no mutation occurred)
	testutil.Contains(t, stderr.String(), "already ENABLED")
}

func TestRunSetState_AlreadyDisabled(t *testing.T) {
	t.Parallel()
	rule := api.AutomationRule{
		ID:    json.Number("42"),
		Name:  "Test Rule",
		State: "DISABLED",
	}

	server := newAutomationTestServer(t, rule)
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

	err = runSetState(context.Background(), opts, "42", false)
	testutil.RequireNoError(t, err)
	// No-change messages route to stderr (no mutation occurred)
	testutil.Contains(t, stderr.String(), "already DISABLED")
}

func TestRunSetState_EnableDisabledRule(t *testing.T) {
	t.Parallel()
	stateChanged := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}

		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodGet {
			state := "DISABLED"
			if stateChanged {
				state = "ENABLED"
			}
			rule := api.AutomationRule{
				ID:    json.Number("42"),
				Name:  "Test Rule",
				State: state,
			}
			_ = json.NewEncoder(w).Encode(rule)
			return
		}

		// PUT state
		stateChanged = true
		_, _ = w.Write([]byte(`{}`))
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

	err = runSetState(context.Background(), opts, "42", true)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "ENABLED")
	testutil.Contains(t, stdout.String(), "Test Rule")
}
