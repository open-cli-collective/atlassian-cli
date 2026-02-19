package fields

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

	customFlag := cmd.Flags().Lookup("custom")
	testutil.NotNil(t, customFlag)
	testutil.Equal(t, customFlag.DefValue, "false")
}

func TestRunList_Table(t *testing.T) {
	t.Parallel()
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

	err = runList(context.Background(), opts, false)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "summary")
	testutil.Contains(t, stdout.String(), "customfield_10100")
	testutil.Contains(t, stdout.String(), "Environment")
}

func TestRunList_JSON(t *testing.T) {
	t.Parallel()
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

	err = runList(context.Background(), opts, false)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), `"id"`)
	testutil.Contains(t, stdout.String(), "customfield_10100")
}

func TestRunList_Empty(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Field{})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, false)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "No fields found")
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
	testutil.Contains(t, stdout.String(), "Created field customfield_10100")
	testutil.Contains(t, stdout.String(), "Environment")
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
	testutil.Contains(t, stdout.String(), "Trashed field customfield_10100")
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
	testutil.Contains(t, stdout.String(), "Deletion cancelled")
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
	testutil.Contains(t, stdout.String(), "Trashed field customfield_10100")
}

func TestRunRestore(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		testutil.Contains(t, r.URL.Path, "/restore")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "test@test.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Output: "table", Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runRestore(context.Background(), opts, "customfield_10100")
	testutil.RequireNoError(t, err)
	testutil.Contains(t, stdout.String(), "Restored field customfield_10100")
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
	testutil.Contains(t, stdout.String(), "Created context 10003")
	testutil.Contains(t, stdout.String(), "Bug Context")
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
		// GetFieldContextOptions
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
		_ = json.NewEncoder(w).Encode(api.FieldContextOptionsResponse{
			Values: []api.FieldContextOption{
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
	testutil.Contains(t, stdout.String(), "Added option 3")
	testutil.Contains(t, stdout.String(), "Option A")
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
		_ = json.NewEncoder(w).Encode(api.FieldContextOptionsResponse{
			Values: []api.FieldContextOption{
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
	testutil.Contains(t, stdout.String(), "Updated option 3")
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
	testutil.Contains(t, stdout.String(), "Deleted option 3")
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
	testutil.Contains(t, stdout.String(), "Deletion cancelled")
}
