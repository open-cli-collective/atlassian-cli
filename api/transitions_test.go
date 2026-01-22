package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindTransitionByName(t *testing.T) {
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
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantID, result.ID)
			}
		})
	}
}

func TestFindTransitionByName_EmptySlice(t *testing.T) {
	result := FindTransitionByName([]Transition{}, "In Progress")
	assert.Nil(t, result)
}

func TestFindTransitionByName_NilSlice(t *testing.T) {
	result := FindTransitionByName(nil, "In Progress")
	assert.Nil(t, result)
}

func TestClient_GetTransitions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/issue/PROJ-123/transitions")
		assert.Empty(t, r.URL.Query().Get("expand"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"transitions": [
				{"id": "11", "name": "To Do", "to": {"id": "1", "name": "To Do"}},
				{"id": "21", "name": "In Progress", "to": {"id": "2", "name": "In Progress"}}
			]
		}`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Email:      "user@example.com",
		APIToken:   "token",
		HTTPClient: server.Client(),
	}

	transitions, err := client.GetTransitions("PROJ-123")
	require.NoError(t, err)
	assert.Len(t, transitions, 2)
	assert.Equal(t, "11", transitions[0].ID)
	assert.Equal(t, "To Do", transitions[0].Name)
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
				_, err := client.GetTransitionsWithFields(tt.issueKey, tt.includeFields)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "/issue/"+tt.issueKey+"/transitions")
				if tt.wantExpand {
					assert.Equal(t, "transitions.fields", r.URL.Query().Get("expand"))
				} else {
					assert.Empty(t, r.URL.Query().Get("expand"))
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
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

			client := &Client{
				BaseURL:    server.URL,
				Email:      "user@example.com",
				APIToken:   "token",
				HTTPClient: server.Client(),
			}

			transitions, err := client.GetTransitionsWithFields(tt.issueKey, tt.includeFields)
			require.NoError(t, err)
			assert.Len(t, transitions, 1)
			assert.Equal(t, "In Progress", transitions[0].Name)
			if tt.includeFields {
				assert.NotEmpty(t, transitions[0].Fields)
				field, ok := transitions[0].Fields["resolution"]
				assert.True(t, ok)
				assert.True(t, field.Required)
				assert.Equal(t, "Resolution", field.Name)
			}
		})
	}
}

func TestClient_DoTransition(t *testing.T) {
	tests := []struct {
		name         string
		issueKey     string
		transitionID string
		fields       map[string]interface{}
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
			fields: map[string]interface{}{
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
				err := client.DoTransition(tt.issueKey, tt.transitionID, tt.fields)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			var receivedBody TransitionRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.URL.Path, "/issue/"+tt.issueKey+"/transitions")
				err := json.NewDecoder(r.Body).Decode(&receivedBody)
				require.NoError(t, err)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			client := &Client{
				BaseURL:    server.URL,
				Email:      "user@example.com",
				APIToken:   "token",
				HTTPClient: server.Client(),
			}

			err := client.DoTransition(tt.issueKey, tt.transitionID, tt.fields)
			require.NoError(t, err)
			assert.Equal(t, tt.transitionID, receivedBody.Transition.ID)
			if tt.fields != nil {
				assert.NotEmpty(t, receivedBody.Fields)
			}
		})
	}
}
