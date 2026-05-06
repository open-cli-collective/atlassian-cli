package me

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

func newTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func userServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/wiki/rest/api/user/current", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
}

func TestRun_Default(t *testing.T) {
	t.Parallel()
	server := userServer(t, `{"accountId":"abc123","displayName":"Rian Stockbower","email":"rian@example.com"}`)
	defer server.Close()

	opts := newTestRootOptions()
	opts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	err := Run(context.Background(), opts, false)
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Equal(t, "abc123 | Rian Stockbower | rian@example.com\n", stdout)
}

func TestRun_IDOnly(t *testing.T) {
	t.Parallel()
	server := userServer(t, `{"accountId":"abc123","displayName":"Rian Stockbower","email":"rian@example.com"}`)
	defer server.Close()

	opts := newTestRootOptions()
	opts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	err := Run(context.Background(), opts, true)
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Equal(t, "abc123\n", stdout)
}

func TestRun_MissingDisplayName(t *testing.T) {
	t.Parallel()
	server := userServer(t, `{"accountId":"abc123","email":"rian@example.com"}`)
	defer server.Close()

	opts := newTestRootOptions()
	opts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	err := Run(context.Background(), opts, false)
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Equal(t, "abc123 | - | rian@example.com\n", stdout)
}

func TestRun_MissingEmail(t *testing.T) {
	t.Parallel()
	server := userServer(t, `{"accountId":"abc123","displayName":"Rian Stockbower"}`)
	defer server.Close()

	opts := newTestRootOptions()
	opts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	err := Run(context.Background(), opts, false)
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Equal(t, "abc123 | Rian Stockbower | -\n", stdout)
}

func TestRun_JSONOutputFallsThroughToOneLiner(t *testing.T) {
	t.Parallel()
	server := userServer(t, `{"accountId":"abc123","displayName":"Rian Stockbower","email":"rian@example.com"}`)
	defer server.Close()

	opts := newTestRootOptions()
	opts.Output = "json"
	opts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	err := Run(context.Background(), opts, false)
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Equal(t, "abc123 | Rian Stockbower | rian@example.com\n", stdout)
}

func TestRun_APIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer server.Close()

	opts := newTestRootOptions()
	opts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	err := Run(context.Background(), opts, false)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "getting current user")
}
