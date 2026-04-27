package fields

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
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRegister(t *testing.T) {
	t.Parallel()
	rootCmd, opts := root.NewCmd()
	Register(rootCmd, opts)

	cmd, _, err := rootCmd.Find([]string{"fields"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, cmd.Name(), "fields")
	testutil.Equal(t, cmd.Aliases, []string{"field", "f"})
}

func TestNewListCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newListCmd(opts)

	testutil.Equal(t, cmd.Use, "list")
	testutil.NotEmpty(t, cmd.Short)

	customFieldsFlag := cmd.Flags().Lookup("custom-fields")
	testutil.NotNil(t, customFieldsFlag)
	testutil.Equal(t, customFieldsFlag.DefValue, "false")

	customFlag := cmd.Flags().Lookup("custom")
	testutil.NotNil(t, customFlag)
	testutil.Equal(t, customFlag.DefValue, "false")
	if customFlag.Hidden != true {
		t.Error("--custom flag should be hidden")
	}

	nameFlag := cmd.Flags().Lookup("name")
	testutil.NotNil(t, nameFlag)
	testutil.Equal(t, nameFlag.DefValue, "")
}

func TestRunList_CustomFieldsFlag_CobraExecute(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{
			{ID: "summary", Name: "Summary", Custom: false, Schema: api.FieldSchema{Type: "string"}},
			{ID: "customfield_10100", Name: "Environment", Custom: true, Schema: api.FieldSchema{Type: "option"}},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	rootCmd, opts := root.NewCmd()
	opts.Output = "table"
	opts.Stdout = &stdout
	opts.Stderr = &bytes.Buffer{}
	opts.SetAPIClient(client)
	Register(rootCmd, opts)

	rootCmd.SetArgs([]string{"fields", "list", "--custom-fields"})
	err = rootCmd.Execute()
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "customfield_10100")
	testutil.NotContains(t, out, "summary")
}

