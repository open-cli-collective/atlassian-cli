package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_CreateSpace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v2/spaces", r.URL.Path)

		var req CreateSpaceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "TEST", req.Key)
		assert.Equal(t, "Test Space", req.Name)
		assert.Equal(t, "global", req.Type)

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

	require.NoError(t, err)
	assert.Equal(t, "123456", space.ID)
	assert.Equal(t, "TEST", space.Key)
	assert.Equal(t, "Test Space", space.Name)
	assert.Equal(t, "global", space.Type)
	assert.Equal(t, "/spaces/TEST", space.Links.WebUI)
}

func TestClient_CreateSpace_WithDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CreateSpaceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.NotNil(t, req.Description)
		assert.Equal(t, "A test space", req.Description.Plain.Value)

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

	require.NoError(t, err)
	assert.Equal(t, "TEST", space.Key)
	assert.NotNil(t, space.Description)
	assert.Equal(t, "A test space", space.Description.Plain.Value)
}

func TestClient_CreateSpace_Error(t *testing.T) {
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

	require.Error(t, err)
}

func TestClient_UpdateSpace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/rest/api/space/TEST", r.URL.Path)

		var req UpdateSpaceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "TEST", req.Key)
		assert.Equal(t, "Updated Name", req.Name)

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

	require.NoError(t, err)
	assert.Equal(t, "123456", space.ID)
	assert.Equal(t, "TEST", space.Key)
	assert.Equal(t, "Updated Name", space.Name)
	assert.Equal(t, "global", space.Type)
	assert.NotNil(t, space.Description)
	assert.Equal(t, "Description", space.Description.Plain.Value)
}

func TestClient_UpdateSpace_WithDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req UpdateSpaceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.NotNil(t, req.Description)
		assert.Equal(t, "New description", req.Description.Plain.Value)
		assert.Equal(t, "plain", req.Description.Plain.Representation)

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

	require.NoError(t, err)
	assert.Equal(t, "New description", space.Description.Plain.Value)
}

func TestClient_UpdateSpace_NotFound(t *testing.T) {
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

	require.Error(t, err)
}

func TestClient_UpdateSpace_NoDescription(t *testing.T) {
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

	require.NoError(t, err)
	assert.Nil(t, space.Description)
}

func TestClient_DeleteSpace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/rest/api/space/TEST", r.URL.Path)

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.DeleteSpace(context.Background(), "TEST")

	require.NoError(t, err)
}

func TestClient_DeleteSpace_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Space not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	err := client.DeleteSpace(context.Background(), "NOPE")

	require.Error(t, err)
}

func TestV1SpaceResponse_ToSpace(t *testing.T) {
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

	assert.Equal(t, "123456", space.ID)
	assert.Equal(t, "TEST", space.Key)
	assert.Equal(t, "Test Space", space.Name)
	assert.Equal(t, "global", space.Type)
	assert.Equal(t, "/spaces/TEST", space.Links.WebUI)
	assert.NotNil(t, space.Description)
	assert.Equal(t, "A test space", space.Description.Plain.Value)
}

func TestV1SpaceResponse_ToSpace_EmptyDescription(t *testing.T) {
	response := &v1SpaceResponse{
		ID:   123456,
		Key:  "TEST",
		Name: "Test Space",
		Type: "global",
	}

	space := response.toSpace()

	assert.Nil(t, space.Description)
}
