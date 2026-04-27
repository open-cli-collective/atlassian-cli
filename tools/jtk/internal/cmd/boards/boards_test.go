package boards

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
	t.Parallel()
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
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
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

func TestRunList_Extended(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{
			Values: []api.Board{
				{
					ID:   23,
					Name: "MON board",
					Type: "scrum",
					Location: api.BoardLocation{
						ProjectKey:  "MON",
						ProjectName: "Platform Development",
					},
				},
			},
			Total:  1,
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "", "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "PROJECT_NAME")
	testutil.Contains(t, output, "Platform Development")
}

func TestRunList_IDOnly(t *testing.T) {
	t.Parallel()
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
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "", "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Equal(t, output, "1\n2\n")
}

func TestRunList_JSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{
			Values: []api.Board{
				{
					ID:   1,
					Name: "Team Alpha Board",
					Type: "scrum",
					Location: api.BoardLocation{
						ProjectKey: "ALPHA",
					},
				},
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

	output := stdout.String()
	testutil.Contains(t, output, `"name"`)
	testutil.Contains(t, output, "Team Alpha Board")
}

func TestRunList_Empty(t *testing.T) {
	t.Parallel()
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
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &stderr}
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
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
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

	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
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
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
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
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	resolvedBoard := &api.Board{ID: 42, Name: "Sprint Board"}
	err = runGet(context.Background(), opts, client, resolvedBoard, "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, stdout.String(), "42\n")
}

func TestRunGet_JSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Board{
			ID:   42,
			Name: "Sprint Board",
			Type: "scrum",
			Location: api.BoardLocation{
				ProjectKey: "PROJ",
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	resolvedBoard := &api.Board{ID: 42, Name: "Sprint Board"}
	err = runGet(context.Background(), opts, client, resolvedBoard, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, `"name"`)
	testutil.Contains(t, output, "Sprint Board")
}

func TestRunList_FieldsWithJSONError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.BoardsResponse{IsLast: true, Values: []api.Board{{ID: 1}}})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Output: "json", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50, "", "ID,NAME")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "--fields is not supported")
}

func TestRunList_InvalidNextPageToken(t *testing.T) {
	t.Parallel()
	opts := &root.Options{Output: "table", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}

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
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 1, "", "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "More results available (next: 1)")
}

