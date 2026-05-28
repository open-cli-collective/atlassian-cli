package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/prompt"
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
		Stdout: &stdout,
		Stderr: &stderr,
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "42", true)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted automation 42\n")
	testutil.Equal(t, stderr.String(), "")
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
		Stdout: &stdout,
		Stderr: &stderr,
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "42", true)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted automation 42\n")
	testutil.Equal(t, stderr.String(), "")
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
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  bytes.NewBufferString("n\n"),
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "42", false)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stderr.String(), "permanently delete")
	testutil.Equal(t, stdout.String(), "Deletion cancelled.\n")
}

func TestRunDelete_EmitsText(t *testing.T) {
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
		Stdout: &stdout,
		Stderr: &stderr,
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "42", true)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted automation 42\n")
	testutil.Equal(t, stderr.String(), "")
}

// TestRunDelete_NonInteractive_WithoutForce_ShortCircuits — §3.4
// early-fail: --non-interactive without --force returns
// ErrConfirmationRequired BEFORE the API GetAutomationRule call.
func TestRunDelete_NonInteractive_WithoutForce_ShortCircuits(t *testing.T) {
	t.Parallel()
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{
		NonInteractive: true,
		Stdout:         &stdout,
		Stderr:         &stderr,
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "42", false)
	if err == nil {
		t.Fatal("expected ErrConfirmationRequired")
	}
	if !errors.Is(err, prompt.ErrConfirmationRequired) {
		t.Fatalf("expected prompt.ErrConfirmationRequired, got %v", err)
	}
	if hits != 0 {
		t.Fatalf("API must not be hit; got %d calls", hits)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr must be empty: %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout must be empty (no PresentDeleteCancelled artifact): %q", stdout.String())
	}
}

// TestRunDelete_NonInteractive_WithForce_Proceeds — --force still
// bypasses confirmation under --non-interactive (matches the family
// pattern in issues/page/attachment/configcmd tests).
func TestRunDelete_NonInteractive_WithForce_Proceeds(t *testing.T) {
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
				Name:  "Test Rule",
				State: "DISABLED",
			}
			_ = json.NewEncoder(w).Encode(rule)
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@x.io", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{
		NonInteractive: true,
		Stdout:         &stdout,
		Stderr:         &stderr,
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "42", true)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted automation 42\n")
}
