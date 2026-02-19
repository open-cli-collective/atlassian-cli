package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestFindTransitionByName(t *testing.T) {
	t.Parallel()
	transitions := []Transition{
		{ID: "11", Name: "To Do", To: Status{Name: "To Do"}},
		{ID: "21", Name: "In Progress", To: Status{Name: "In Progress"}},
		{ID: "31", Name: "Done", To: Status{Name: "Done"}},
	}

	tests := []struct {
		name       string
		searchName string
		wantID     string
		wantNil    bool
	}{
		{
			name:       "exact match",
			searchName: "In Progress",
			wantID:     "21",
		},
		{
			name:       "case insensitive",
			searchName: "in progress",
			wantID:     "21",
		},
		{
			name:       "uppercase",
			searchName: "DONE",
			wantID:     "31",
		},
		{
			name:       "not found",
			searchName: "Blocked",
			wantNil:    true,
		},
		{
			name:       "empty name",
			searchName: "",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindTransitionByName(transitions, tt.searchName)
			if tt.wantNil {
				testutil.Nil(t, result)
			} else {
				testutil.NotNil(t, result)
				testutil.Equal(t, result.ID, tt.wantID)
			}
		})
	}
}

func TestFindTransitionByName_EmptySlice(t *testing.T) {
	result := FindTransitionByName([]Transition{}, "In Progress")
	testutil.Nil(t, result)
}

func TestFindTransitionByName_NilSlice(t *testing.T) {
	result := FindTransitionByName(nil, "In Progress")
	testutil.Nil(t, result)
}

func TestClient_GetTransitions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Contains(t, r.URL.Path, "/issue/PROJ-123/transitions")
		testutil.Empty(t, r.URL.Query().Get("expand"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"transitions": [
				{"id": "11", "name": "To Do", "to": {"id": "1", "name": "To Do"}},
				{"id": "21", "name": "In Progress", "to": {"id": "2", "name": "In Progress"}}
			]
		}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      server.URL,
		Email:    "user@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	transitions, err := client.GetTransitions(context.Background(), "PROJ-123")
	testutil.RequireNoError(t, err)
	testutil.Len(t, transitions, 2)
	testutil.Equal(t, transitions[0].ID, "11")
	testutil.Equal(t, transitions[0].Name, "To Do")
}

func TestClient_GetTransitionsWithFields(t *testing.T) {
	tests := []struct {
		name          string
		issueKey      string
		includeFields bool
		wantExpand    bool
		wantErr       error
	}{
		{
			name:          "without fields",
			issueKey:      "PROJ-123",
			includeFields: false,
			wantExpand:    false,
		},
		{
			name:          "with fields",
			issueKey:      "PROJ-456",
			includeFields: true,
			wantExpand:    true,
		},
		{
			name:     "empty issue key",
			issueKey: "",
			wantErr:  ErrIssueKeyRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr != nil {
				client := &Client{}
				_, err := client.GetTransitionsWithFields(context.Background(), tt.issueKey, tt.includeFields)
				testutil.True(t, errors.Is(err, tt.wantErr))
				return
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				testutil.Contains(t, r.URL.Path, "/issue/"+tt.issueKey+"/transitions")
				if tt.wantExpand {
					testutil.Equal(t, r.URL.Query().Get("expand"), "transitions.fields")
				} else {
					testutil.Empty(t, r.URL.Query().Get("expand"))
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"transitions": [
						{
							"id": "21",
							"name": "In Progress",
							"to": {"id": "2", "name": "In Progress"},
							"fields": {
								"resolution": {
									"required": true,
									"name": "Resolution",
									"allowedValues": [
										{"id": "1", "name": "Done"},
										{"id": "2", "name": "Won't Do"}
									]
								}
							}
						}
					]
				}`))
			}))
			defer server.Close()

			client, err := New(ClientConfig{
				URL:      server.URL,
				Email:    "user@example.com",
				APIToken: "token",
			})
			testutil.RequireNoError(t, err)

			transitions, err := client.GetTransitionsWithFields(context.Background(), tt.issueKey, tt.includeFields)
			testutil.RequireNoError(t, err)
			testutil.Len(t, transitions, 1)
			testutil.Equal(t, transitions[0].Name, "In Progress")
			if tt.includeFields {
				testutil.NotEmpty(t, transitions[0].Fields)
				field, ok := transitions[0].Fields["resolution"]
				testutil.True(t, ok)
				testutil.True(t, field.Required)
				testutil.Equal(t, field.Name, "Resolution")
			}
		})
	}
}

func TestClient_DoTransition(t *testing.T) {
	tests := []struct {
		name         string
		issueKey     string
		transitionID string
		fields       map[string]any
		wantErr      error
	}{
		{
			name:         "simple transition",
			issueKey:     "PROJ-123",
			transitionID: "21",
			fields:       nil,
		},
		{
			name:         "transition with fields",
			issueKey:     "PROJ-123",
			transitionID: "31",
			fields: map[string]any{
				"resolution": map[string]string{"name": "Done"},
			},
		},
		{
			name:         "empty issue key",
			issueKey:     "",
			transitionID: "21",
			wantErr:      ErrIssueKeyRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr != nil {
				client := &Client{}
				err := client.DoTransition(context.Background(), tt.issueKey, tt.transitionID, tt.fields)
				testutil.True(t, errors.Is(err, tt.wantErr))
				return
			}

			var receivedBody TransitionRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				testutil.Equal(t, r.Method, http.MethodPost)
				testutil.Contains(t, r.URL.Path, "/issue/"+tt.issueKey+"/transitions")
				err := json.NewDecoder(r.Body).Decode(&receivedBody)
				testutil.RequireNoError(t, err)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			client, err := New(ClientConfig{
				URL:      server.URL,
				Email:    "user@example.com",
				APIToken: "token",
			})
			testutil.RequireNoError(t, err)

			err = client.DoTransition(context.Background(), tt.issueKey, tt.transitionID, tt.fields)
			testutil.RequireNoError(t, err)
			testutil.Equal(t, receivedBody.Transition.ID, tt.transitionID)
			if tt.fields != nil {
				testutil.NotEmpty(t, receivedBody.Fields)
			}
		})
	}
}
