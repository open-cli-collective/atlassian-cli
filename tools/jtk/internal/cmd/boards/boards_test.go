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

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Team Alpha Board")
	testutil.Contains(t, output, "Team Beta Board")
	testutil.Contains(t, output, "ALPHA")
	testutil.Contains(t, output, "BETA")
	testutil.Contains(t, output, "scrum")
	testutil.Contains(t, output, "kanban")
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

	err = runList(context.Background(), opts, "", 50)
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

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)

	combined := stdout.String() + stderr.String()
	testutil.Contains(t, combined, "No boards found")
}

func TestRunGet_Table(t *testing.T) {
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
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, 42)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "42")
	testutil.Contains(t, output, "Sprint Board")
	testutil.Contains(t, output, "scrum")
	testutil.Contains(t, output, "PROJ")
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

	err = runGet(context.Background(), opts, 42)
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, `"name"`)
	testutil.Contains(t, output, "Sprint Board")
}

func TestRunGet_InvalidID(t *testing.T) {
	t.Parallel()
	rootCmd, opts := root.NewCmd()
	Register(rootCmd, opts)

	rootCmd.SetArgs([]string{"boards", "get", "abc"})
	err := rootCmd.Execute()
	testutil.NotNil(t, err)
	testutil.Contains(t, err.Error(), "invalid board ID")
}
