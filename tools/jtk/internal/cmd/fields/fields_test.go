package fields

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

	cmd, _, err := rootCmd.Find([]string{"fields"})
	require.NoError(t, err)
	assert.Equal(t, "fields", cmd.Name())
	assert.Equal(t, []string{"field", "f"}, cmd.Aliases)
}

func TestNewListCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newListCmd(opts)

	assert.Equal(t, "list", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	customFlag := cmd.Flags().Lookup("custom")
	require.NotNil(t, customFlag)
	assert.Equal(t, "false", customFlag.DefValue)
}

func TestRunList_Table(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.Field{
			{ID: "summary", Name: "Summary", Schema: api.FieldSchema{Type: "string"}},
			{ID: "customfield_10100", Name: "Environment", Custom: true, Schema: api.FieldSchema{Type: "option"}},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(opts, false)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "summary")
	assert.Contains(t, stdout.String(), "customfield_10100")
	assert.Contains(t, stdout.String(), "Environment")
}

func TestRunList_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.Field{
			{ID: "customfield_10100", Name: "Environment", Custom: true},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(opts, false)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), `"id"`)
	assert.Contains(t, stdout.String(), "customfield_10100")
}

func TestRunList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.Field{})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(opts, false)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "No fields found")
}

func TestNewCreateCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newCreateCmd(opts)

	assert.Equal(t, "create", cmd.Use)

	nameFlag := cmd.Flags().Lookup("name")
	require.NotNil(t, nameFlag)

	typeFlag := cmd.Flags().Lookup("type")
	require.NotNil(t, typeFlag)

	descFlag := cmd.Flags().Lookup("description")
	require.NotNil(t, descFlag)
}

func TestRunCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.Field{
			ID:     "customfield_10100",
			Name:   "Environment",
			Custom: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(opts, "Environment", "com.atlassian.jira.plugin.system.customfieldtypes:select", "")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Created field customfield_10100")
	assert.Contains(t, stdout.String(), "Environment")
}

func TestRunCreate_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.Field{
			ID:     "customfield_10100",
			Name:   "Environment",
			Custom: true,
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "json", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runCreate(opts, "Environment", "select", "")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "customfield_10100")
}

func TestNewDeleteCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newDeleteCmd(opts)

	assert.Equal(t, "delete <field-id>", cmd.Use)

	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	assert.Equal(t, "false", forceFlag.DefValue)
}

func TestRunDelete_Force(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/trash")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runDelete(opts, "customfield_10100", true)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Trashed field customfield_10100")
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

	err = runDelete(opts, "customfield_10100", false)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Deletion cancelled")
}

func TestRunDelete_NoForce_Accepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
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

	err = runDelete(opts, "customfield_10100", false)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Trashed field customfield_10100")
}

func TestRunRestore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/restore")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runRestore(opts, "customfield_10100")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Restored field customfield_10100")
}

// --- Contexts tests ---

func TestNewContextsCmd(t *testing.T) {
	rootCmd, opts := root.NewCmd()
	Register(rootCmd, opts)

	cmd, _, err := rootCmd.Find([]string{"fields", "contexts"})
	require.NoError(t, err)
	assert.Equal(t, "contexts", cmd.Name())
	assert.Equal(t, []string{"context", "ctx"}, cmd.Aliases)
}

func TestRunContextsList_Table(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.FieldContextsResponse{
			Values: []api.FieldContext{
				{ID: "10001", Name: "Default", IsGlobalContext: true, IsAnyIssueType: true},
				{ID: "10002", Name: "Bug Context", IsGlobalContext: false, IsAnyIssueType: false},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runContextsList(opts, "customfield_10100")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Default")
	assert.Contains(t, stdout.String(), "Bug Context")
}

func TestRunContextsList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.FieldContextsResponse{Values: []api.FieldContext{}})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runContextsList(opts, "customfield_10100")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "No contexts found")
}

func TestRunContextsCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.FieldContext{
			ID:   "10003",
			Name: "Bug Context",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runContextsCreate(opts, "customfield_10100", "Bug Context", "")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Created context 10003")
	assert.Contains(t, stdout.String(), "Bug Context")
}

func TestRunContextsDelete_Force(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runContextsDelete(opts, "customfield_10100", "10003", true)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Deleted context 10003")
}

func TestRunContextsDelete_NoForce_Declined(t *testing.T) {
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

	err = runContextsDelete(opts, "customfield_10100", "10003", false)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Deletion cancelled")
}

// --- Options tests ---

func TestNewOptionsCmd(t *testing.T) {
	rootCmd, opts := root.NewCmd()
	Register(rootCmd, opts)

	cmd, _, err := rootCmd.Find([]string{"fields", "options"})
	require.NoError(t, err)
	assert.Equal(t, "options", cmd.Name())
	assert.Equal(t, []string{"option", "opt"}, cmd.Aliases)
}

func TestResolveContextID_Explicit(t *testing.T) {
	// When context flag is provided, it should be used directly
	id, err := resolveContextID(nil, "customfield_10100", "10001")
	require.NoError(t, err)
	assert.Equal(t, "10001", id)
}

func TestResolveContextID_AutoDetect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.FieldContextsResponse{
			Values: []api.FieldContext{
				{ID: "10001", Name: "Default"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	id, err := resolveContextID(client, "customfield_10100", "")
	require.NoError(t, err)
	assert.Equal(t, "10001", id)
}

func TestRunOptionsList_Table(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// GetFieldContexts (auto-detect)
			json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		// GetFieldContextOptions
		json.NewEncoder(w).Encode(api.FieldContextOptionsResponse{
			Values: []api.FieldContextOption{
				{ID: "1", Value: "Production", Disabled: false},
				{ID: "2", Value: "Staging", Disabled: true},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runOptionsList(opts, "customfield_10100", "")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Production")
	assert.Contains(t, stdout.String(), "Staging")
}

func TestRunOptionsList_Empty(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		json.NewEncoder(w).Encode(api.FieldContextOptionsResponse{Values: []api.FieldContextOption{}})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runOptionsList(opts, "customfield_10100", "")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "No options found")
}

func TestRunOptionsAdd(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		assert.Equal(t, http.MethodPost, r.Method)
		json.NewEncoder(w).Encode(api.FieldContextOptionsResponse{
			Values: []api.FieldContextOption{
				{ID: "3", Value: "Option A"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runOptionsAdd(opts, "customfield_10100", "Option A", "")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Added option 3")
	assert.Contains(t, stdout.String(), "Option A")
}

func TestRunOptionsUpdate(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		assert.Equal(t, http.MethodPut, r.Method)
		json.NewEncoder(w).Encode(api.FieldContextOptionsResponse{
			Values: []api.FieldContextOption{
				{ID: "3", Value: "Option A (updated)"},
			},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runOptionsUpdate(opts, "customfield_10100", "3", "Option A (updated)", "")
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Updated option 3")
}

func TestRunOptionsDelete_Force(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode(api.FieldContextsResponse{
				Values: []api.FieldContext{{ID: "10001", Name: "Default"}},
			})
			return
		}
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runOptionsDelete(opts, "customfield_10100", "3", "", true)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Deleted option 3")
}

func TestRunOptionsDelete_NoForce_Declined(t *testing.T) {
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

	err = runOptionsDelete(opts, "customfield_10100", "3", "", false)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Deletion cancelled")
}
