package transitions

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
)

func TestFormatFieldValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		field *api.Field
		value string
		want  any
	}{
		{
			name:  "nil field - returns string as-is",
			field: nil,
			value: "some value",
			want:  "some value",
		},
		{
			name: "option field - wraps in value map",
			field: &api.Field{
				ID:   "customfield_10001",
				Name: "Change Type",
				Schema: api.FieldSchema{
					Type: "option",
				},
			},
			value: "Bug Fix",
			want:  map[string]string{"value": "Bug Fix"},
		},
		{
			name: "array of options - wraps in array of value maps",
			field: &api.Field{
				ID:   "customfield_10002",
				Name: "Categories",
				Schema: api.FieldSchema{
					Type:  "array",
					Items: "option",
				},
			},
			value: "Frontend",
			want:  []map[string]string{{"value": "Frontend"}},
		},
		{
			name: "array of strings - wraps in string array",
			field: &api.Field{
				ID:   "labels",
				Name: "Labels",
				Schema: api.FieldSchema{
					Type:  "array",
					Items: "string",
				},
			},
			value: "urgent",
			want:  []string{"urgent"},
		},
		{
			name: "user field - wraps in accountId map",
			field: &api.Field{
				ID:   "assignee",
				Name: "Assignee",
				Schema: api.FieldSchema{
					Type: "user",
				},
			},
			value: "abc123",
			want:  map[string]string{"accountId": "abc123"},
		},
		{
			name: "string field - returns as-is",
			field: &api.Field{
				ID:   "summary",
				Name: "Summary",
				Schema: api.FieldSchema{
					Type: "string",
				},
			},
			value: "Updated summary",
			want:  "Updated summary",
		},
		{
			name: "number field - converts to float64",
			field: &api.Field{
				ID:   "customfield_10003",
				Name: "Story Points",
				Schema: api.FieldSchema{
					Type: "number",
				},
			},
			value: "5",
			want:  float64(5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := api.FormatFieldValue(tt.field, tt.value)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func TestGetRequiredFields(t *testing.T) {
	tests := []struct {
		name       string
		transition api.Transition
		want       string
	}{
		{
			name: "no fields",
			transition: api.Transition{
				ID:     "21",
				Name:   "In Progress",
				Fields: nil,
			},
			want: "-",
		},
		{
			name: "empty fields map",
			transition: api.Transition{
				ID:     "21",
				Name:   "In Progress",
				Fields: map[string]api.TransitionField{},
			},
			want: "-",
		},
		{
			name: "no required fields",
			transition: api.Transition{
				ID:   "21",
				Name: "In Progress",
				Fields: map[string]api.TransitionField{
					"resolution": {
						Required: false,
						Name:     "Resolution",
					},
				},
			},
			want: "-",
		},
		{
			name: "one required field",
			transition: api.Transition{
				ID:   "31",
				Name: "Done",
				Fields: map[string]api.TransitionField{
					"resolution": {
						Required: true,
						Name:     "Resolution",
					},
				},
			},
			want: "Resolution",
		},
		{
			name: "multiple required fields",
			transition: api.Transition{
				ID:   "31",
				Name: "Done",
				Fields: map[string]api.TransitionField{
					"resolution": {
						Required: true,
						Name:     "Resolution",
					},
					"customfield_10001": {
						Required: true,
						Name:     "Root Cause",
					},
					"comment": {
						Required: false,
						Name:     "Comment",
					},
				},
			},
			want: "Resolution, Root Cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRequiredFields(tt.transition)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func transitionsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := api.TransitionsResponse{
			Transitions: []api.Transition{
				{ID: "11", Name: "Backlog", To: api.Status{Name: "Backlog", StatusCategory: api.StatusCategory{Name: "To Do"}}},
				{ID: "31", Name: "In Development", To: api.Status{Name: "In Development", StatusCategory: api.StatusCategory{Name: "In Progress"}}},
			},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestRunList_Default(t *testing.T) {
	t.Parallel()
	server := transitionsServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "e@x", APIToken: "t"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "TEST-1", false)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "ID")
	testutil.Contains(t, out, "Backlog")
	testutil.Contains(t, out, "In Development")
}

func TestRunList_Extended(t *testing.T) {
	t.Parallel()
	server := transitionsServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "e@x", APIToken: "t"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "TEST-1", true)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Contains(t, out, "STATUS_CATEGORY")
	testutil.Contains(t, out, "To Do")
	testutil.Contains(t, out, "In Progress")
}

func TestRunList_IDOnly(t *testing.T) {
	t.Parallel()
	server := transitionsServer(t)
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "e@x", APIToken: "t"})
	testutil.RequireNoError(t, err)

	var stdout bytes.Buffer
	opts := &root.Options{Stdout: &stdout, Stderr: &bytes.Buffer{}, IDOnly: true}
	opts.SetAPIClient(client)

	err = runList(context.Background(), opts, "TEST-1", false)
	testutil.RequireNoError(t, err)

	out := stdout.String()
	testutil.Equal(t, out, "11\n31\n")
}

func TestRunList_DeprecatedFieldsAlias(t *testing.T) {
	t.Parallel()
	cmd := newListCmd(&root.Options{})
	flag := cmd.Flags().Lookup("fields")
	if flag == nil {
		t.Fatal("expected --fields flag to exist as deprecated alias")
	}
	if !strings.Contains(flag.Deprecated, "extended") {
		t.Errorf("expected deprecation message to mention --extended, got %q", flag.Deprecated)
	}
}
