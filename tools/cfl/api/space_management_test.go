package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestClient_CreateSpace(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "POST", r.Method)
		testutil.Equal(t, "/api/v2/spaces", r.URL.Path)

		var req CreateSpaceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.Equal(t, "TEST", req.Key)
		testutil.Equal(t, "Test Space", req.Name)
		testutil.Equal(t, "global", req.Type)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "123456",
			"key": "TEST",
			"name": "Test Space",
			"type": "global",
			"status": "current",
			"_links": {"webui": "/spaces/TEST"}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	space, err := client.CreateSpace(context.Background(), &CreateSpaceRequest{
		Key:  "TEST",
		Name: "Test Space",
		Type: "global",
	})

	testutil.RequireNoError(t, err)
	testutil.Equal(t, "123456", space.ID)
	testutil.Equal(t, "TEST", space.Key)
	testutil.Equal(t, "Test Space", space.Name)
	testutil.Equal(t, "global", space.Type)
	testutil.Equal(t, "/spaces/TEST", space.Links.WebUI)
}

func TestClient_CreateSpace_WithDescription(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CreateSpaceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.NotNil(t, req.Description)
		testutil.Equal(t, "A test space", req.Description.Plain.Value)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "123456",
			"key": "TEST",
			"name": "Test Space",
			"type": "global",
			"description": {"plain": {"value": "A test space"}}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	space, err := client.CreateSpace(context.Background(), &CreateSpaceRequest{
		Key:  "TEST",
		Name: "Test Space",
		Type: "global",
		Description: &SpaceDescription{
			Plain: &DescriptionValue{Value: "A test space"},
		},
	})

	testutil.RequireNoError(t, err)
	testutil.Equal(t, "TEST", space.Key)
	testutil.NotNil(t, space.Description)
	testutil.Equal(t, "A test space", space.Description.Plain.Value)
}

func TestClient_CreateSpace_Error(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message": "Space key already exists"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	_, err := client.CreateSpace(context.Background(), &CreateSpaceRequest{
		Key:  "DUPE",
		Name: "Duplicate",
	})

	testutil.RequireError(t, err)
}

func TestClient_UpdateSpace(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "PUT", r.Method)
		testutil.Equal(t, "/rest/api/space/TEST", r.URL.Path)

		var req UpdateSpaceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.Equal(t, "TEST", req.Key)
		testutil.Equal(t, "Updated Name", req.Name)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 123456,
			"key": "TEST",
			"name": "Updated Name",
			"type": "global",
			"description": {"plain": {"value": "Description", "representation": "plain"}},
			"_links": {"webui": "/spaces/TEST", "self": "https://example.atlassian.net/wiki/rest/api/space/TEST"}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	space, err := client.UpdateSpace(context.Background(), "TEST", &UpdateSpaceRequest{
		Key:  "TEST",
		Name: "Updated Name",
	})

	testutil.RequireNoError(t, err)
	testutil.Equal(t, "123456", space.ID)
	testutil.Equal(t, "TEST", space.Key)
	testutil.Equal(t, "Updated Name", space.Name)
	testutil.Equal(t, "global", space.Type)
	testutil.NotNil(t, space.Description)
	testutil.Equal(t, "Description", space.Description.Plain.Value)
}

func TestClient_UpdateSpace_WithDescription(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req UpdateSpaceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.NotNil(t, req.Description)
		testutil.Equal(t, "New description", req.Description.Plain.Value)
		testutil.Equal(t, "plain", req.Description.Plain.Representation)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 123456,
			"key": "TEST",
			"name": "Test Space",
			"type": "global",
			"description": {"plain": {"value": "New description", "representation": "plain"}},
			"_links": {"webui": "/spaces/TEST"}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	space, err := client.UpdateSpace(context.Background(), "TEST", &UpdateSpaceRequest{
		Key:  "TEST",
		Name: "Test Space",
		Description: &V1SpaceDescription{
			Plain: &V1DescriptionValue{
				Value:          "New description",
				Representation: "plain",
			},
		},
	})

	testutil.RequireNoError(t, err)
	testutil.Equal(t, "New description", space.Description.Plain.Value)
}

func TestClient_UpdateSpace_NotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Space not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	_, err := client.UpdateSpace(context.Background(), "NOPE", &UpdateSpaceRequest{
		Key:  "NOPE",
		Name: "Updated",
	})

	testutil.RequireError(t, err)
}

func TestClient_UpdateSpace_NoDescription(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 123456,
			"key": "TEST",
			"name": "Test Space",
			"type": "global",
			"description": {"plain": {"value": "", "representation": "plain"}},
			"_links": {"webui": "/spaces/TEST"}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	space, err := client.UpdateSpace(context.Background(), "TEST", &UpdateSpaceRequest{
		Key:  "TEST",
		Name: "Test Space",
	})

	testutil.RequireNoError(t, err)
	testutil.Nil(t, space.Description)
}

func TestClient_DeleteSpace(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "DELETE", r.Method)
		testutil.Equal(t, "/rest/api/space/TEST", r.URL.Path)

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.DeleteSpace(context.Background(), "TEST")

	testutil.RequireNoError(t, err)
}

func TestClient_DeleteSpace_NotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Space not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.DeleteSpace(context.Background(), "NOPE")

	testutil.RequireError(t, err)
}

func TestV1SpaceResponse_ToSpace(t *testing.T) {
	t.Parallel()
	response := &v1SpaceResponse{
		ID:   123456,
		Key:  "TEST",
		Name: "Test Space",
		Type: "global",
	}
	response.Description.Plain.Value = "A test space"
	response.Description.Plain.Representation = "plain"
	response.Links.WebUI = "/spaces/TEST"

	space := response.toSpace()

	testutil.Equal(t, "123456", space.ID)
	testutil.Equal(t, "TEST", space.Key)
	testutil.Equal(t, "Test Space", space.Name)
	testutil.Equal(t, "global", space.Type)
	testutil.Equal(t, "/spaces/TEST", space.Links.WebUI)
	testutil.NotNil(t, space.Description)
	testutil.Equal(t, "A test space", space.Description.Plain.Value)
}

func TestV1SpaceResponse_ToSpace_EmptyDescription(t *testing.T) {
	t.Parallel()
	response := &v1SpaceResponse{
		ID:   123456,
		Key:  "TEST",
		Name: "Test Space",
		Type: "global",
	}

	space := response.toSpace()

	testutil.Nil(t, space.Description)
}