func TestRunGet_Extended(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	requestPaths := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestPaths = append(requestPaths, r.URL.Path)
		mu.Unlock()
		if strings.Contains(r.URL.Path, "/configuration") {
			_ = json.NewEncoder(w).Encode(api.BoardConfiguration{
				ID:     42,
				Name:   "Sprint Board",
				Filter: api.BoardFilter{ID: "100", Name: "my filter"},
				ColumnConfig: api.BoardColumnConfig{
					Columns: []api.BoardColumn{{Name: "Backlog"}, {Name: "Done"}},
				},
			})
		} else {
			_ = json.NewEncoder(w).Encode(api.Board{
				ID: 42, Name: "Sprint Board", Type: "scrum",
				Location: api.BoardLocation{ProjectKey: "PROJ", ProjectName: "My Project"},
			})
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	resolvedBoard := &api.Board{ID: 42, Name: "Sprint Board"}
	err = runGet(context.Background(), opts, client, resolvedBoard, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Filter: my filter (id: 100)")
	testutil.Contains(t, output, "Column config: Backlog, Done")
	// Verify both board and configuration endpoints were hit
	mu.Lock()
	pathCount := len(requestPaths)
	mu.Unlock()
	if pathCount < 2 {
		t.Errorf("expected 2 API calls (board + config), got %d", pathCount)
	}
}

func TestRunGet_Extended_EmptyFilterName(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/configuration") {
			_ = json.NewEncoder(w).Encode(api.BoardConfiguration{
				ID:     42,
				Name:   "Sprint Board",
				Filter: api.BoardFilter{ID: "10084", Name: ""},
				ColumnConfig: api.BoardColumnConfig{
					Columns: []api.BoardColumn{
						{Name: "Backlog"},
						{Name: "Ready for Development"},
						{Name: "In Development"},
					},
				},
			})
		} else {
			_ = json.NewEncoder(w).Encode(api.Board{
				ID: 42, Name: "Sprint Board", Type: "scrum",
				Location: api.BoardLocation{ProjectKey: "PROJ", ProjectName: "My Project"},
			})
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	resolvedBoard := &api.Board{ID: 42, Name: "Sprint Board"}
	err = runGet(context.Background(), opts, client, resolvedBoard, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Filter: id: 10084")
	if strings.Contains(output, "Filter:  (id:") {
		t.Errorf("filter should not have leading space before (id:): %q", output)
	}
	testutil.Contains(t, output, "Column config: Backlog, Ready for Development, In Development")
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
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	// Resolved board has name from cache, but API response lacks it
	resolvedBoard := &api.Board{ID: 42, Name: "Cached Name"}
	err = runGet(context.Background(), opts, client, resolvedBoard, "")
	testutil.RequireNoError(t, err)

	testutil.Contains(t, stdout.String(), "Cached Name")
}

func TestRunGet_FieldsWithJSONError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Board{ID: 42, Name: "B", Type: "scrum"})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	opts := &root.Options{Output: "json", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	resolvedBoard := &api.Board{ID: 42}
	err = runGet(context.Background(), opts, client, resolvedBoard, "ID,NAME")
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "--fields is not supported")
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

func TestRunGet_Extended_FilterNameAlreadyPresent_NoExtraFetch(t *testing.T) {
	t.Parallel()
	var filterFetched bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/filter/"):
			filterFetched = true
			w.WriteHeader(http.StatusInternalServerError)
		case strings.Contains(r.URL.Path, "/configuration"):
			_ = json.NewEncoder(w).Encode(api.BoardConfiguration{
				Filter: api.BoardFilter{ID: "100", Name: "already set"},
				ColumnConfig: api.BoardColumnConfig{
					Columns: []api.BoardColumn{{Name: "Done"}},
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(api.Board{
				ID: 23, Name: "B", Type: "scrum",
				Location: api.BoardLocation{ProjectKey: "MON"},
			})
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, client, &api.Board{ID: 23, Name: "B"}, "")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Filter: already set (id: 100)")
	if filterFetched {
		t.Error("filter endpoint should not be called when name is already present")
	}
}

func TestRunGet_Extended_FilterNameResolved(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/configuration"):
			_ = json.NewEncoder(w).Encode(api.BoardConfiguration{
				ID:     23,
				Name:   "MON board",
				Filter: api.BoardFilter{ID: "10026", Name: ""},
				ColumnConfig: api.BoardColumnConfig{
					Columns: []api.BoardColumn{{Name: "Ready"}, {Name: "Done"}},
				},
			})
		case strings.Contains(r.URL.Path, "/filter/"):
			_, _ = w.Write([]byte(`{"id":"10026","name":"MON Aggregate"}`))
		default:
			_ = json.NewEncoder(w).Encode(api.Board{
				ID: 23, Name: "MON board", Type: "scrum",
				Location: api.BoardLocation{ProjectKey: "MON", ProjectName: "Platform Development"},
			})
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	resolvedBoard := &api.Board{ID: 23, Name: "MON board"}
	err = runGet(context.Background(), opts, client, resolvedBoard, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Filter: MON Aggregate (id: 10026)")
}

func TestRunGet_Extended_FilterNameFallback(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/configuration"):
			_ = json.NewEncoder(w).Encode(api.BoardConfiguration{
				ID:     23,
				Name:   "MON board",
				Filter: api.BoardFilter{ID: "10026", Name: ""},
				ColumnConfig: api.BoardColumnConfig{
					Columns: []api.BoardColumn{{Name: "Ready"}},
				},
			})
		case strings.Contains(r.URL.Path, "/filter/"):
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"errorMessages":["You do not have permission"]}`))
		default:
			_ = json.NewEncoder(w).Encode(api.Board{
				ID: 23, Name: "MON board", Type: "scrum",
				Location: api.BoardLocation{ProjectKey: "MON", ProjectName: "Platform Development"},
			})
		}
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}, Extended: true}
	opts.SetAPIClient(client)

	resolvedBoard := &api.Board{ID: 23, Name: "MON board"}
	err = runGet(context.Background(), opts, client, resolvedBoard, "")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Filter: id: 10026")
}
