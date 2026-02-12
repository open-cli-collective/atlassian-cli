package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	require.NoError(t, err)
	if server != nil {
		client.BaseURL = server.URL + "/rest/api/3"
	}
	return client
}

func TestCreateField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/api/3/field", r.URL.Path)

		var req CreateFieldRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "Environment", req.Name)
		assert.Equal(t, "com.atlassian.jira.plugin.system.customfieldtypes:select", req.Type)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Field{
			ID:     "customfield_10100",
			Name:   "Environment",
			Custom: true,
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	field, err := client.CreateField(&CreateFieldRequest{
		Name: "Environment",
		Type: "com.atlassian.jira.plugin.system.customfieldtypes:select",
	})
	require.NoError(t, err)
	assert.Equal(t, "customfield_10100", field.ID)
	assert.Equal(t, "Environment", field.Name)
	assert.True(t, field.Custom)
}

func TestCreateField_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errorMessages":["Field name already exists"]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.CreateField(&CreateFieldRequest{Name: "Dupe", Type: "select"})
	assert.Error(t, err)
}

func TestTrashField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/api/3/field/customfield_10100/trash", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.TrashField("customfield_10100")
	assert.NoError(t, err)
}

func TestTrashField_EmptyID(t *testing.T) {
	client := newTestClient(t, nil)
	err := client.TrashField("")
	assert.ErrorIs(t, err, ErrFieldIDRequired)
}

func TestRestoreField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/api/3/field/customfield_10100/restore", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.RestoreField("customfield_10100")
	assert.NoError(t, err)
}

func TestRestoreField_EmptyID(t *testing.T) {
	client := newTestClient(t, nil)
	err := client.RestoreField("")
	assert.ErrorIs(t, err, ErrFieldIDRequired)
}

