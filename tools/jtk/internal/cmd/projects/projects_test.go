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

func TestRunList_Table(t *testing.T) {
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
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "TST")
	testutil.Contains(t, stdout.String(), "Test")
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

	err = runList(context.Background(), opts, "", 50)
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

	err = runList(context.Background(), opts, "", 50)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No projects found")
}

func TestNewGetCmd(t *testing.T) {
	t.Parallel()
	opts := &root.Options{}
	cmd := newGetCmd(opts)

	testutil.Equal(t, cmd.Use, "get <project-key>")
}

func TestRunGet_Table(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.ProjectDetail{
			ID:             json.Number("10001"),
			Key:            "TST",
			Name:           "Test",
			ProjectTypeKey: "software",
			Lead:           &api.User{DisplayName: "Lead"},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(context.Background(), opts, "TST")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "TST")
	testutil.Contains(t, stdout.String(), "Lead")
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
	t.Parallel()
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

	err = runCreate(context.Background(), opts, "TST", "Test Project", "software", "abc123", "")
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

	err = runTypes(context.Background(), opts)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "software")
	testutil.Contains(t, stdout.String(), "Software")
}
