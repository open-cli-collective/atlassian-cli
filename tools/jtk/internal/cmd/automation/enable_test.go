package automation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &stderr,
	}
	opts.SetAPIClient(client)

	err = runSetState(opts, "42", true)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "already ENABLED")
}

func TestRunSetState_AlreadyDisabled(t *testing.T) {
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
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &stderr,
	}
	opts.SetAPIClient(client)

	err = runSetState(opts, "42", false)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "already DISABLED")
}

func TestRunSetState_EnableDisabledRule(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}

		requestCount++
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodGet {
			rule := api.AutomationRule{
				ID:    json.Number("42"),
				Name:  "Test Rule",
				State: "DISABLED",
			}
			_ = json.NewEncoder(w).Encode(rule)
			return
		}

		// PUT state
		_, _ = w.Write([]byte(`{}`))
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

	err = runSetState(opts, "42", true)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "DISABLED")
	assert.Contains(t, stdout.String(), "ENABLED")
	assert.Equal(t, 2, requestCount) // GET + PUT
}
