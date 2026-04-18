package users

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func newClient(t *testing.T, url string) *api.Client {
	t.Helper()
	c, err := api.New(api.ClientConfig{URL: url, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)
	return c
}

func newTestUserServer(_ *testing.T, user api.User) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(user)
	}))
}

func newTestUsersServer(_ *testing.T, users []api.User) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(users)
	}))
}

// ----- users get -----

func TestNewGetCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newGetCmd(opts)

	testutil.Equal(t, cmd.Use, "get <account-id>")
	testutil.NotEmpty(t, cmd.Short)
	testutil.NotNil(t, cmd.Flags().Lookup("fields"))
}

func TestRunGet_DefaultOutputMatchesSpecOneLiner(t *testing.T) {
	t.Parallel()
	server := newTestUserServer(t, api.User{AccountID: "abc123", DisplayName: "John Doe", EmailAddress: "john@example.com", Active: true})
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runGet(context.Background(), opts, "abc123", ""))

	want := "abc123 | John Doe | john@example.com\n"
	if stdout.String() != want {
		t.Errorf("get default output:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRunGet_Extended_EmitsThreeSpecRows(t *testing.T) {
	t.Parallel()
	user := api.User{
		AccountID: "abc", DisplayName: "Rian", EmailAddress: "r@x.io", Active: true,
		TimeZone: "Etc/GMT", Locale: "en_US",
		Groups: &api.UserCountBlock{Size: 9}, ApplicationRoles: &api.UserCountBlock{Size: 1},
	}
	server := newTestUserServer(t, user)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runGet(context.Background(), opts, "abc", ""))

	want := "abc | Rian | r@x.io\n" +
		"Timezone: Etc/GMT   Locale: en_US   Active: yes\n" +
		"Groups: 9   Application Roles: 1\n"
	if stdout.String() != want {
		t.Errorf("get --extended:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRunGet_IDOnly_ShortCircuitsEverythingElse(t *testing.T) {
	t.Parallel()
	server := newTestUserServer(t, api.User{AccountID: "abc123", DisplayName: "X", Active: true})
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runGet(context.Background(), opts, "abc123", "NAME,EMAIL"))
	testutil.Equal(t, stdout.String(), "abc123\n")
}

func TestRunGet_Fields_ProjectsDetailSection(t *testing.T) {
	t.Parallel()
	user := api.User{AccountID: "abc", DisplayName: "Rian", EmailAddress: "r@x.io", Active: true}
	server := newTestUserServer(t, user)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runGet(context.Background(), opts, "abc", "NAME,EMAIL"))

	// Identity is prepended by projection.Resolve; output flattens to labeled
	// Key:Value lines mirroring `issues get --fields`.
	want := "ACCOUNT_ID: abc\nNAME: Rian\nEMAIL: r@x.io\n"
	if stdout.String() != want {
		t.Errorf("get --fields:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRunGet_Fields_JSONReturnsError(t *testing.T) {
	t.Parallel()
	server := newTestUserServer(t, api.User{AccountID: "abc", DisplayName: "X"})
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	err := runGet(context.Background(), opts, "abc", "NAME")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "--fields is not supported")
}

func TestRunGet_Fields_UnknownHeaderFails(t *testing.T) {
	t.Parallel()
	server := newTestUserServer(t, api.User{AccountID: "abc", DisplayName: "X", Active: true})
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	// Trigger the no-op fetch path (noFieldFetch returns nil) — this token
	// matches no header, no FieldID, and no human name. It must surface as
	// UnknownFieldError, not an UnrenderedFieldError or an API call.
	err := runGet(context.Background(), opts, "abc", "NOSUCHFIELD")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "unknown field")
}

func TestRunGet_JSON_PreservesArtifactOutput(t *testing.T) {
	t.Parallel()
	server := newTestUserServer(t, api.User{AccountID: "abc123", DisplayName: "John Doe", Active: true})
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runGet(context.Background(), opts, "abc123", ""))
	testutil.Contains(t, stdout.String(), `"accountId"`)
}

// ----- users search -----

func TestNewSearchCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newSearchCmd(opts)

	testutil.Equal(t, cmd.Use, "search <query>")
	testutil.NotNil(t, cmd.Flags().Lookup("max"))
	testutil.NotNil(t, cmd.Flags().Lookup("next-page-token"))
	testutil.NotNil(t, cmd.Flags().Lookup("fields"))
	testutil.Equal(t, cmd.Flags().Lookup("max").DefValue, "10")
}

func TestRunSearch_DefaultTableMatchesSpecColumnOrder(t *testing.T) {
	t.Parallel()
	users := []api.User{
		{AccountID: "a1", DisplayName: "Alice", EmailAddress: "a@x.io", Active: true},
		{AccountID: "b2", DisplayName: "Bob", Active: false},
	}
	server := newTestUsersServer(t, users)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runSearch(context.Background(), opts, "al", 10, "", ""))

	out := stdout.String()
	testutil.Contains(t, out, "ACCOUNT_ID | NAME | EMAIL | ACTIVE")
	testutil.Contains(t, out, "a1 | Alice | a@x.io | yes")
	testutil.Contains(t, out, "b2 | Bob | - | no")
}

func TestRunSearch_Extended_AppendsTimezoneLocale(t *testing.T) {
	t.Parallel()
	users := []api.User{
		{AccountID: "a1", DisplayName: "Alice", EmailAddress: "a@x.io", Active: true, TimeZone: "Etc/GMT", Locale: "en_US"},
	}
	server := newTestUsersServer(t, users)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runSearch(context.Background(), opts, "al", 10, "", ""))

	out := stdout.String()
	testutil.Contains(t, out, "ACCOUNT_ID | NAME | EMAIL | ACTIVE | TIMEZONE | LOCALE")
	testutil.Contains(t, out, "a1 | Alice | a@x.io | yes | Etc/GMT | en_US")
}

