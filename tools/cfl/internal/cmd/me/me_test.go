package me

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/spf13/cobra"

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

func TestRun_NormalizesPipesAndNewlines(t *testing.T) {
	t.Parallel()
	// Display name contains an embedded pipe and newline; without normalization
	// the row would have more than three pipe-delimited fields and would split
	// across multiple lines, breaking the documented contract.
	server := userServer(t, `{"accountId":"abc123","displayName":"Joe | Pwn\nNext","email":"joe@example.com"}`)
	defer server.Close()

	opts := newTestRootOptions()
	opts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	err := Run(context.Background(), opts, false)
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Equal(t, "abc123 | Joe \\| Pwn Next | joe@example.com\n", stdout)
}

func TestRegister_RegistersMeWithIDFlag(t *testing.T) {
	t.Parallel()
	rootCmd := &cobra.Command{Use: "cfl"}
	opts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	Register(rootCmd, opts)

	meCmd, _, err := rootCmd.Find([]string{"me"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "me", meCmd.Use)
	testutil.NotEmpty(t, meCmd.Short)

	idFlag := meCmd.Flags().Lookup("id")
	testutil.NotNil(t, idFlag)
	testutil.Equal(t, "false", idFlag.DefValue)
}

// TestExecute_IDFlagWiredThroughCobra drives the command via cobra.Execute()
// to confirm the --id flag actually toggles output, not just that the flag
// exists. Catches regressions where the flag is dropped or the RunE glue
// stops threading the boolean through to Run().
func TestExecute_IDFlagWiredThroughCobra(t *testing.T) {
	t.Parallel()
	server := userServer(t, `{"accountId":"abc123","displayName":"Rian Stockbower","email":"rian@example.com"}`)
	defer server.Close()

	opts := newTestRootOptions()
	opts.SetAPIClient(api.NewClient(server.URL, "test@example.com", "token"))

	rootCmd := &cobra.Command{Use: "cfl"}
	Register(rootCmd, opts)
	rootCmd.SetArgs([]string{"me", "--id"})

	err := rootCmd.Execute()
	testutil.RequireNoError(t, err)

	stdout := opts.Stdout.(*bytes.Buffer).String()
	testutil.Equal(t, "abc123\n", stdout)
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
