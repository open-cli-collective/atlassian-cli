package configcmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRunTest_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/user/current") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"accountId": "123", "displayName": "Test User", "email": "test@example.com"}`))
			return
		}
		// Spaces endpoint
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	err := runTest(context.Background(), rootOpts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "Testing connection... success!\n\nAuthentication successful\nAPI access verified\n\nAuthenticated as: Test User (test@example.com)\nAccount ID: 123\n", rootOpts.Stderr.(*bytes.Buffer).String())
}

func TestRunTest_SuccessUserLookupFallback(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/user/current") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "user lookup failed"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	err := runTest(context.Background(), rootOpts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "Testing connection... success!\n\nYour cfl configuration is working correctly.\n", rootOpts.Stderr.(*bytes.Buffer).String())
}

func TestRunTest_AuthFailure(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message": "Unauthorized"}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "bad-token")
	rootOpts.SetAPIClient(client)

	err := runTest(context.Background(), rootOpts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "connection test failed")
	testutil.Equal(t, "", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "Testing connection... failed!\n\nTroubleshooting:\n  - Verify your URL is correct (should include https://)\n  - Check your email and API token\n  - Ensure your API token hasn't expired\n  - Verify you have permission to access Confluence\n\nTo regenerate an API token:\n  https://id.atlassian.com/manage-profile/security/api-tokens\n", rootOpts.Stderr.(*bytes.Buffer).String())
}

func TestRunTest_ServerError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message": "Server error"}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	err := runTest(context.Background(), rootOpts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "connection test failed")
	testutil.Equal(t, "", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "Testing connection... failed!\n\nTroubleshooting:\n  - Verify your URL is correct (should include https://)\n  - Check your email and API token\n  - Ensure your API token hasn't expired\n  - Verify you have permission to access Confluence\n\nTo regenerate an API token:\n  https://id.atlassian.com/manage-profile/security/api-tokens\n", rootOpts.Stderr.(*bytes.Buffer).String())
}
