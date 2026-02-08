package projects

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRegister(t *testing.T) {
	rootCmd, opts := root.NewCmd()
	Register(rootCmd, opts)

	cmd, _, err := rootCmd.Find([]string{"projects"})
	require.NoError(t, err)
	assert.Equal(t, "projects", cmd.Name())
	assert.Equal(t, []string{"project", "proj", "p"}, cmd.Aliases)
}

func TestNewListCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newListCmd(opts)

	assert.Equal(t, "list", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	queryFlag := cmd.Flags().Lookup("query")
	require.NotNil(t, queryFlag)
	assert.Equal(t, "", queryFlag.DefValue)

	maxFlag := cmd.Flags().Lookup("max")
	require.NotNil(t, maxFlag)
	assert.Equal(t, "50", maxFlag.DefValue)
}

func TestRunList_Table(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.ProjectSearchResponse{
			Values: []api.ProjectDetail{
				{Key: "TST", Name: "Test", ProjectTypeKey: "software", Lead: &api.User{DisplayName: "Lead"}},
			},
			Total:  1,
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(opts, "", 50)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "TST")
	assert.Contains(t, stdout.String(), "Test")
}

func TestRunList_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.ProjectSearchResponse{
			Values: []api.ProjectDetail{
				{Key: "TST", Name: "Test", ProjectTypeKey: "software"},
			},
			Total:  1,
			IsLast: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(opts, "", 50)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), `"key"`)
	assert.Contains(t, stdout.String(), "TST")
}

func TestRunList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.ProjectSearchResponse{Values: []api.ProjectDetail{}, Total: 0, IsLast: true})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(opts, "", 50)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "No projects found")
}

func TestNewGetCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newGetCmd(opts)

	assert.Equal(t, "get <project-key>", cmd.Use)
}

func TestRunGet_Table(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.ProjectDetail{
			ID:             "10001",
			Key:            "TST",
			Name:           "Test",
			ProjectTypeKey: "software",
			Lead:           &api.User{DisplayName: "Lead"},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runGet(opts, "TST")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "TST")
	assert.Contains(t, stdout.String(), "Lead")
}

func TestNewCreateCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newCreateCmd(opts)

	assert.Equal(t, "create", cmd.Use)

	keyFlag := cmd.Flags().Lookup("key")
	require.NotNil(t, keyFlag)

	nameFlag := cmd.Flags().Lookup("name")
	require.NotNil(t, nameFlag)

	leadFlag := cmd.Flags().Lookup("lead")
	require.NotNil(t, leadFlag)
}

func TestRunCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.ProjectDetail{
			ID:   "10001",
			Key:  "TST",
			Name: "Test Project",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(opts, "TST", "Test Project", "software", "abc123", "")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Created project TST")
}

func TestNewDeleteCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newDeleteCmd(opts)

	assert.Equal(t, "delete <project-key>", cmd.Use)

	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	assert.Equal(t, "false", forceFlag.DefValue)
}

func TestRunDelete_Force(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runDelete(opts, "TST", true)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Deleted project TST")
}

func TestRunDelete_NoForce_Declined(t *testing.T) {
	client, err := api.New(api.ClientConfig{URL: "https://test.atlassian.net", Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Stdin:  bytes.NewBufferString("n\n"),
	}
	opts.SetAPIClient(client)

	err = runDelete(opts, "TST", false)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Deletion cancelled")
}

func TestRunDelete_NoForce_Accepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Stdin:  bytes.NewBufferString("y\n"),
	}
	opts.SetAPIClient(client)

	err = runDelete(opts, "TST", false)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Deleted project TST")
}

func TestRunUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		json.NewEncoder(w).Encode(api.ProjectDetail{
			ID:   "10001",
			Key:  "TST",
			Name: "Updated Name",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runUpdate(opts, "TST", "Updated Name", "", "")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Updated project TST")
}

func TestRunRestore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.ProjectDetail{
			ID:   "10001",
			Key:  "TST",
			Name: "Test Project",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runRestore(opts, "TST")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Restored project TST")
}

func TestRunTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.ProjectType{
			{Key: "software", FormattedKey: "Software"},
			{Key: "business", FormattedKey: "Business"},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runTypes(opts)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "software")
	assert.Contains(t, stdout.String(), "Software")
}
