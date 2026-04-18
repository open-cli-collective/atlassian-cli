package projects

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	testutil.RequireNoError(t, runList(context.Background(), opts, "", 50, "", ""))

	want := "KEY | TYPE | LEAD | NAME\nTST | software | Lead | Test\n"
	if stdout.String() != want {
		t.Errorf("projects list default:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRunList_Extended_MatchesSpecShape(t *testing.T) {
	t.Parallel()
	// Per #230: extended headers are KEY|TYPE|STYLE|LEAD|ISSUE_TYPES|
	// COMPONENTS|NAME with ISSUE_TYPES rendered as comma-joined names and
	// COMPONENTS as a count.
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

	want := "KEY | TYPE | STYLE | LEAD | ISSUE_TYPES | COMPONENTS | NAME\n" +
		"TST | software | classic | Lead | Epic, SDLC | 2 | Test\n"
	if stdout.String() != want {
		t.Errorf("projects list --extended:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
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

func TestRunList_Fields_JSONReturnsError(t *testing.T) {
	t.Parallel()
	// `--fields` + `-o json` is explicitly rejected (parity with users search
	// / issues list). The guard in runList must fire before hitting the
	// server so the user gets a predictable error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectSearchResponse{Values: []api.ProjectDetail{}, IsLast: true})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "", "KEY")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "--fields is not supported")
}

func TestRunList_NextPageToken_RejectsNegative(t *testing.T) {
	t.Parallel()
	// strconv.Atoi parses "-1" successfully; the n < 0 guard must still
	// reject it. A parallel test lives in users_test.go since the helper is
	// duplicated per package.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectSearchResponse{Values: []api.ProjectDetail{}, IsLast: true})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "-5", "")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "invalid --next-page-token")
	testutil.Contains(t, err.Error(), "non-negative")
}

func TestRunGet_Fields_JSONReturnsError(t *testing.T) {
	t.Parallel()
	// Mirrors TestRunGet_Fields_JSONReturnsError in users_test.go — the
	// errFieldsWithJSON guard in runGet was introduced for this migration
	// and should not silently accept both flags.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectDetail{Key: "TST", Name: "Test"})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "TST", "NAME")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "--fields is not supported")
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

	want := "KEY | NAME\nTST | Test\n"
	if stdout.String() != want {
		t.Errorf("projects list --fields KEY,NAME:\ngot:  %q\nwant: %q", stdout.String(), want)
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

	want := "TST  Test\n" +
		"Type: software   Lead: Lead (u1)   Style: classic\n" +
		"Issue Types: Epic (1)\n" +
		"Components: 2\n" +
		"  c1 | A\n" +
		"  c2 | B\n" +
		"Versions: 0\n" +
		"Simplified: no   Private: no\n"
	if stdout.String() != want {
		t.Errorf("projects get --extended:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRunGet_Extended_ComponentListTruncatesAtLimit(t *testing.T) {
	t.Parallel()
	// The presenter has a unit test for the `... [N more]` truncation;
	// this locks the same shape at the command layer so rendered output is
	// covered end-to-end.
	simplified := false
	private := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectDetail{
			ID: json.Number("10001"), Key: "TST", Name: "Test",
			ProjectTypeKey: "software", Style: "classic",
			Lead:       &api.User{AccountID: "u1", DisplayName: "Lead"},
			IssueTypes: []api.IssueType{{ID: "1", Name: "Epic"}},
			Components: []api.Component{
				{ID: "c1", Name: "A"},
				{ID: "c2", Name: "B"},
				{ID: "c3", Name: "C"},
				{ID: "c4", Name: "D"},
				{ID: "c5", Name: "E"},
				{ID: "c6", Name: "F"},
			},
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

	want := "TST  Test\n" +
		"Type: software   Lead: Lead (u1)   Style: classic\n" +
		"Issue Types: Epic (1)\n" +
		"Components: 6\n" +
		"  c1 | A\n" +
		"  c2 | B\n" +
		"  c3 | C\n" +
		"  c4 | D\n" +
		"  ... [2 more]\n" +
		"Versions: 0\n" +
		"Simplified: no   Private: no\n"
	if stdout.String() != want {
		t.Errorf("projects get --extended (>limit components):\ngot:  %q\nwant: %q", stdout.String(), want)
	}
}

func TestRunGet_Default_IssueTypesRowPresentEvenWhenEmpty(t *testing.T) {
	t.Parallel()
	// Command-level Fix 2 regression. The reviewer's original finding was a
	// rendered-output contract issue, so we lock this at the command layer
	// alongside the presenter-layer test.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectDetail{
			ID: json.Number("10001"), Key: "EMPTY", Name: "Empty",
			ProjectTypeKey: "software", Style: "classic",
			Lead: &api.User{DisplayName: "Lead"},
			// No IssueTypes, no Components, no Versions.
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", NoColor: true, Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	testutil.RequireNoError(t, runGet(context.Background(), opts, "EMPTY", ""))

	want := "EMPTY  Empty\n" +
		"Type: software   Lead: Lead   Style: classic\n" +
		"Issue Types: -\n" +
		"Components: 0   Versions: 0\n"
	if stdout.String() != want {
		t.Errorf("projects get default (no issue types):\ngot:  %q\nwant: %q", stdout.String(), want)
	}
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

	// KEY is the identity column; projection.Resolve always prepends it.
	want := "KEY: TST\nNAME: Test\nLEAD: Lead\n"
	if stdout.String() != want {
		t.Errorf("projects get --fields NAME,LEAD:\ngot:  %q\nwant: %q", stdout.String(), want)
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

	testutil.RequireNoError(t, runTypes(context.Background(), opts, ""))

	want := "KEY | NAME\nsoftware | Software\nbusiness | Business\n"
	if stdout.String() != want {
		t.Errorf("projects types default:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
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

	want := "KEY | NAME | DESCRIPTION_KEY\nsoftware | Software | jira.project.type.software.description\n"
	if stdout.String() != want {
		t.Errorf("projects types --extended:\ngot:  %q\nwant: %q", stdout.String(), want)
	}
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
