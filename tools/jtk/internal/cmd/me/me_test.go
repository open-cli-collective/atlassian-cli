package me

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

func TestNewMeCmd(t *testing.T) {
	t.Parallel()
	rootCmd, opts := root.NewCmd()
	Register(rootCmd, opts)

	cmd, _, err := rootCmd.Find([]string{"me"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, cmd.Use, "me")
	testutil.NotEmpty(t, cmd.Short)
}

func newTestUserServer(_ *testing.T, statusCode int, user *api.User) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
		if user != nil {
			_ = json.NewEncoder(w).Encode(user)
		}
	}))
}

func TestRun_Table(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:    "abc123",
		DisplayName:  "John Doe",
		EmailAddress: "john@example.com",
		Active:       true,
	}

	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = run(context.Background(), opts)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "abc123")
	testutil.Contains(t, output, "John Doe")
	testutil.Contains(t, output, "john@example.com")
	testutil.Contains(t, output, "yes")
}

func TestRun_IDOnly(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:    "abc123",
		DisplayName:  "John Doe",
		EmailAddress: "john@example.com",
		Active:       true,
	}

	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, run(context.Background(), opts))

	testutil.Equal(t, stdout.String(), "abc123\n")
}

func TestRun_IDOnlyPrecedenceOverExtended(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:    "abc123",
		DisplayName:  "John Doe",
		EmailAddress: "john@example.com",
		Active:       true,
	}

	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Extended: true, FullText: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, run(context.Background(), opts))

	// --id wins: only the accountID, no presenter output.
	testutil.Equal(t, stdout.String(), "abc123\n")
}

func TestRun_JSON(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:    "abc123",
		DisplayName:  "John Doe",
		EmailAddress: "john@example.com",
		Active:       true,
	}

	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = run(context.Background(), opts)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, `"accountId"`)
	testutil.Contains(t, output, "abc123")
	testutil.Contains(t, output, `"displayName"`)
	testutil.Contains(t, output, "John Doe")
}

func TestRun_WithEmail(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:    "abc123",
		DisplayName:  "John Doe",
		EmailAddress: "john@example.com",
		Active:       true,
	}

	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = run(context.Background(), opts)
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "john@example.com")
}

func TestRun_WithoutEmail(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:   "abc123",
		DisplayName: "John Doe",
		Active:      true,
	}

	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = run(context.Background(), opts)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.NotContains(t, output, "Email:")
}

func TestRun_Plain(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:   "abc123",
		DisplayName: "John Doe",
		Active:      true,
	}

	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "plain", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = run(context.Background(), opts)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "abc123")
	testutil.NotContains(t, output, "John Doe")
}

func TestRun_AuthFailure(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = run(context.Background(), opts)
	testutil.NotNil(t, err)
}

func TestRun_UnpaddedKeyValueOutput(t *testing.T) {
	t.Parallel()
	// Verifies the migration from manual padding to RenderKeyValues produces unpadded output
	user := &api.User{
		AccountID:    "abc123",
		DisplayName:  "Alice",
		EmailAddress: "alice@example.com",
		Active:       true,
	}

	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = run(context.Background(), opts)
	testutil.RequireNoError(t, err)

	want := "Account ID: abc123\nDisplay Name: Alice\nEmail: alice@example.com\nActive: yes\n"
	if stdout.String() != want {
		t.Errorf("me output:\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}
