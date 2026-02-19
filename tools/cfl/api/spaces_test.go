package api //nolint:revive // package name is intentional

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func loadTestData(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename)) //nolint:gosec // reading test fixture data
	testutil.RequireNoError(t, err)
	return data
}

func TestClient_ListSpaces(t *testing.T) {
	t.Parallel()
	testData := loadTestData(t, "spaces.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/spaces", r.URL.Path)
		testutil.Equal(t, "GET", r.Method)

		// Check query params
		testutil.Equal(t, "25", r.URL.Query().Get("limit"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.ListSpaces(context.Background(), nil)

	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Results, 2)
	testutil.True(t, result.HasMore())

	// Check first space
	space := result.Results[0]
	testutil.Equal(t, "123456", space.ID)
	testutil.Equal(t, "DEV", space.Key)
	testutil.Equal(t, "Development", space.Name)
	testutil.Equal(t, "global", space.Type)
}

func TestClient_ListSpaces_WithOptions(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "50", r.URL.Query().Get("limit"))
		testutil.Equal(t, "global", r.URL.Query().Get("type"))
		testutil.Equal(t, "current", r.URL.Query().Get("status"))

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
	testutil.RequireNoError(t, err)
}

func TestClient_GetSpace(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/spaces/123456", r.URL.Path)
		testutil.Equal(t, "GET", r.Method)

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

	testutil.RequireNoError(t, err)
	testutil.Equal(t, "123456", space.ID)
	testutil.Equal(t, "DEV", space.Key)
	testutil.Equal(t, "Development", space.Name)
}

func TestClient_GetSpace_NotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Space not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	_, err := client.GetSpace(context.Background(), "invalid")

	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "Space not found")
}

func TestClient_GetSpaceByKey_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/api/v2/spaces", r.URL.Path)
		testutil.Equal(t, "DEV", r.URL.Query().Get("keys"))
		testutil.Equal(t, "1", r.URL.Query().Get("limit"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [{
				"id": "123456",
				"key": "DEV",
				"name": "Development",
				"type": "global"
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	space, err := client.GetSpaceByKey(context.Background(), "DEV")

	testutil.RequireNoError(t, err)
	testutil.Equal(t, "123456", space.ID)
	testutil.Equal(t, "DEV", space.Key)
	testutil.Equal(t, "Development", space.Name)
}

func TestClient_GetSpaceByKey_NotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "NONEXISTENT", r.URL.Query().Get("keys"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	_, err := client.GetSpaceByKey(context.Background(), "NONEXISTENT")

	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "not found")
}

func TestClient_ListSpaces_WithMultipleKeys(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that multiple keys are passed correctly
		keys := r.URL.Query()["keys"]
		keysStr := strings.Join(keys, ",")
		testutil.Contains(t, keysStr, "DEV")
		testutil.Contains(t, keysStr, "PROD")
		testutil.Contains(t, keysStr, "TEST")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{"id": "1", "key": "DEV", "name": "Development"},
				{"id": "2", "key": "PROD", "name": "Production"},
				{"id": "3", "key": "TEST", "name": "Testing"}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	opts := &ListSpacesOptions{
		Keys: []string{"DEV", "PROD", "TEST"},
	}
	result, err := client.ListSpaces(context.Background(), opts)

	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Results, 3)
}

func TestClient_ListSpaces_EmptyResults(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.ListSpaces(context.Background(), nil)

	testutil.RequireNoError(t, err)
	testutil.Empty(t, result.Results)
	testutil.False(t, result.HasMore())
}

func TestClient_ListSpaces_NullDescription(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [{
				"id": "123456",
				"key": "TEST",
				"name": "Test Space",
				"description": null
			}]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "token")
	result, err := client.ListSpaces(context.Background(), nil)

	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Results, 1)
	testutil.Equal(t, "TEST", result.Results[0].Key)
}

func TestClient_ListSpaces_APIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message": "Authentication required"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com", "bad-token")
	_, err := client.ListSpaces(context.Background(), nil)

	testutil.RequireError(t, err)
}
