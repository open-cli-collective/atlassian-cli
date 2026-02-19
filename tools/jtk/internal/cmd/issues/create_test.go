package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunCreate_RequestBodyNoDoubleQuoting(t *testing.T) {
	t.Parallel()
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Issue{
				Key: "TEST-1",
				ID:  "10001",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "MYPROJECT", "Task", "Fix login bug", "Users cannot log in with SSO credentials", "", "", nil)
	testutil.RequireNoError(t, err)

	// Parse the captured request body
	testutil.NotEmpty(t, capturedBody)

	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)

	// Summary must be the exact string without extra quotes
	summary := fields["summary"].(string)
	testutil.Equal(t, summary, "Fix login bug")
	testutil.NotContains(t, summary, `"`)

	// Description should be ADF format, extract text from first paragraph
	desc := fields["description"].(map[string]any)
	testutil.Equal(t, desc["type"], "doc")
	content := desc["content"].([]any)
	testutil.NotEmpty(t, content)

	// Walk ADF to extract text
	firstPara := content[0].(map[string]any)
	paraContent := firstPara["content"].([]any)
	firstTextNode := paraContent[0].(map[string]any)
	descText := firstTextNode["text"].(string)
	testutil.Equal(t, descText, "Users cannot log in with SSO credentials")
	testutil.NotContains(t, descText, `"`)
}

func TestRunCreate_SummaryWithSpecialCharacters(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Issue{Key: "TEST-2", ID: "10002"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "PROJ", "Bug", `Error: "unexpected token" in parser`, "", "", "", nil)
	testutil.RequireNoError(t, err)

	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	summary := fields["summary"].(string)
	testutil.Equal(t, summary, `Error: "unexpected token" in parser`)
}

func TestNewCreateCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newCreateCmd(opts)

	testutil.Equal(t, cmd.Use, "create")
	testutil.Equal(t, cmd.Short, "Create a new issue")

	// Check required flags
	summaryFlag := cmd.Flags().Lookup("summary")
	testutil.NotNil(t, summaryFlag)
	testutil.Equal(t, summaryFlag.Shorthand, "s")

	projectFlag := cmd.Flags().Lookup("project")
	testutil.NotNil(t, projectFlag)
	testutil.Equal(t, projectFlag.Shorthand, "p")

	descFlag := cmd.Flags().Lookup("description")
	testutil.NotNil(t, descFlag)
	testutil.Equal(t, descFlag.Shorthand, "d")

	parentFlag := cmd.Flags().Lookup("parent")
	testutil.NotNil(t, parentFlag)
	testutil.Equal(t, parentFlag.Shorthand, "")

	assigneeFlag := cmd.Flags().Lookup("assignee")
	testutil.NotNil(t, assigneeFlag)
	testutil.Equal(t, assigneeFlag.Shorthand, "a")
}

func TestCreateCmd_CobraExecution_NoDoubleQuoting(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Issue{Key: "TEST-1", ID: "10001"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	cmd := newCreateCmd(opts)
	cmd.SetArgs([]string{
		"--project", "PROJ",
		"--type", "Task",
		"--summary", "Fix login bug",
		"--description", "Users cannot log in with SSO credentials",
	})

	err = cmd.Execute()
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)

	// Verify no double-quoting via Cobra flag parsing
	summary := fields["summary"].(string)
	testutil.Equal(t, summary, "Fix login bug")
	testutil.False(t, summary[0] == '"', "summary must not start with a literal quote")
}

func TestRunCreate_WithParent(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-456", ID: "10456"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "PROJ", "Task", "Child task", "", "PROJ-100", "", nil)
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)

	// Parent should be an object with "key" field
	parentField := fields["parent"].(map[string]any)
	testutil.Equal(t, parentField["key"], "PROJ-100")
}

func TestRunCreate_WithoutParent(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-789", ID: "10789"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "PROJ", "Task", "Standalone task", "", "", "", nil)
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	testutil.Nil(t, fields["parent"])
}

func TestCreateCmd_CobraExecution_WithParent(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-456", ID: "10456"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	cmd := newCreateCmd(opts)
	cmd.SetArgs([]string{
		"--project", "PROJ",
		"--type", "Task",
		"--summary", "Child task",
		"--parent", "PROJ-100",
	})

	err = cmd.Execute()
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	parentField := fields["parent"].(map[string]any)
	testutil.Equal(t, parentField["key"], "PROJ-100")
}

func TestRunCreate_WithAssigneeAccountID(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-500", ID: "10500"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "PROJ", "Task", "Assigned task", "", "", "61292e4c4f29230069621c5f", nil)
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	assigneeField := fields["assignee"].(map[string]any)
	testutil.Equal(t, assigneeField["accountId"], "61292e4c4f29230069621c5f")
}

func TestRunCreate_WithAssigneeMe(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/myself" && r.Method == "GET" {
			_ = json.NewEncoder(w).Encode(api.User{
				AccountID:   "myself-account-id",
				DisplayName: "Test User",
			})
			return
		}
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-501", ID: "10501"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "PROJ", "Task", "My task", "", "", "me", nil)
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	assigneeField := fields["assignee"].(map[string]any)
	testutil.Equal(t, assigneeField["accountId"], "myself-account-id")
}

func TestRunCreate_WithAssigneeEmail(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/user/search" && r.Method == "GET" {
			_ = json.NewEncoder(w).Encode([]api.User{
				{AccountID: "found-account-id", DisplayName: "Found User"},
			})
			return
		}
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-502", ID: "10502"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "PROJ", "Task", "Their task", "", "", "user@example.com", nil)
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	assigneeField := fields["assignee"].(map[string]any)
	testutil.Equal(t, assigneeField["accountId"], "found-account-id")
}

func TestRunCreate_DescriptionEscapeSequences(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Issue{Key: "TEST-10", ID: "10010"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	// Simulate what the shell passes when user types: --description "First paragraph.\n\nSecond paragraph."
	// The shell delivers literal backslash-n, not actual newlines.
	err = runCreate(context.Background(), opts, "PROJ", "Task", "Test", `First paragraph.\n\nSecond paragraph.`, "", "", nil)
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	desc := fields["description"].(map[string]any)
	testutil.Equal(t, desc["type"], "doc")

	// With escape interpretation, the description should produce multiple paragraphs
	// (not a single paragraph with literal \n text)
	content := desc["content"].([]any)
	testutil.GreaterOrEqual(t, len(content), 2)
}

func TestRunCreate_WithoutAssignee(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/issue" && r.Method == "POST" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.Issue{Key: "PROJ-503", ID: "10503"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runCreate(context.Background(), opts, "PROJ", "Task", "Unassigned task", "", "", "", nil)
	testutil.RequireNoError(t, err)

	testutil.NotEmpty(t, capturedBody)
	var reqBody map[string]any
	err = json.Unmarshal(capturedBody, &reqBody)
	testutil.RequireNoError(t, err)

	fields := reqBody["fields"].(map[string]any)
	testutil.Nil(t, fields["assignee"])
}
