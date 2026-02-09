package issues

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestRunUpdate_RequestBodyNoDoubleQuoting(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(opts, "PROJ-123", "Updated summary", "Updated description", nil)
	require.NoError(t, err)

	require.NotEmpty(t, capturedBody)

	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})

	// Summary must be the exact string without extra quotes
	summary := fields["summary"].(string)
	assert.Equal(t, "Updated summary", summary, "summary should not have extra quotes")
	assert.NotContains(t, summary, `"`, "summary should not contain literal quote characters")

	// Description should be ADF format
	desc := fields["description"].(map[string]interface{})
	assert.Equal(t, "doc", desc["type"], "description should be ADF document")
	content := desc["content"].([]interface{})
	require.NotEmpty(t, content)

	firstPara := content[0].(map[string]interface{})
	paraContent := firstPara["content"].([]interface{})
	firstTextNode := paraContent[0].(map[string]interface{})
	descText := firstTextNode["text"].(string)
	assert.Equal(t, "Updated description", descText,
		"description text should not have extra quotes")
}

func TestNewUpdateCmd(t *testing.T) {
	opts := &root.Options{}
	cmd := newUpdateCmd(opts)

	assert.Equal(t, "update <issue-key>", cmd.Use)
	assert.Equal(t, "Update an issue", cmd.Short)

	summaryFlag := cmd.Flags().Lookup("summary")
	require.NotNil(t, summaryFlag)
	assert.Equal(t, "s", summaryFlag.Shorthand)

	descFlag := cmd.Flags().Lookup("description")
	require.NotNil(t, descFlag)
	assert.Equal(t, "d", descFlag.Shorthand)
}

func TestRunUpdate_SummaryOnly(t *testing.T) {
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			capturedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
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
	require.NoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{
		Output: "table",
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
	}
	opts.SetAPIClient(client)

	err = runUpdate(opts, "PROJ-123", "New summary", "", nil)
	require.NoError(t, err)

	var reqBody map[string]interface{}
	err = json.Unmarshal(capturedBody, &reqBody)
	require.NoError(t, err)

	fields := reqBody["fields"].(map[string]interface{})
	assert.Equal(t, "New summary", fields["summary"])
	assert.Nil(t, fields["description"], "description should not be present when empty")
}

func TestRunUpdate_NoFieldsError(t *testing.T) {
	opts := &root.Options{
		Output: "table",
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	err := runUpdate(opts, "PROJ-123", "", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified")
}