func TestGetFieldContexts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/rest/api/3/field/customfield_10100/context", r.URL.Path)
		json.NewEncoder(w).Encode(FieldContextsResponse{
			MaxResults: 50,
			Total:      2,
			IsLast:     true,
			Values: []FieldContext{
				{ID: "10001", Name: "Default", IsGlobalContext: true, IsAnyIssueType: true},
				{ID: "10002", Name: "Bug Context", IsGlobalContext: false, IsAnyIssueType: false},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	result, err := client.GetFieldContexts("customfield_10100")
	require.NoError(t, err)
	assert.Len(t, result.Values, 2)
	assert.Equal(t, "Default", result.Values[0].Name)
	assert.True(t, result.Values[0].IsGlobalContext)
}

func TestGetFieldContexts_EmptyID(t *testing.T) {
	client := newTestClient(t, nil)
	_, err := client.GetFieldContexts("")
	assert.ErrorIs(t, err, ErrFieldIDRequired)
}

func TestGetDefaultFieldContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(FieldContextsResponse{
			Values: []FieldContext{
				{ID: "10001", Name: "Default"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	ctx, err := client.GetDefaultFieldContext("customfield_10100")
	require.NoError(t, err)
	assert.Equal(t, "10001", ctx.ID)
	assert.Equal(t, "Default", ctx.Name)
}

func TestGetDefaultFieldContext_NoContexts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(FieldContextsResponse{Values: []FieldContext{}})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.GetDefaultFieldContext("customfield_10100")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no contexts found")
}

func TestCreateFieldContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/api/3/field/customfield_10100/context", r.URL.Path)

		var req CreateFieldContextRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "Bug Context", req.Name)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(FieldContext{
			ID:   "10003",
			Name: "Bug Context",
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	ctx, err := client.CreateFieldContext("customfield_10100", &CreateFieldContextRequest{
		Name: "Bug Context",
	})
	require.NoError(t, err)
	assert.Equal(t, "10003", ctx.ID)
	assert.Equal(t, "Bug Context", ctx.Name)
}

func TestCreateFieldContext_EmptyID(t *testing.T) {
	client := newTestClient(t, nil)
	_, err := client.CreateFieldContext("", &CreateFieldContextRequest{Name: "test"})
	assert.ErrorIs(t, err, ErrFieldIDRequired)
}

func TestDeleteFieldContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/rest/api/3/field/customfield_10100/context/10003", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.DeleteFieldContext("customfield_10100", "10003")
	assert.NoError(t, err)
}

func TestDeleteFieldContext_EmptyID(t *testing.T) {
	client := newTestClient(t, nil)
	err := client.DeleteFieldContext("", "10003")
	assert.ErrorIs(t, err, ErrFieldIDRequired)
}

func TestGetFieldContextOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/rest/api/3/field/customfield_10100/context/10001/option", r.URL.Path)
		json.NewEncoder(w).Encode(FieldContextOptionsResponse{
			MaxResults: 50,
			Total:      2,
			IsLast:     true,
			Values: []FieldContextOption{
				{ID: "1", Value: "Production", Disabled: false},
				{ID: "2", Value: "Staging", Disabled: false},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	result, err := client.GetFieldContextOptions("customfield_10100", "10001")
	require.NoError(t, err)
	assert.Len(t, result.Values, 2)
	assert.Equal(t, "Production", result.Values[0].Value)
}

func TestGetFieldContextOptions_EmptyID(t *testing.T) {
	client := newTestClient(t, nil)
	_, err := client.GetFieldContextOptions("", "10001")
	assert.ErrorIs(t, err, ErrFieldIDRequired)
}

func TestCreateFieldContextOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rest/api/3/field/customfield_10100/context/10001/option", r.URL.Path)

		var req CreateFieldContextOptionsRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Len(t, req.Options, 1)
		assert.Equal(t, "Option A", req.Options[0].Value)

		json.NewEncoder(w).Encode(FieldContextOptionsResponse{
			Values: []FieldContextOption{
				{ID: "3", Value: "Option A"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	options, err := client.CreateFieldContextOptions("customfield_10100", "10001", &CreateFieldContextOptionsRequest{
		Options: []CreateFieldContextOptionEntry{
			{Value: "Option A"},
		},
	})
	require.NoError(t, err)
	assert.Len(t, options, 1)
	assert.Equal(t, "Option A", options[0].Value)
}

func TestCreateFieldContextOptions_EmptyID(t *testing.T) {
	client := newTestClient(t, nil)
	_, err := client.CreateFieldContextOptions("", "10001", &CreateFieldContextOptionsRequest{})
	assert.ErrorIs(t, err, ErrFieldIDRequired)
}

func TestUpdateFieldContextOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/rest/api/3/field/customfield_10100/context/10001/option", r.URL.Path)

		var req UpdateFieldContextOptionsRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Len(t, req.Options, 1)
		assert.Equal(t, "3", req.Options[0].ID)
		assert.Equal(t, "Option A (updated)", req.Options[0].Value)

		json.NewEncoder(w).Encode(FieldContextOptionsResponse{
			Values: []FieldContextOption{
				{ID: "3", Value: "Option A (updated)"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	options, err := client.UpdateFieldContextOptions("customfield_10100", "10001", &UpdateFieldContextOptionsRequest{
		Options: []UpdateFieldContextOptionEntry{
			{ID: "3", Value: "Option A (updated)"},
		},
	})
	require.NoError(t, err)
	assert.Len(t, options, 1)
	assert.Equal(t, "Option A (updated)", options[0].Value)
}

func TestUpdateFieldContextOptions_EmptyID(t *testing.T) {
	client := newTestClient(t, nil)
	_, err := client.UpdateFieldContextOptions("", "10001", &UpdateFieldContextOptionsRequest{})
	assert.ErrorIs(t, err, ErrFieldIDRequired)
}

func TestDeleteFieldContextOption(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/rest/api/3/field/customfield_10100/context/10001/option/3", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.DeleteFieldContextOption("customfield_10100", "10001", "3")
	assert.NoError(t, err)
}

func TestDeleteFieldContextOption_EmptyID(t *testing.T) {
	client := newTestClient(t, nil)
	err := client.DeleteFieldContextOption("", "10001", "3")
	assert.ErrorIs(t, err, ErrFieldIDRequired)
}
