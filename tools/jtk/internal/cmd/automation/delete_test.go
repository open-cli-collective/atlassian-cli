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

func TestRunDelete_DisabledRule(t *testing.T) {
	t.Parallel()

	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}

		methods = append(methods, r.Method)
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodGet {
			rule := api.AutomationRule{
				ID:    json.Number("42"),
				Name:  "Test Rule",
				State: "DISABLED",
			}
			_ = json.NewEncoder(w).Encode(rule)
		}
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

	err = runDelete(context.Background(), opts, "42", true)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Deleted")
	testutil.Contains(t, stdout.String(), "Test Rule")
	// Should be GET + DELETE (no disable needed)
	testutil.Len(t, methods, 2)
	testutil.Equal(t, methods[0], http.MethodGet)
	testutil.Equal(t, methods[1], http.MethodDelete)
}

func TestRunDelete_EnabledRule_DisablesFirst(t *testing.T) {
	t.Parallel()

	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}

		methods = append(methods, r.Method)
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodGet {
			rule := api.AutomationRule{
				ID:    json.Number("42"),
				Name:  "Enabled Rule",
				State: "ENABLED",
			}
			_ = json.NewEncoder(w).Encode(rule)
		}
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

	err = runDelete(context.Background(), opts, "42", true)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Deleted")
	// Should be GET + PUT (disable) + DELETE
	testutil.Len(t, methods, 3)
	testutil.Equal(t, methods[0], http.MethodGet)
	testutil.Equal(t, methods[1], http.MethodPut)
	testutil.Equal(t, methods[2], http.MethodDelete)
}

func TestRunDelete_PromptDeclined(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			rule := api.AutomationRule{
				ID:    json.Number("42"),
				Name:  "Do Not Delete",
				State: "DISABLED",
			}
			_ = json.NewEncoder(w).Encode(rule)
		}
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
		Stdin:  bytes.NewBufferString("n\n"),
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "42", false)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stderr.String(), "permanently delete")
	testutil.Contains(t, stdout.String(), "cancelled")
}

func TestRunDelete_JSONOutput(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_edge/tenant_info" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"cloudId":"test-cloud"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			rule := api.AutomationRule{
				ID:    json.Number("42"),
				Name:  "JSON Rule",
				State: "DISABLED",
			}
			_ = json.NewEncoder(w).Encode(rule)
		}
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
		Output: "json",
		Stdout: &stdout,
		Stderr: &stderr,
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "42", true)
	testutil.RequireNoError(t, err)

	var result map[string]string
	testutil.RequireNoError(t, json.Unmarshal(stdout.Bytes(), &result))
	testutil.Equal(t, result["status"], "deleted")
	testutil.Equal(t, result["name"], "JSON Rule")
}
