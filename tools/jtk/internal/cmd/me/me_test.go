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

func newClient(t *testing.T, url string) *api.Client {
	t.Helper()
	client, err := api.New(api.ClientConfig{URL: url, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)
	return client
}

func TestRun_DefaultOutputMatchesSpecOneLiner(t *testing.T) {
	t.Parallel()
	user := &api.User{AccountID: "abc123", DisplayName: "John Doe", EmailAddress: "john@example.com", Active: true}
	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, run(context.Background(), opts))

	want := "abc123 | John Doe | john@example.com\n"
	if stdout.String() != want {
		t.Errorf("me default output:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRun_EmptyEmailRendersDash(t *testing.T) {
	t.Parallel()
	user := &api.User{AccountID: "abc", DisplayName: "No Email", Active: true}
	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, run(context.Background(), opts))

	want := "abc | No Email | -\n"
	if stdout.String() != want {
		t.Errorf("me empty-email output:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRun_IDOnly(t *testing.T) {
	t.Parallel()
	user := &api.User{AccountID: "abc123", DisplayName: "John Doe", Active: true}
	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{IDOnly: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, run(context.Background(), opts))
	testutil.Equal(t, stdout.String(), "abc123\n")
}

func TestRun_AuthFailure(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	err := run(context.Background(), opts)
	testutil.NotNil(t, err)
}
