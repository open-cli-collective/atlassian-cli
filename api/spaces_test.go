package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestData(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	require.NoError(t, err)
	return data
}

func TestClient_ListSpaces(t *testing.T) {
	testData := loadTestData(t, "spaces.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/spaces", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		// Check query params
		assert.Equal(t, "25", r.URL.Query().Get("limit"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.ListSpaces(context.Background(), nil)

	require.NoError(t, err)
	assert.Len(t, result.Results, 2)
	assert.True(t, result.HasMore())

	// Check first space
	space := result.Results[0]
	assert.Equal(t, "123456", space.ID)
	assert.Equal(t, "DEV", space.Key)
	assert.Equal(t, "Development", space.Name)
	assert.Equal(t, "global", space.Type)
}

func TestClient_ListSpaces_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		assert.Equal(t, "global", r.URL.Query().Get("type"))
		assert.Equal(t, "current", r.URL.Query().Get("status"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	opts := &ListSpacesOptions{
		Limit:  50,
		Type:   "global",
		Status: "current",
	}
	_, err := client.ListSpaces(context.Background(), opts)
	require.NoError(t, err)
}

func TestClient_GetSpace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/spaces/123456", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "123456",
			"key": "DEV",
			"name": "Development",
			"type": "global"
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	space, err := client.GetSpace(context.Background(), "123456")

	require.NoError(t, err)
	assert.Equal(t, "123456", space.ID)
	assert.Equal(t, "DEV", space.Key)
	assert.Equal(t, "Development", space.Name)
}

func TestClient_GetSpace_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Space not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	_, err := client.GetSpace(context.Background(), "invalid")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Space not found")
}