func TestRunSearch_Extended_DashesForRedactedFields(t *testing.T) {
	t.Parallel()
	// Instances that omit timeZone/locale from /user/search must not render
	// literal "false"/empty strings in the table cells.
	users := []api.User{{AccountID: "a1", DisplayName: "Alice", Active: true}}
	server := newTestUsersServer(t, users)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runSearch(context.Background(), opts, "al", 10, "", ""))
	testutil.Contains(t, stdout.String(), "a1 | Alice | - | yes | - | -")
}

func TestRunSearch_IDOnly_EmitsKeysOnly(t *testing.T) {
	t.Parallel()
	users := []api.User{
		{AccountID: "a1", DisplayName: "Alice"},
		{AccountID: "b2", DisplayName: "Bob"},
	}
	server := newTestUsersServer(t, users)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runSearch(context.Background(), opts, "al", 10, "", ""))
	testutil.Equal(t, stdout.String(), "a1\nb2\n")
}

func TestRunSearch_HasMore_AppendsTokenizedContinuation(t *testing.T) {
	t.Parallel()
	// len(users) == --max triggers hasMore. Continuation line embeds next
	// startAt per #230.
	users := []api.User{
		{AccountID: "a1", DisplayName: "Alice"},
		{AccountID: "b2", DisplayName: "Bob"},
	}
	server := newTestUsersServer(t, users)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runSearch(context.Background(), opts, "al", 2, "", ""))
	testutil.Contains(t, stdout.String(), "More results available (next: 2)")
}

func TestRunSearch_IDOnly_EmitsTokenizedContinuation(t *testing.T) {
	t.Parallel()
	users := []api.User{
		{AccountID: "a1", DisplayName: "Alice"},
		{AccountID: "b2", DisplayName: "Bob"},
	}
	server := newTestUsersServer(t, users)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runSearch(context.Background(), opts, "al", 2, "", ""))
	testutil.Equal(t, stdout.String(), "a1\nb2\nMore results available (next: 2)\n")
}

func TestRunSearch_NextPageToken_AdvancesStartAt(t *testing.T) {
	t.Parallel()
	var capturedStartAt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedStartAt = r.URL.Query().Get("startAt")
		_ = json.NewEncoder(w).Encode([]api.User{{AccountID: "c3", DisplayName: "Carol"}})
	}))
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runSearch(context.Background(), opts, "al", 10, "20", ""))
	testutil.Equal(t, capturedStartAt, "20")
}

func TestRunSearch_NextPageToken_RejectsNonNumeric(t *testing.T) {
	t.Parallel()
	server := newTestUsersServer(t, []api.User{})
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	err := runSearch(context.Background(), opts, "al", 10, "not-a-number", "")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "invalid --next-page-token")
}

func TestRunSearch_NextPageToken_RejectsNegative(t *testing.T) {
	t.Parallel()
	// strconv.Atoi happily parses "-1"; the n < 0 guard must still reject it.
	server := newTestUsersServer(t, []api.User{})
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	err := runSearch(context.Background(), opts, "al", 10, "-1", "")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "invalid --next-page-token")
	testutil.Contains(t, err.Error(), "non-negative")
}

func TestRunSearch_Fields_JSONReturnsError(t *testing.T) {
	t.Parallel()
	server := newTestUsersServer(t, []api.User{})
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	err := runSearch(context.Background(), opts, "al", 10, "", "NAME")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "--fields is not supported")
}

func TestRunSearch_Fields_ProjectsToSelectedColumns(t *testing.T) {
	t.Parallel()
	users := []api.User{{AccountID: "a1", DisplayName: "Alice", EmailAddress: "a@x.io", Active: true}}
	server := newTestUsersServer(t, users)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runSearch(context.Background(), opts, "al", 10, "", "NAME"))
	out := stdout.String()
	// ACCOUNT_ID is the identity column and is always retained.
	testutil.Contains(t, out, "ACCOUNT_ID | NAME")
	if strings.Contains(out, "EMAIL") || strings.Contains(out, "ACTIVE") {
		t.Errorf("projected output leaked non-selected columns: %q", out)
	}
}

func TestRunSearch_Empty_ShowsFriendlyMessage(t *testing.T) {
	t.Parallel()
	server := newTestUsersServer(t, []api.User{})
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runSearch(context.Background(), opts, "ghost", 10, "", ""))
	testutil.Contains(t, stdout.String(), "No users found")
}

func TestRunSearch_JSON_RendersArtifacts(t *testing.T) {
	t.Parallel()
	users := []api.User{{AccountID: "a1", DisplayName: "Alice", EmailAddress: "a@x.io", Active: true}}
	server := newTestUsersServer(t, users)
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runSearch(context.Background(), opts, "al", 10, "", ""))
	testutil.Contains(t, stdout.String(), `"accountId"`)
}

func TestRunSearch_ExpandParamNotSentOnSearchEndpoint(t *testing.T) {
	t.Parallel()
	// /user/search ignores expand; we must not send it (wasted bytes + risk
	// of upstream behavior change).
	var captured string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query().Get("expand")
		_ = json.NewEncoder(w).Encode([]api.User{})
	}))
	defer server.Close()

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(newClient(t, server.URL))

	testutil.RequireNoError(t, runSearch(context.Background(), opts, "al", 10, "", ""))
	if captured != "" {
		t.Errorf("expand param should not be sent to /user/search, got %q", captured)
	}
}
