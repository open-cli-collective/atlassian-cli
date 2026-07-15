package remotelinks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestNewListCmd(t *testing.T) {
	t.Parallel()
	cmd := newListCmd(&root.Options{})
	testutil.Equal(t, cmd.Use, "list <issue-key>")
	testutil.Equal(t, cmd.Short, "List remote links on an issue")
}

func TestNewAddCmd_RequiresURL(t *testing.T) {
	t.Parallel()
	cmd := newAddCmd(&root.Options{})
	testutil.Equal(t, cmd.Use, "add <issue-key>")
	// --url is marked required.
	flag := cmd.Flags().Lookup("url")
	testutil.NotNil(t, flag)
}

func TestRegister_ExecutesCanonicalAndAliasCommands(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "canonical", args: []string{"remotelinks", "list", "PROJ-123", "--id"}},
		{name: "singular-alias", args: []string{"remotelink", "list", "PROJ-123", "--id"}},
		{name: "short-alias", args: []string{"rl", "list", "PROJ-123", "--id"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := remoteLinkListServer(t)
			defer server.Close()

			client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
			testutil.RequireNoError(t, err)

			rootCmd, opts := root.NewCmd()
			var stdout bytes.Buffer
			opts.Stdout = &stdout
			opts.Stderr = &bytes.Buffer{}
			opts.SetAPIClient(client)
			Register(rootCmd, opts)
			rootCmd.SetArgs(tc.args)

			err = rootCmd.Execute()
			testutil.RequireNoError(t, err)
			testutil.Equal(t, stdout.String(), "10001\n")
		})
	}
}

func TestRegister_RemoveVerbRejected(t *testing.T) {
	t.Parallel()

	rootCmd, opts := root.NewCmd()
	Register(rootCmd, opts)

	cmd, _, err := rootCmd.Find([]string{"remotelinks"})
	testutil.RequireNoError(t, err)
	testutil.NotNil(t, cmd)

	subcommands := cmd.Commands()
	testutil.Equal(t, len(subcommands), 3)

	var names []string
	for _, subcommand := range subcommands {
		names = append(names, subcommand.Name())
	}

	joined := "," + strings.Join(names, ",") + ","
	testutil.Contains(t, joined, ",list,")
	testutil.Contains(t, joined, ",add,")
	testutil.Contains(t, joined, ",delete,")
	testutil.NotContains(t, joined, ",remove,")
}

func remoteLinkListServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":           10001,
				"relationship": "mentioned in",
				"object": map[string]any{
					"url":     "https://github.com/owner/repo/issues/456",
					"title":   "GitHub #456",
					"summary": "Some issue",
				},
			},
		})
	}))
}

func TestRunList(t *testing.T) {
	t.Parallel()
	server := remoteLinkListServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "PROJ-123", "")
	testutil.RequireNoError(t, err)
	out := stdout.String()
	testutil.Contains(t, out, "10001")
	testutil.Contains(t, out, "GitHub #456")
	testutil.Contains(t, out, "https://github.com/owner/repo/issues/456")
}

func TestRunList_Extended(t *testing.T) {
	t.Parallel()
	server := remoteLinkListServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "PROJ-123", "")
	testutil.RequireNoError(t, err)
	out := stdout.String()
	testutil.Contains(t, out, "RELATIONSHIP")
	testutil.Contains(t, out, "SUMMARY")
	testutil.Contains(t, out, "mentioned in")
}

func TestRunList_IDOnly(t *testing.T) {
	t.Parallel()
	server := remoteLinkListServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "PROJ-123", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "10001\n")
}

func TestRunList_FieldsProjection(t *testing.T) {
	t.Parallel()
	server := remoteLinkListServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "PROJ-123", "TITLE")
	testutil.RequireNoError(t, err)
	out := stdout.String()
	// ID is always present (Identity pin) even though not in --fields.
	testutil.Contains(t, out, "ID")
	testutil.Contains(t, out, "TITLE")
	testutil.NotContains(t, out, "URL")
}

func TestRunList_NoLinks(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "PROJ-123", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No remote links on PROJ-123")
}

func TestRunAdd(t *testing.T) {
	t.Parallel()
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issue/PROJ-123/remotelink")
		testutil.Equal(t, r.Method, http.MethodPost)
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 10010})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runAdd(context.Background(), opts, "PROJ-123", "https://example.com", "Example", "", "")
	testutil.RequireNoError(t, err)
	out := stdout.String()
	testutil.Contains(t, out, "Added remote link 10010 to PROJ-123")
	testutil.Contains(t, out, "https://example.com")

	var sent api.CreateRemoteLinkRequest
	err = json.Unmarshal(capturedBody, &sent)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, sent.Object.URL, "https://example.com")
	testutil.Equal(t, sent.Object.Title, "Example")
}

func TestRunAdd_SummaryAndRelationshipRoundTrip(t *testing.T) {
	t.Parallel()
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issue/PROJ-123/remotelink")
		testutil.Equal(t, r.Method, http.MethodPost)
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 10013})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runAdd(
		context.Background(),
		opts,
		"PROJ-123",
		"https://example.com",
		"Example",
		"Reference docs",
		"mentioned in",
	)
	testutil.RequireNoError(t, err)

	var sent api.CreateRemoteLinkRequest
	err = json.Unmarshal(capturedBody, &sent)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, sent.Relationship, "mentioned in")
	testutil.Equal(t, sent.Object.Summary, "Reference docs")
}

func TestRunAdd_TitleDefaultsToURL(t *testing.T) {
	t.Parallel()
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 10011})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runAdd(context.Background(), opts, "PROJ-123", "https://example.com", "", "", "")
	testutil.RequireNoError(t, err)

	var sent api.CreateRemoteLinkRequest
	err = json.Unmarshal(capturedBody, &sent)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, sent.Object.Title, "https://example.com")
}

func TestRunAdd_IDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 10012})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runAdd(context.Background(), opts, "PROJ-123", "https://example.com", "Example", "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "10012\n")
}

func TestRunDelete(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issue/PROJ-123/remotelink/10001")
		testutil.Equal(t, r.Method, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "PROJ-123", "10001")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted remote link 10001 from PROJ-123\n")
	testutil.Equal(t, stderr.String(), "")
}

func TestRunDelete_NonNumericID(t *testing.T) {
	t.Parallel()
	// A non-numeric link ID is rejected before any API call. No client is set
	// on opts, so reaching the API would panic — proving validation comes first.
	opts := &root.Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}

	err := runDelete(context.Background(), opts, "PROJ-123", "not-a-number")
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "invalid link ID")
}
