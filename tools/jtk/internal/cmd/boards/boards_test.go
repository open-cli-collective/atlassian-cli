package boards

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

func TestNewListCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newListCmd(opts)

	testutil.Equal(t, cmd.Use, "list")
	testutil.NotEmpty(t, cmd.Short)

	projectFlag := cmd.Flags().Lookup("project")
	testutil.NotNil(t, projectFlag)
	testutil.Equal(t, projectFlag.DefValue, "")

	maxFlag := cmd.Flags().Lookup("max")
	testutil.NotNil(t, maxFlag)
	testutil.Equal(t, maxFlag.DefValue, "50")

	nextPageTokenFlag := cmd.Flags().Lookup("next-page-token")
	testutil.NotNil(t, nextPageTokenFlag)

	fieldsFlag := cmd.Flags().Lookup("fields")
	testutil.NotNil(t, fieldsFlag)
}

func TestRunList_Table(t *testing.T) {
	// NOT t.Parallel(): isolates the package-global cache so the boards
	// lookup is a guaranteed miss and the mock server is exercised (was
	// latently non-hermetic — read the dev's real ~/.jtk cache).
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{
			Values: []api.Board{
				{
					ID:   1,
					Name: "Team Alpha Board",
					Type: "scrum",
					Location: api.BoardLocation{
						ProjectID:  10001,
						ProjectKey: "ALPHA",
					},
				},
				{
					ID:   2,
					Name: "Team Beta Board",
					Type: "kanban",
					Location: api.BoardLocation{
						ProjectID:  10002,
						ProjectKey: "BETA",
					},
				},
			},
			Total:  2,
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "", "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Team Alpha Board")
	testutil.Contains(t, output, "Team Beta Board")
	testutil.Contains(t, output, "ALPHA")
	testutil.Contains(t, output, "BETA")
	testutil.Contains(t, output, "scrum")
	testutil.Contains(t, output, "kanban")
}

func TestRunList_IDOnly(t *testing.T) {
	// NOT t.Parallel(): cache isolation (see TestRunList_Table).
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{
			Values: []api.Board{
				{ID: 1, Name: "Board A"},
				{ID: 2, Name: "Board B"},
			},
			Total:  2,
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "", "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Equal(t, output, "1\n2\n")
}

func TestRunList_Empty(t *testing.T) {
	// NOT t.Parallel(): cache isolation (see TestRunList_Table).
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{
			Values: []api.Board{},
			Total:  0,
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout, stderr bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &stderr}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "", "")
	testutil.RequireNoError(t, err)

	combined := stdout.String() + stderr.String()
	testutil.Contains(t, combined, "No boards found")
}

func TestRunList_ResolvesProjectByName(t *testing.T) {
	// NOT t.Parallel(): SetRootForTest / SetInstanceKeyForTest mutate package
	// globals in the cache package.
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	testutil.RequireNoError(t, cache.WriteResource("projects", "24h", []api.Project{
		{Key: "PLAT", Name: "Platform"},
	}))

	var capturedProject string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedProject = r.URL.Query().Get("projectKeyOrId")
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{
			IsLast: true, Values: []api.Board{{ID: 1, Name: "B", Type: "scrum"}},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "Platform", 50, "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, capturedProject, "PLAT")
}

func TestRunList_ProjectKeyShapePassesThrough(t *testing.T) {
	// NOT t.Parallel(): see the comment on TestRunList_ResolvesProjectByName.
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))

	var capturedProject string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedProject = r.URL.Query().Get("projectKeyOrId")
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{IsLast: true, Values: []api.Board{}})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "UNCACHED", 50, "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, capturedProject, "UNCACHED")
}

func TestRunGet_Table(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Board{
			ID:   42,
			Name: "Sprint Board",
			Type: "scrum",
			Location: api.BoardLocation{
				ProjectKey:  "PROJ",
				ProjectName: "My Project",
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	resolvedBoard := &api.Board{ID: 42, Name: "Sprint Board"}
	err = runGet(context.Background(), opts, client, resolvedBoard, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "42")
	testutil.Contains(t, output, "Sprint Board")
	testutil.Contains(t, output, "scrum")
	testutil.Contains(t, output, "PROJ")
}

func TestRunGet_IDOnly(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("API should not be called in --id mode")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	resolvedBoard := &api.Board{ID: 42, Name: "Sprint Board"}
	err = runGet(context.Background(), opts, client, resolvedBoard, "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "42\n")
}

func TestRunList_InvalidNextPageToken(t *testing.T) {
	t.Parallel()
	opts := &root.Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}

	err := runList(context.Background(), opts, "", 50, "abc", "")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "--next-page-token")
}

func TestRunList_Pagination(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{
			Values: []api.Board{{ID: 1, Name: "Board A", Type: "scrum"}},
			IsLast: false,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 1, "", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "More results available (next: 1)")
}

func TestRunGet_NameFallback(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// API returns board without name
		_ = json.NewEncoder(w).Encode(api.Board{
			ID: 42, Type: "scrum",
			Location: api.BoardLocation{ProjectKey: "PROJ"},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	// Resolved board has name from cache, but API response lacks it
	resolvedBoard := &api.Board{ID: 42, Name: "Cached Name"}
	err = runGet(context.Background(), opts, client, resolvedBoard, "")
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "Cached Name")
}

func TestRunGet_ResolvesBoardByName(t *testing.T) {
	// NOT t.Parallel(): cache globals
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))
	testutil.RequireNoError(t, cache.WriteResource("boards", "24h", []api.Board{
		{ID: 42, Name: "MON board", Type: "scrum"},
	}))

	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(api.Board{
			ID: 42, Name: "MON board", Type: "scrum",
			Location: api.BoardLocation{ProjectKey: "MON", ProjectName: "Platform Development"},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	rootCmd, opts := root.NewCmd()
	opts.SetAPIClient(client)
	opts.Stdout = &bytes.Buffer{}
	opts.Stderr = &bytes.Buffer{}
	Register(rootCmd, opts)

	rootCmd.SetArgs([]string{"boards", "get", "MON board"})
	err = rootCmd.Execute()
	testutil.RequireNoError(t, err)
	testutil.Equal(t, capturedPath, "/rest/agile/1.0/board/42")
}

func TestRunGet_MissingArg(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	rootCmd, opts := root.NewCmd()
	opts.SetAPIClient(client)
	Register(rootCmd, opts)

	rootCmd.SetArgs([]string{"boards", "get"})
	err = rootCmd.Execute()
	testutil.NotNil(t, err)
}

func TestRunList_FreshCacheSkipsLive(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, cache.WriteResource("boards", "24h", []api.Board{
		{ID: 1, Name: "Board One", Type: "scrum", Location: api.BoardLocation{ProjectKey: "PROJ"}},
		{ID: 2, Name: "Board Two", Type: "kanban", Location: api.BoardLocation{ProjectKey: "OTHER"}},
	}))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when boards cache is fresh")
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Board One")
	testutil.Contains(t, stdout.String(), "Board Two")
}

func TestRunList_FreshCacheSkipsLive_IDOnly(t *testing.T) {
	t.Cleanup(cache.SetRootForTest(t.TempDir()))
	t.Cleanup(cache.SetInstanceKeyForTest("test.atlassian.net"))

	testutil.RequireNoError(t, cache.WriteResource("boards", "24h", []api.Board{
		{ID: 1, Name: "Board One", Type: "scrum"},
		{ID: 2, Name: "Board Two", Type: "kanban"},
	}))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("live API must not be called when boards cache is fresh")
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "", "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "1\n2\n")
}
