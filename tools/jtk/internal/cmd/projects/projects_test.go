package projects

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

	cmd, _, err := rootCmd.Find([]string{"projects"})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, cmd.Name(), "projects")
	testutil.Equal(t, cmd.Aliases, []string{"project", "proj", "p"})
}

func TestNewListCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newListCmd(opts)

	testutil.Equal(t, cmd.Use, "list")
	testutil.NotEmpty(t, cmd.Short)

	queryFlag := cmd.Flags().Lookup("query")
	testutil.NotNil(t, queryFlag)
	testutil.Equal(t, queryFlag.DefValue, "")

	maxFlag := cmd.Flags().Lookup("max")
	testutil.NotNil(t, maxFlag)
	testutil.Equal(t, maxFlag.DefValue, "50")
}

func TestRunList_DefaultColumnOrderMatchesSpec(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectSearchResponse{
			Values: []api.ProjectDetail{
				{Key: "TST", Name: "Test", ProjectTypeKey: "software", Lead: &api.User{DisplayName: "Lead"}},
			},
			Total:  1,
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "", "")
	testutil.RequireNoError(t, err)
	out := stdout.String()
	testutil.Contains(t, out, "KEY | TYPE | LEAD | NAME")
	testutil.Contains(t, out, "TST | software | Lead | Test")
}

func TestRunList_Extended_AppendsStyleAndCounts(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectSearchResponse{
			Values: []api.ProjectDetail{
				{
					Key: "TST", Name: "Test", ProjectTypeKey: "software",
					Lead:       &api.User{DisplayName: "Lead"},
					Style:      "classic",
					IssueTypes: []api.IssueType{{ID: "1", Name: "Epic"}, {ID: "2", Name: "SDLC"}},
					Components: []api.Component{{ID: "c1", Name: "A"}, {ID: "c2", Name: "B"}},
				},
			},
			Total: 1, IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runList(context.Background(), opts, "", 50, "", ""))
	out := stdout.String()
	testutil.Contains(t, out, "KEY | TYPE | LEAD | NAME | STYLE | ISSUE_TYPES | COMPONENTS")
	testutil.Contains(t, out, "TST | software | Lead | Test | classic | 2 | 2")
}

func TestRunList_HasMore_EmbedsTokenInContinuationLine(t *testing.T) {
	t.Parallel()
	// Two projects, IsLast=false → token should advance to startAt=2.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectSearchResponse{
			Values: []api.ProjectDetail{
				{Key: "A", Name: "A", ProjectTypeKey: "software"},
				{Key: "B", Name: "B", ProjectTypeKey: "software"},
			},
			StartAt: 0, MaxResults: 2, Total: 20, IsLast: false,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runList(context.Background(), opts, "", 2, "", ""))
	testutil.Contains(t, stdout.String(), "More results available (next: 2)")
}

func TestRunList_NextPageToken_AdvancesStartAt(t *testing.T) {
	t.Parallel()
	var captured string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query().Get("startAt")
		_ = json.NewEncoder(w).Encode(api.ProjectSearchResponse{Values: []api.ProjectDetail{}, IsLast: true})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runList(context.Background(), opts, "", 50, "25", ""))
	testutil.Equal(t, captured, "25")
}

func TestRunList_IDOnly_EmitsKeysOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectSearchResponse{
			Values: []api.ProjectDetail{
				{Key: "A"}, {Key: "B"},
			},
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runList(context.Background(), opts, "", 50, "", ""))
	testutil.Equal(t, stdout.String(), "A\nB\n")
}

func TestRunList_Fields_ProjectsToSelectedColumns(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectSearchResponse{
			Values: []api.ProjectDetail{
				{Key: "TST", Name: "Test", ProjectTypeKey: "software"},
			},
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runList(context.Background(), opts, "", 50, "", "KEY,NAME"))
	out := stdout.String()
	testutil.Contains(t, out, "KEY | NAME")
	if strings.Contains(out, "TYPE") || strings.Contains(out, "LEAD") {
		t.Errorf("projected output leaked non-selected columns: %q", out)
	}
}

func TestRunList_JSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectSearchResponse{
			Values: []api.ProjectDetail{
				{Key: "TST", Name: "Test", ProjectTypeKey: "software"},
			},
			Total:  1,
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), `"key"`)
	testutil.Contains(t, stdout.String(), "TST")
}

func TestRunList_Empty(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectSearchResponse{Values: []api.ProjectDetail{}, Total: 0, IsLast: true})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No projects found")
}

func TestNewGetCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newGetCmd(opts)

	testutil.Equal(t, cmd.Use, "get <project-key>")
}

