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
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
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
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, run(context.Background(), opts))

	want := "abc | No Email | -\n"
	if stdout.String() != want {
		t.Errorf("me empty-email output:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRun_Extended_EmitsThreeSpecRows(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:        "abc123",
		DisplayName:      "Rian Stockbower",
		EmailAddress:     "rian@monitapp.io",
		Active:           true,
		TimeZone:         "Etc/GMT",
		Locale:           "en_US",
		Groups:           &api.UserCountBlock{Size: 9},
		ApplicationRoles: &api.UserCountBlock{Size: 1},
	}
	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, run(context.Background(), opts))

	want := "abc123 | Rian Stockbower | rian@monitapp.io\n" +
		"Timezone: Etc/GMT   Locale: en_US   Active: yes\n" +
		"Groups: 9   Application Roles: 1\n"
	if stdout.String() != want {
		t.Errorf("me --extended output:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRun_Extended_DashesWhenOptionalFieldsMissing(t *testing.T) {
	t.Parallel()
	user := &api.User{AccountID: "abc", DisplayName: "Bob", Active: false}
	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, run(context.Background(), opts))

	want := "abc | Bob | -\n" +
		"Timezone: -   Locale: -   Active: no\n" +
		"Groups: -   Application Roles: -\n"
	if stdout.String() != want {
		t.Errorf("me --extended output with missing fields:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRun_IDOnly(t *testing.T) {
	t.Parallel()
	user := &api.User{AccountID: "abc123", DisplayName: "John Doe", Active: true}
	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, run(context.Background(), opts))
	testutil.Equal(t, stdout.String(), "abc123\n")
}

func TestRun_IDOnlyPrecedenceOverExtended(t *testing.T) {
	t.Parallel()
	user := &api.User{AccountID: "abc123", DisplayName: "John Doe", Active: true}
	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	var stdout bytes.Buffer
	// Even with --extended and --fulltext set, --id wins: only the accountID.
	opts := &root.Options{Output: "table", IDOnly: true, Extended: true, FullText: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, run(context.Background(), opts))
	testutil.Equal(t, stdout.String(), "abc123\n")
}

func TestRun_JSON_UsesArtifactLayer(t *testing.T) {
	t.Parallel()
	user := &api.User{AccountID: "abc123", DisplayName: "John Doe", EmailAddress: "john@example.com", Active: true}
	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, run(context.Background(), opts))

	output := stdout.String()
	testutil.Contains(t, output, `"accountId"`)
	testutil.Contains(t, output, "abc123")
	testutil.Contains(t, output, `"displayName"`)
	testutil.Contains(t, output, "John Doe")
}

func TestRun_Plain_EmitsBareAccountID(t *testing.T) {
	t.Parallel()
	// Legacy contract (pre-#237): `-o plain` emits just the accountID, no
	// name, no email. --id is the preferred surface, but the plain format
	// stays working for backwards compatibility with unmigrated scripts.
	user := &api.User{AccountID: "abc123", DisplayName: "John Doe", EmailAddress: "john@example.com", Active: true}
	server := newTestUserServer(t, http.StatusOK, user)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "plain", Stdout: &stdout, Stderr: &bytes.Buffer{}}
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
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	err := run(context.Background(), opts)
	testutil.NotNil(t, err)
}

func TestRun_ExpandParamGatedOnExtended(t *testing.T) {
	t.Parallel()
	// Verifies the /myself request carries expand=groups,applicationRoles
	// ONLY when --extended is set. Default-mode callers don't render those
	// counts, so the wasted payload is skipped.
	cases := []struct {
		name     string
		extended bool
		wantExp  string
	}{
		{"default omits expand", false, ""},
		{"extended requests groups+roles", true, "groups,applicationRoles"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var captured string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				captured = r.URL.Query().Get("expand")
				_ = json.NewEncoder(w).Encode(&api.User{AccountID: "a", DisplayName: "x", Active: true})
			}))
			defer server.Close()

			var stdout bytes.Buffer
			opts := &root.Options{Output: "table", Extended: tc.extended, Stdout: &stdout, Stderr: &bytes.Buffer{}}
			opts.SetAPIClient(newClient(t, server.URL))

			testutil.RequireNoError(t, run(context.Background(), opts))
			if captured != tc.wantExp {
				t.Errorf("expand = %q, want %q", captured, tc.wantExp)
			}
		})
	}
}