func TestRunList_Table(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{
			{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
			{ID: "customfield_10100", Name: "Environment", Custom: true, Schema: api.FieldSchema{Type: "option"}},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false, "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "summary")
	testutil.Contains(t, stdout.String(), "customfield_10100")
	testutil.Contains(t, stdout.String(), "Environment")
}

func TestRunList_ColumnOrder(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{
			{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false, "")
	testutil.RequireNoError(t, err)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	cols := strings.Split(lines[1], " | ")
	testutil.Equal(t, cols[0], "summary")
	testutil.Equal(t, cols[1], "string")
	testutil.Equal(t, cols[2], "Summary")
}

func TestRunList_IDOnly(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{
			{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
			{ID: "customfield_10100", Name: "Environment", Custom: true, Schema: api.FieldSchema{Type: "option"}},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false, "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "summary\ncustomfield_10100\n")
}

func TestRunList_IDOnly_Empty(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false, "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "")
}

func TestRunList_Extended_ColumnOrder(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{
			{
				ID:          "summary",
				Name:        "Summary",
				Schema:      api.FieldSchema{Type: "string"},
				Searchable:  true,
				Navigable:   true,
				Orderable:   true,
				ClauseNames: []string{"summary"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false, "")
	testutil.RequireNoError(t, err)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	cols := strings.Split(lines[1], " | ")
	testutil.Equal(t, cols[0], "summary")
	testutil.Equal(t, cols[1], "string")
	testutil.Equal(t, cols[2], "yes")
	testutil.Equal(t, cols[3], "yes")
	testutil.Equal(t, cols[4], "yes")
	testutil.Equal(t, cols[5], "summary")
	testutil.Equal(t, cols[6], "Summary")
}

func TestRunList_JSON(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{
			{ID: "customfield_10100", Name: "Environment", Custom: true},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false, "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), `"id"`)
	testutil.Contains(t, stdout.String(), "customfield_10100")
}

func TestRunList_Empty(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false, "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No fields found")
}

func TestRunList_NameFilter(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{
			{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
			{ID: "customfield_10016", Name: "Story Points", Custom: true, Schema: api.FieldSchema{Type: "number"}},
			{ID: "customfield_10020", Name: "Story Status", Custom: true, Schema: api.FieldSchema{Type: "option"}},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false, "story")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Story Points")
	testutil.Contains(t, stdout.String(), "Story Status")
	testutil.NotContains(t, stdout.String(), "Summary")
}

func TestRunList_NameFilter_CaseInsensitive(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{
			{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
			{ID: "customfield_10016", Name: "Story Points", Custom: true, Schema: api.FieldSchema{Type: "number"}},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false, "STORY")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Story Points")
	testutil.NotContains(t, stdout.String(), "Summary")
}

func TestRunList_NameFilter_NoMatch(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{
			{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false, "nonexistent")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No fields found")
}

func TestRunList_NameFilter_JSON(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{
			{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
			{ID: "customfield_10016", Name: "Story Points", Custom: true, Schema: api.FieldSchema{Type: "number"}},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false, "story")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Story Points")
	testutil.NotContains(t, stdout.String(), "Summary")
}

func TestRunList_NameFilter_WithCustom(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Server returns a mix; the custom-only filter is applied locally after fetching all fields.
		_ = json.NewEncoder(w).Encode([]api.Field{
			{ID: "summary", Name: "Summary", Custom: false, Schema: api.FieldSchema{Type: "string"}},
			{ID: "customfield_10016", Name: "Story Points", Custom: true, Schema: api.FieldSchema{Type: "number"}},
			{ID: "customfield_10020", Name: "Sprint", Custom: true, Schema: api.FieldSchema{Type: "array"}},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, true, "story")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Story Points")
	testutil.NotContains(t, stdout.String(), "Sprint")
	testutil.NotContains(t, stdout.String(), "Summary")
}

func TestNewCreateCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newCreateCmd(opts)

	testutil.Equal(t, cmd.Use, "create")

	nameFlag := cmd.Flags().Lookup("name")
	testutil.NotNil(t, nameFlag)

	typeFlag := cmd.Flags().Lookup("type")
	testutil.NotNil(t, typeFlag)

	descFlag := cmd.Flags().Lookup("description")
	testutil.NotNil(t, descFlag)
}

func TestRunCreate(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(api.Field{
			ID:     "customfield_10100",
			Name:   "Environment",
			Custom: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "Environment", "com.atlassian.jira.plugin.system.customfieldtypes:select", "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	testutil.Contains(t, lines[0], "ID")
	testutil.Contains(t, lines[0], "NAME")
	testutil.Contains(t, lines[0], "TYPE")
	testutil.Contains(t, out, "customfield_10100")
	testutil.Contains(t, out, "Environment")
}

func TestRunCreate_IDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(api.Field{
			ID:     "customfield_10100",
			Name:   "Environment",
			Custom: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "Environment", "select", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "customfield_10100\n")
}

func TestRunCreate_JSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(api.Field{
			ID:     "customfield_10100",
			Name:   "Environment",
			Custom: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "Environment", "select", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "customfield_10100")
}

func TestNewDeleteCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newDeleteCmd(opts)

	testutil.Equal(t, cmd.Use, "delete <field-id>")

	forceFlag := cmd.Flags().Lookup("force")
	testutil.NotNil(t, forceFlag)
	testutil.Equal(t, forceFlag.DefValue, "false")
}

func TestRunDelete_Force(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		testutil.Contains(t, r.URL.Path, "/trash")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "customfield_10100", true)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted field customfield_10100 (moved to trash — use fields restore to recover)\n")
}

func TestRunDelete_NoForce_Declined(t *testing.T) {
	t.Parallel()
	client, err := api.New(api.ClientConfig{URL: "https://test.atlassian.net", Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Stdin:  bytes.NewBufferString("n\n"),
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "customfield_10100", false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deletion cancelled.\n")
}

func TestRunDelete_NoForce_Accepted(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Stdin:  bytes.NewBufferString("y\n"),
	}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "customfield_10100", false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted field customfield_10100 (moved to trash — use fields restore to recover)\n")
}

func TestRunRestore(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			testutil.Contains(t, r.URL.Path, "/restore")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]api.Field{
				{ID: "customfield_10100", Name: "Environment", Custom: true},
			})
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runRestore(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	testutil.Contains(t, lines[0], "ID")
	testutil.Contains(t, lines[0], "NAME")
	testutil.Contains(t, out, "customfield_10100")
}

func TestRunRestore_IDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runRestore(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "customfield_10100\n")
}

// --- Contexts tests ---

func TestNewContextsCmd(t *testing.T) {
	t.Parallel()
	rootCmd, opts := root.NewCmd()
	Register(rootCmd, opts)

	cmd, _, err := rootCmd.Find([]string{"fields", "contexts"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, cmd.Name(), "contexts")
	testutil.Equal(t, cmd.Aliases, []string{"context", "ctx"})
}

func TestRunContextsList_Table(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
			Values: []api.FieldContext{
				{ID: "10001", Name: "Default", IsGlobalContext: true, IsAnyIssueType: true},
				{ID: "10002", Name: "Bug Context", IsGlobalContext: false, IsAnyIssueType: false},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runContextsList(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Default")
	testutil.Contains(t, stdout.String(), "Bug Context")
}

func TestRunContextsList_Empty(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{Values: []api.FieldContext{}})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runContextsList(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No contexts found")
}

func TestRunContextsCreate(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(api.FieldContext{
			ID:   "10003",
			Name: "Bug Context",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runContextsCreate(context.Background(), opts, "customfield_10100", "Bug Context", "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	testutil.Contains(t, lines[0], "ID")
	testutil.Contains(t, lines[0], "NAME")
	testutil.Contains(t, out, "10003")
	testutil.Contains(t, out, "Bug Context")
}

func TestRunContextsCreate_IDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(api.FieldContext{
			ID:   "10003",
			Name: "Bug Context",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runContextsCreate(context.Background(), opts, "customfield_10100", "Bug Context", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "10003\n")
}

func TestRunContextsDelete_Force(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runContextsDelete(context.Background(), opts, "customfield_10100", "10003", true)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Deleted context 10003")
}

func TestRunContextsDelete_NoForce_Declined(t *testing.T) {
	t.Parallel()
	client, err := api.New(api.ClientConfig{URL: "https://test.atlassian.net", Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Stdin:  bytes.NewBufferString("n\n"),
	}
	opts.SetAPIClient(client)

	err = runContextsDelete(context.Background(), opts, "customfield_10100", "10003", false)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Deletion cancelled")
}

// --- Options tests ---

func TestNewOptionsCmd(t *testing.T) {
	t.Parallel()
	rootCmd, opts := root.NewCmd()
	Register(rootCmd, opts)

	cmd, _, err := rootCmd.Find([]string{"fields", "options"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, cmd.Name(), "options")
	testutil.Equal(t, cmd.Aliases, []string{"option", "opt"})
}

func TestResolveContextID_Explicit(t *testing.T) {
	t.Parallel()
	// When context flag is provided, it should be used directly
	id, err := resolveContextID(context.Background(), nil, "customfield_10100", "10001")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, id, "10001")
}

func TestResolveContextID_AutoDetect(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
			Values: []api.FieldContext{
				{ID: "10001", Name: "Default"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	id, err := resolveContextID(context.Background(), client, "customfield_10100", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, id, "10001")
}

func TestRunOptionsList_Table(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			// GetFieldContexts (auto-detect)
			_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		// GetFieldContextOptions (GET uses "values" key)
		_ = json.NewEncoder(w).Encode(api.FieldContextOptionsResponse{
			Values: []api.FieldContextOption{
				{ID: "1", Value: "Production", Disabled: false},
				{ID: "2", Value: "Staging", Disabled: true},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runOptionsList(context.Background(), opts, "customfield_10100", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Production")
	testutil.Contains(t, stdout.String(), "Staging")
}

func TestRunOptionsList_Empty(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(api.FieldContextOptionsResponse{Values: []api.FieldContextOption{}})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runOptionsList(context.Background(), opts, "customfield_10100", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No options found")
}

func TestRunOptionsAdd(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		testutil.Equal(t, r.Method, http.MethodPost)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"options": []api.FieldContextOption{
				{ID: "3", Value: "Option A"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runOptionsAdd(context.Background(), opts, "customfield_10100", "Option A", "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	testutil.Contains(t, lines[0], "ID")
	testutil.Contains(t, lines[0], "VALUE")
	testutil.Contains(t, out, "3")
	testutil.Contains(t, out, "Option A")
}

func TestRunOptionsAdd_IDOnly(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"options": []api.FieldContextOption{
				{ID: "3", Value: "Option A"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runOptionsAdd(context.Background(), opts, "customfield_10100", "Option A", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "3\n")
}

func TestRunOptionsUpdate(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		testutil.Equal(t, r.Method, http.MethodPut)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"options": []api.FieldContextOption{
				{ID: "3", Value: "Option A (updated)"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runOptionsUpdate(context.Background(), opts, "customfield_10100", "3", "Option A (updated)", "")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	testutil.Contains(t, lines[0], "ID")
	testutil.Contains(t, lines[0], "VALUE")
	testutil.Contains(t, out, "Option A (updated)")
}

func TestRunOptionsUpdate_IDOnly(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"options": []api.FieldContextOption{
				{ID: "3", Value: "Option A (updated)"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runOptionsUpdate(context.Background(), opts, "customfield_10100", "3", "Option A (updated)", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "3\n")
}

func TestRunOptionsDelete_Force(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		testutil.Equal(t, r.Method, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runOptionsDelete(context.Background(), opts, "customfield_10100", "3", "", true)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted option 3 from context 10001\n")
}

func TestRunOptionsDelete_NoForce_Declined(t *testing.T) {
	t.Parallel()
	client, err := api.New(api.ClientConfig{URL: "https://test.atlassian.net", Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Stdin:  bytes.NewBufferString("n\n"),
	}
	opts.SetAPIClient(client)

	err = runOptionsDelete(context.Background(), opts, "customfield_10100", "3", "", false)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deletion cancelled.\n")
}

func TestRunDelete_JSONOutputEmitsText(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "customfield_10100", true)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted field customfield_10100 (moved to trash — use fields restore to recover)\n")
	testutil.Equal(t, stderr.String(), "")
}

func TestRunOptionsDelete_JSONOutputEmitsText(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		testutil.Equal(t, r.Method, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runOptionsDelete(context.Background(), opts, "customfield_10100", "3", "", true)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "Deleted option 3 from context 10001\n")
	testutil.Equal(t, stderr.String(), "")
}

// --- Cache-hit tests ---
// These tests are non-parallel: SetRootForTest / SetInstanceKeyForTest are
// process-globals that races with t.Parallel() tests writing them.

func TestRunList_CacheHit_SkipsLiveCall(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	testutil.RequireNoError(t, cache.WriteResource("fields", "24h", []api.Field{
		{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
		{ID: "customfield_10016", Name: "Story Points", Custom: true, Schema: api.FieldSchema{Type: "number"}},
	}))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when fields cache is fresh")
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false, "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Summary")
	testutil.Contains(t, stdout.String(), "Story Points")
}

func TestRunList_CacheHit_CustomFilter_SkipsLiveCall(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	testutil.RequireNoError(t, cache.WriteResource("fields", "24h", []api.Field{
		{ID: "summary", Name: "Summary", Custom: false, Schema: api.FieldSchema{Type: "string"}},
		{ID: "customfield_10016", Name: "Story Points", Custom: true, Schema: api.FieldSchema{Type: "number"}},
		{ID: "customfield_10020", Name: "Sprint", Custom: true, Schema: api.FieldSchema{Type: "array"}},
	}))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when fields cache is fresh")
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, true, "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Story Points")
	testutil.Contains(t, stdout.String(), "Sprint")
	testutil.NotContains(t, stdout.String(), "Summary")
}

// --- Show command tests ---

func newShowTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/context/projectmapping"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"contextId": "10100", "isGlobalContext": true},
					{"contextId": "10101", "projectId": "10001", "isGlobalContext": false},
					{"contextId": "10102", "projectId": "10002", "isGlobalContext": false},
				},
				"isLast": true,
			})
		case strings.Contains(r.URL.Path, "/context/10100/option"):
			_ = json.NewEncoder(w).Encode(api.FieldContextOptionsResponse{
				Values: []api.FieldContextOption{
					{ID: "20001", Value: "Platform"},
					{ID: "20002", Value: "Integration"},
				},
				IsLast: true,
			})
		case strings.Contains(r.URL.Path, "/context/10101/option"):
			_ = json.NewEncoder(w).Encode(api.FieldContextOptionsResponse{
				Values: []api.FieldContextOption{
					{ID: "20010", Value: "CapOne"},
				},
				IsLast: true,
			})
		case strings.Contains(r.URL.Path, "/context/10102/option"):
			_ = json.NewEncoder(w).Encode(api.FieldContextOptionsResponse{
				Values: []api.FieldContextOption{},
				IsLast: true,
			})
		case strings.Contains(r.URL.Path, "/context"):
			_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{
					{ID: "10100", Name: "Default Context", IsGlobalContext: true},
					{ID: "10101", Name: "MON Project Context"},
					{ID: "10102", Name: "ON Project Context"},
				},
				IsLast: true,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestRunShow_Table(t *testing.T) {
	t.Parallel()
	server := newShowTestServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runShow(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	testutil.True(t, len(lines) >= 5, "expected header + 4 data rows")

	// Header
	testutil.Contains(t, lines[0], "CONTEXT_ID")
	testutil.Contains(t, lines[0], "OPTION_VALUE")

	// Global context with options
	testutil.Contains(t, out, "(global)")
	testutil.Contains(t, out, "Platform")
	testutil.Contains(t, out, "Integration")

	// Project context with options
	testutil.Contains(t, out, "CapOne")

	// Empty context renders - | -
	cols := strings.Split(lines[4], " | ")
	testutil.Equal(t, cols[0], "10102")
	testutil.Equal(t, cols[3], "-")
	testutil.Equal(t, cols[4], "-")
}

func TestRunShow_IDOnly(t *testing.T) {
	t.Parallel()
	server := newShowTestServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runShow(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "10100\n10101\n10102\n")
}

func TestRunShow_JSON(t *testing.T) {
	t.Parallel()
	server := newShowTestServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runShow(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, `"context_id"`)
	testutil.Contains(t, out, `"option_value"`)
	testutil.Contains(t, out, "Platform")
}

func TestRunShow_Empty(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
			Values: []api.FieldContext{},
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runShow(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No contexts found")
}

func TestRunShow_OptionFetchError4xx(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/context/projectmapping"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"contextId": "10100", "isGlobalContext": true},
				},
				"isLast": true,
			})
		case strings.Contains(r.URL.Path, "/context/10100/option"):
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"errorMessage": "not a select field"})
		case strings.Contains(r.URL.Path, "/context"):
			_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{
					{ID: "10100", Name: "Default Context", IsGlobalContext: true},
				},
				IsLast: true,
			})
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runShow(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	cols := strings.Split(lines[1], " | ")
	testutil.Equal(t, cols[3], "-")
	testutil.Equal(t, cols[4], "-")
}

func TestRunShow_OptionFetchError404(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/context/projectmapping"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"contextId": "10100", "isGlobalContext": true},
				},
				"isLast": true,
			})
		case strings.Contains(r.URL.Path, "/context/10100/option"):
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"errorMessage": "not found"})
		case strings.Contains(r.URL.Path, "/context"):
			_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{
					{ID: "10100", Name: "Default Context", IsGlobalContext: true},
				},
				IsLast: true,
			})
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runShow(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	testutil.True(t, len(lines) >= 2, "expected header + data row")
	cols := strings.Split(lines[1], " | ")
	testutil.Equal(t, cols[3], "-")
	testutil.Equal(t, cols[4], "-")
}

func TestRunShow_OptionFetchError5xx(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/context/projectmapping"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"contextId": "10100", "isGlobalContext": true},
				},
				"isLast": true,
			})
		case strings.Contains(r.URL.Path, "/context/10100/option"):
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"errorMessage": "server error"})
		case strings.Contains(r.URL.Path, "/context"):
			_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{
					{ID: "10100", Name: "Default Context", IsGlobalContext: true},
				},
				IsLast: true,
			})
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runShow(context.Background(), opts, "customfield_10100")
	testutil.True(t, err != nil, "expected 5xx error to propagate")
	testutil.Contains(t, err.Error(), "server error")
}

func TestRunShow_EmptyJSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
			Values: []api.FieldContext{},
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runShow(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, strings.TrimSpace(stdout.String()), "[]")
}

func TestRunShow_EmptyIDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
			Values: []api.FieldContext{},
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runShow(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "")
}

func TestRunShow_MultiPageContexts(t *testing.T) {
	t.Parallel()
	contextCallCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/context/projectmapping"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"contextId": "10100", "isGlobalContext": true},
					{"contextId": "10101", "isGlobalContext": true},
				},
				"isLast": true,
			})
		case strings.Contains(r.URL.Path, "/option"):
			_ = json.NewEncoder(w).Encode(api.FieldContextOptionsResponse{
				Values: []api.FieldContextOption{},
				IsLast: true,
			})
		case strings.Contains(r.URL.Path, "/context"):
			contextCallCount++
			if contextCallCount == 1 {
				_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
					Values: []api.FieldContext{
						{ID: "10100", Name: "Page 1 Context", IsGlobalContext: true},
					},
					IsLast: false,
				})
			} else {
				_ = json.NewEncoder(w).Encode(api.FieldContextsResponse{
					Values: []api.FieldContext{
						{ID: "10101", Name: "Page 2 Context", IsGlobalContext: true},
					},
					IsLast: true,
				})
			}
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runShow(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "Page 1 Context")
	testutil.Contains(t, out, "Page 2 Context")
}

func TestNewShowCmd(t *testing.T) {
	t.Parallel()
	rootCmd, opts := root.NewCmd()
	Register(rootCmd, opts)

	cmd, _, err := rootCmd.Find([]string{"fields", "show"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, cmd.Name(), "show")
}