func TestRunGet_DefaultSpecShape(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectDetail{
			ID: json.Number("10001"), Key: "TST", Name: "Test",
			ProjectTypeKey: "software", Style: "classic",
			Lead:       &api.User{DisplayName: "Lead"},
			IssueTypes: []api.IssueType{{ID: "1", Name: "Epic"}},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runGet(context.Background(), opts, "TST", ""))

	want := "TST  Test\n" +
		"Type: software   Lead: Lead   Style: classic\n" +
		"Issue Types: Epic\n" +
		"Components: 0   Versions: 0\n"
	if stdout.String() != want {
		t.Errorf("get default:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRunGet_Extended_EnumeratesComponentsAndFlags(t *testing.T) {
	t.Parallel()
	simplified := false
	private := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectDetail{
			ID: json.Number("10001"), Key: "TST", Name: "Test",
			ProjectTypeKey: "software", Style: "classic",
			Lead:       &api.User{AccountID: "u1", DisplayName: "Lead"},
			IssueTypes: []api.IssueType{{ID: "1", Name: "Epic"}},
			Components: []api.Component{{ID: "c1", Name: "A"}, {ID: "c2", Name: "B"}},
			Simplified: &simplified,
			IsPrivate:  &private,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runGet(context.Background(), opts, "TST", ""))

	out := stdout.String()
	testutil.Contains(t, out, "TST  Test")
	testutil.Contains(t, out, "Type: software   Lead: Lead (u1)   Style: classic")
	testutil.Contains(t, out, "Issue Types: Epic (1)")
	testutil.Contains(t, out, "Components: 2")
	testutil.Contains(t, out, "  c1 | A")
	testutil.Contains(t, out, "  c2 | B")
	testutil.Contains(t, out, "Versions: 0")
	testutil.Contains(t, out, "Simplified: no   Private: no")
}

func TestRunGet_Extended_MissingFlagsRenderDashes(t *testing.T) {
	t.Parallel()
	// API response lacks simplified / isPrivate — presenters must not render
	// literal "no" in either position.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"10001","key":"TST","name":"Test","projectTypeKey":"software","style":"classic","lead":{"displayName":"Lead"},"issueTypes":[],"components":[]}`))
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runGet(context.Background(), opts, "TST", ""))
	testutil.Contains(t, stdout.String(), "Simplified: -   Private: -")
}

func TestRunGet_IDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectDetail{
			ID: json.Number("10001"), Key: "TST", Name: "Test",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runGet(context.Background(), opts, "TST", "NAME"))
	testutil.Equal(t, stdout.String(), "TST\n")
}

func TestRunGet_Fields_ProjectsDetailSection(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectDetail{
			ID: json.Number("10001"), Key: "TST", Name: "Test",
			ProjectTypeKey: "software", Style: "classic",
			Lead: &api.User{AccountID: "u1", DisplayName: "Lead"},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runGet(context.Background(), opts, "TST", "NAME,LEAD"))
	out := stdout.String()
	testutil.Contains(t, out, "KEY: TST")
	testutil.Contains(t, out, "NAME: Test")
	testutil.Contains(t, out, "LEAD: Lead")
	if strings.Contains(out, "STYLE:") {
		t.Errorf("projection leaked unselected field in output: %q", out)
	}
}

func TestNewCreateCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newCreateCmd(opts)

	testutil.Equal(t, cmd.Use, "create")

	keyFlag := cmd.Flags().Lookup("key")
	testutil.NotNil(t, keyFlag)

	nameFlag := cmd.Flags().Lookup("name")
	testutil.NotNil(t, nameFlag)

	leadFlag := cmd.Flags().Lookup("lead")
	testutil.NotNil(t, leadFlag)
}

func TestRunCreate(t *testing.T) {
	// Isolate the cache and seed a user matching the input accountId so the
	// resolver can return it without hitting the refresh path.
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	testutil.RequireNoError(t, cache.WriteResource("users", "24h", []api.User{
		{AccountID: "557058:295fe89c-10c2-4b0c-ba84-a4dd14ea7729", DisplayName: "Lead"},
	}))

	// Jira's create endpoint returns an empty name, so the success message
	// should use the input name, not the response name.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(api.ProjectDetail{
			ID:   json.Number("10001"),
			Key:  "TST",
			Name: "",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "TST", "Test Project", "software", "557058:295fe89c-10c2-4b0c-ba84-a4dd14ea7729", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Created project TST")
	testutil.Contains(t, stdout.String(), "Test Project")
}

func TestNewDeleteCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newDeleteCmd(opts)

	testutil.Equal(t, cmd.Use, "delete <project-key>")

	forceFlag := cmd.Flags().Lookup("force")
	testutil.NotNil(t, forceFlag)
	testutil.Equal(t, forceFlag.DefValue, "false")
}

func TestRunDelete_Force(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runDelete(context.Background(), opts, "TST", true)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Deleted project TST")
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

	err = runDelete(context.Background(), opts, "TST", false)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Deletion cancelled")
}

func TestRunDelete_NoForce_Accepted(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
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

	err = runDelete(context.Background(), opts, "TST", false)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Deleted project TST")
}

func TestRunUpdate(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPut)
		_ = json.NewEncoder(w).Encode(api.ProjectDetail{
			ID:   json.Number("10001"),
			Key:  "TST",
			Name: "Updated Name",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runUpdate(context.Background(), opts, "TST", "Updated Name", "", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Updated project TST")
}

func TestRunRestore(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectDetail{
			ID:   json.Number("10001"),
			Key:  "TST",
			Name: "Test Project",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runRestore(context.Background(), opts, "TST")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Restored project TST")
}

func TestRunTypes(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.ProjectType{
			{Key: "software", FormattedKey: "Software"},
			{Key: "business", FormattedKey: "Business"},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runTypes(context.Background(), opts, "")
	testutil.RequireNoError(t, err)
	out := stdout.String()
	testutil.Contains(t, out, "KEY | NAME")
	testutil.Contains(t, out, "software | Software")
}

func TestRunTypes_Extended_AddsDescriptionKey(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.ProjectType{
			{Key: "software", FormattedKey: "Software", DescriptionI18nKey: "jira.project.type.software.description"},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Extended: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runTypes(context.Background(), opts, ""))
	out := stdout.String()
	testutil.Contains(t, out, "KEY | NAME | DESCRIPTION_KEY")
	testutil.Contains(t, out, "software | Software | jira.project.type.software.description")
}

func TestRunTypes_IDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.ProjectType{
			{Key: "software", FormattedKey: "Software"},
			{Key: "business", FormattedKey: "Business"},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", IDOnly: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runTypes(context.Background(), opts, ""))
	testutil.Equal(t, stdout.String(), "software\nbusiness\n")
}
