package users

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

func newTestUserServer(_ *testing.T, user api.User) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(user)
	}))
}

func TestNewGetCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newGetCmd(opts)

	testutil.Equal(t, cmd.Use, "get <account-id>")
	testutil.NotEmpty(t, cmd.Short)
}

func TestRunGet_Table(t *testing.T) {
	t.Parallel()
	user := api.User{
		AccountID:    "abc123",
		DisplayName:  "John Doe",
		EmailAddress: "john@example.com",
		Active:       true,
	}

	server := newTestUserServer(t, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "abc123")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "abc123")
	testutil.Contains(t, output, "John Doe")
	testutil.Contains(t, output, "john@example.com")
	testutil.Contains(t, output, "yes")
}

func TestRunGet_IDOnly(t *testing.T) {
	t.Parallel()
	user := api.User{AccountID: "abc123", DisplayName: "John Doe", Active: true}

	server := newTestUserServer(t, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runGet(context.Background(), opts, "abc123"))
	testutil.Equal(t, stdout.String(), "abc123\n")
}

func TestRunGet_IDOnlyPrecedenceOverExtended(t *testing.T) {
	t.Parallel()
	user := api.User{AccountID: "abc123", DisplayName: "John Doe", Active: true}

	server := newTestUserServer(t, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Extended: true, FullText: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runGet(context.Background(), opts, "abc123"))
	testutil.Equal(t, stdout.String(), "abc123\n")
}

func TestRunGet_JSON(t *testing.T) {
	t.Parallel()
	user := api.User{
		AccountID:   "abc123",
		DisplayName: "John Doe",
		Active:      true,
	}

	server := newTestUserServer(t, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "abc123")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, `"accountId"`)
	testutil.Contains(t, output, "abc123")
}

func TestRunGet_InactiveUser(t *testing.T) {
	t.Parallel()
	user := api.User{
		AccountID:   "abc123",
		DisplayName: "John Doe",
		Active:      false,
	}

	server := newTestUserServer(t, user)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "abc123")
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "no")
}

func TestNewSearchCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newSearchCmd(opts)

	testutil.Equal(t, cmd.Use, "search <query>")
	testutil.NotEmpty(t, cmd.Short)

	maxFlag := cmd.Flags().Lookup("max")
	testutil.NotNil(t, maxFlag)
	testutil.Equal(t, maxFlag.DefValue, "10")
}

func newTestUsersServer(_ *testing.T, users []api.User) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(users)
	}))
}

func TestRunSearch_Table(t *testing.T) {
	t.Parallel()
	users := []api.User{
		{AccountID: "abc123", DisplayName: "John Doe", EmailAddress: "john@example.com", Active: true},
		{AccountID: "def456", DisplayName: "Jane Smith", EmailAddress: "jane@example.com", Active: false},
	}

	server := newTestUsersServer(t, users)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runSearch(context.Background(), opts, "john", 10)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "abc123")
	testutil.Contains(t, output, "John Doe")
	testutil.Contains(t, output, "john@example.com")
	testutil.Contains(t, output, "def456")
	testutil.Contains(t, output, "Jane Smith")
}

func TestRunSearch_JSON(t *testing.T) {
	t.Parallel()
	users := []api.User{
		{AccountID: "abc123", DisplayName: "John Doe", EmailAddress: "john@example.com", Active: true},
	}

	server := newTestUsersServer(t, users)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runSearch(context.Background(), opts, "john", 10)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, `"accountId"`)
	testutil.Contains(t, output, "abc123")
	testutil.Contains(t, output, `"displayName"`)
}

func TestRunSearch_Empty(t *testing.T) {
	t.Parallel()
	server := newTestUsersServer(t, []api.User{})
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runSearch(context.Background(), opts, "nobody", 10)
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "No users found")
}

func TestRunSearch_ActiveUser(t *testing.T) {
	t.Parallel()
	users := []api.User{
		{AccountID: "abc123", DisplayName: "John Doe", Active: true},
	}

	server := newTestUsersServer(t, users)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runSearch(context.Background(), opts, "john", 10)
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "yes")
}

func TestRunSearch_InactiveUser(t *testing.T) {
	t.Parallel()
	users := []api.User{
		{AccountID: "abc123", DisplayName: "John Doe", Active: false},
	}

	server := newTestUsersServer(t, users)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runSearch(context.Background(), opts, "john", 10)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "no")
}
