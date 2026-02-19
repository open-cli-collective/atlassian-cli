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

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	testutil.RequireNoError(t, err)
	if server != nil {
		client.BaseURL = server.URL + "/rest/api/3"
	}
	return client
}

func TestCreateField(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/field")

		var req CreateFieldRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.Equal(t, req.Name, "Environment")
		testutil.Equal(t, req.Type, "com.atlassian.jira.plugin.system.customfieldtypes:select")

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Field{
			ID:     "customfield_10100",
			Name:   "Environment",
			Custom: true,
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	field, err := client.CreateField(context.Background(), &CreateFieldRequest{
		Name: "Environment",
		Type: "com.atlassian.jira.plugin.system.customfieldtypes:select",
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, field.ID, "customfield_10100")
	testutil.Equal(t, field.Name, "Environment")
	testutil.True(t, field.Custom)
}

func TestCreateField_ServerError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errorMessages":["Field name already exists"]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.CreateField(context.Background(), &CreateFieldRequest{Name: "Dupe", Type: "select"})
	testutil.Error(t, err)
}

func TestTrashField(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/field/customfield_10100/trash")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.TrashField(context.Background(), "customfield_10100")
	testutil.NoError(t, err)
}

func TestTrashField_EmptyID(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, nil)
	err := client.TrashField(context.Background(), "")
	testutil.True(t, errors.Is(err, ErrFieldIDRequired))
}

func TestRestoreField(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/field/customfield_10100/restore")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.RestoreField(context.Background(), "customfield_10100")
	testutil.NoError(t, err)
}

func TestRestoreField_EmptyID(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, nil)
	err := client.RestoreField(context.Background(), "")
	testutil.True(t, errors.Is(err, ErrFieldIDRequired))
}

func TestGetFieldContexts(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodGet)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/field/customfield_10100/context")
		_ = json.NewEncoder(w).Encode(FieldContextsResponse{
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
	result, err := client.GetFieldContexts(context.Background(), "customfield_10100")
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Values, 2)
	testutil.Equal(t, result.Values[0].Name, "Default")
	testutil.True(t, result.Values[0].IsGlobalContext)
}

func TestGetFieldContexts_EmptyID(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, nil)
	_, err := client.GetFieldContexts(context.Background(), "")
	testutil.True(t, errors.Is(err, ErrFieldIDRequired))
}

func TestGetDefaultFieldContext(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(FieldContextsResponse{
			Values: []FieldContext{
				{ID: "10001", Name: "Default"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	ctx, err := client.GetDefaultFieldContext(context.Background(), "customfield_10100")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, ctx.ID, "10001")
	testutil.Equal(t, ctx.Name, "Default")
}

func TestGetDefaultFieldContext_NoContexts(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(FieldContextsResponse{Values: []FieldContext{}})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.GetDefaultFieldContext(context.Background(), "customfield_10100")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "no contexts found")
}

func TestCreateFieldContext(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/field/customfield_10100/context")

		var req CreateFieldContextRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.Equal(t, req.Name, "Bug Context")

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(FieldContext{
			ID:   "10003",
			Name: "Bug Context",
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	ctx, err := client.CreateFieldContext(context.Background(), "customfield_10100", &CreateFieldContextRequest{
		Name: "Bug Context",
	})
	testutil.RequireNoError(t, err)
	testutil.Equal(t, ctx.ID, "10003")
	testutil.Equal(t, ctx.Name, "Bug Context")
}

func TestCreateFieldContext_EmptyID(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, nil)
	_, err := client.CreateFieldContext(context.Background(), "", &CreateFieldContextRequest{Name: "test"})
	testutil.True(t, errors.Is(err, ErrFieldIDRequired))
}

func TestDeleteFieldContext(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodDelete)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/field/customfield_10100/context/10003")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.DeleteFieldContext(context.Background(), "customfield_10100", "10003")
	testutil.NoError(t, err)
}

func TestDeleteFieldContext_EmptyID(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, nil)
	err := client.DeleteFieldContext(context.Background(), "", "10003")
	testutil.True(t, errors.Is(err, ErrFieldIDRequired))
}

func TestGetFieldContextOptions(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodGet)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/field/customfield_10100/context/10001/option")
		_ = json.NewEncoder(w).Encode(FieldContextOptionsResponse{
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
	result, err := client.GetFieldContextOptions(context.Background(), "customfield_10100", "10001")
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Values, 2)
	testutil.Equal(t, result.Values[0].Value, "Production")
}

func TestGetFieldContextOptions_EmptyID(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, nil)
	_, err := client.GetFieldContextOptions(context.Background(), "", "10001")
	testutil.True(t, errors.Is(err, ErrFieldIDRequired))
}

func TestCreateFieldContextOptions(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPost)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/field/customfield_10100/context/10001/option")

		var req CreateFieldContextOptionsRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.Len(t, req.Options, 1)
		testutil.Equal(t, req.Options[0].Value, "Option A")

		_ = json.NewEncoder(w).Encode(FieldContextOptionsResponse{
			Values: []FieldContextOption{
				{ID: "3", Value: "Option A"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	options, err := client.CreateFieldContextOptions(context.Background(), "customfield_10100", "10001", &CreateFieldContextOptionsRequest{
		Options: []CreateFieldContextOptionEntry{
			{Value: "Option A"},
		},
	})
	testutil.RequireNoError(t, err)
	testutil.Len(t, options, 1)
	testutil.Equal(t, options[0].Value, "Option A")
}

func TestCreateFieldContextOptions_EmptyID(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, nil)
	_, err := client.CreateFieldContextOptions(context.Background(), "", "10001", &CreateFieldContextOptionsRequest{})
	testutil.True(t, errors.Is(err, ErrFieldIDRequired))
}

func TestUpdateFieldContextOptions(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodPut)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/field/customfield_10100/context/10001/option")

		var req UpdateFieldContextOptionsRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.Len(t, req.Options, 1)
		testutil.Equal(t, req.Options[0].ID, "3")
		testutil.Equal(t, req.Options[0].Value, "Option A (updated)")

		_ = json.NewEncoder(w).Encode(FieldContextOptionsResponse{
			Values: []FieldContextOption{
				{ID: "3", Value: "Option A (updated)"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	options, err := client.UpdateFieldContextOptions(context.Background(), "customfield_10100", "10001", &UpdateFieldContextOptionsRequest{
		Options: []UpdateFieldContextOptionEntry{
			{ID: "3", Value: "Option A (updated)"},
		},
	})
	testutil.RequireNoError(t, err)
	testutil.Len(t, options, 1)
	testutil.Equal(t, options[0].Value, "Option A (updated)")
}

func TestUpdateFieldContextOptions_EmptyID(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, nil)
	_, err := client.UpdateFieldContextOptions(context.Background(), "", "10001", &UpdateFieldContextOptionsRequest{})
	testutil.True(t, errors.Is(err, ErrFieldIDRequired))
}

func TestDeleteFieldContextOption(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.Method, http.MethodDelete)
		testutil.Equal(t, r.URL.Path, "/rest/api/3/field/customfield_10100/context/10001/option/3")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.DeleteFieldContextOption(context.Background(), "customfield_10100", "10001", "3")
	testutil.NoError(t, err)
}

func TestDeleteFieldContextOption_EmptyID(t *testing.T) {
	t.Parallel()
	client := newTestClient(t, nil)
	err := client.DeleteFieldContextOption(context.Background(), "", "10001", "3")
	testutil.True(t, errors.Is(err, ErrFieldIDRequired))
}
