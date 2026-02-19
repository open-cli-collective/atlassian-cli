package space

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
	"github.com/open-cli-collective/confluence-cli/internal/config"
)

// spaceListResponse returns a v2 list response with a single space.
const spaceListResponse = `{
	"results": [{
		"id": "123456",
		"key": "TEST",
		"name": "Test Space",
		"type": "global",
		"status": "current",
		"description": {"plain": {"value": "A test space"}}
	}]
}`

// v1SpaceUpdateResponse returns a v1 API space response.
const v1SpaceUpdateResponse = `{
	"id": 123456,
	"key": "TEST",
	"name": "Updated Name",
	"type": "global",
	"description": {"plain": {"value": "Updated description", "representation": "plain"}},
	"_links": {"webui": "/spaces/TEST"}
}`

// --- View tests ---

func TestRunView_Table(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "GET", r.Method)
		testutil.Contains(t, r.URL.Path, "/spaces")
		testutil.Equal(t, "TEST", r.URL.Query().Get("keys"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(spaceListResponse))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	rootOpts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  stdout,
		Stderr:  &bytes.Buffer{},
	}
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{Options: rootOpts}
	err := runView(context.Background(), "TEST", opts)

	testutil.RequireNoError(t, err)
	output := stdout.String()
	testutil.Contains(t, output, "TEST")
	testutil.Contains(t, output, "Test Space")
	testutil.Contains(t, output, "global")
	testutil.Contains(t, output, "A test space")
}

func TestRunView_JSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(spaceListResponse))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	rootOpts := &root.Options{
		Output:  "json",
		NoColor: true,
		Stdout:  stdout,
		Stderr:  &bytes.Buffer{},
	}
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{Options: rootOpts}
	err := runView(context.Background(), "TEST", opts)

	testutil.RequireNoError(t, err)
	var result map[string]any
	err = json.Unmarshal(stdout.Bytes(), &result)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "TEST", result["key"])
}

func TestRunView_NotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &viewOptions{Options: rootOpts}
	err := runView(context.Background(), "NONEXISTENT", opts)

	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "not found")
}

// --- Create tests ---

func TestRunCreate(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "POST", r.Method)
		testutil.Equal(t, "/api/v2/spaces", r.URL.Path)

		var req api.CreateSpaceRequest
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
			"_links": {"webui": "/spaces/TEST"}
		}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	rootOpts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  stdout,
		Stderr:  &bytes.Buffer{},
	}
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.SetConfig(&config.Config{URL: "https://example.atlassian.net/wiki"})

	opts := &createOptions{
		Options:   rootOpts,
		key:       "TEST",
		name:      "Test Space",
		spaceType: "global",
	}

	err := runCreate(context.Background(), opts)

	testutil.RequireNoError(t, err)
	output := stdout.String()
	testutil.Contains(t, output, "Created space")
	testutil.Contains(t, output, "Test Space")
	testutil.Contains(t, output, "TEST")
}

func TestRunCreate_JSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "123456",
			"key": "TEST",
			"name": "Test Space",
			"type": "global"
		}`))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	rootOpts := &root.Options{
		Output:  "json",
		NoColor: true,
		Stdout:  stdout,
		Stderr:  &bytes.Buffer{},
	}
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.SetConfig(&config.Config{URL: "https://example.atlassian.net/wiki"})

	opts := &createOptions{
		Options:   rootOpts,
		key:       "TEST",
		name:      "Test Space",
		spaceType: "global",
	}

	err := runCreate(context.Background(), opts)

	testutil.RequireNoError(t, err)
	var result map[string]any
	err = json.Unmarshal(stdout.Bytes(), &result)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "TEST", result["key"])
}

func TestRunCreate_WithDescription(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req api.CreateSpaceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.NotNil(t, req.Description)
		testutil.Equal(t, "A test space", req.Description.Plain.Value)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "123456",
			"key": "TEST",
			"name": "Test Space",
			"type": "global"
		}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)
	rootOpts.SetConfig(&config.Config{URL: "https://example.atlassian.net/wiki"})

	opts := &createOptions{
		Options:     rootOpts,
		key:         "TEST",
		name:        "Test Space",
		description: "A test space",
		spaceType:   "global",
	}

	err := runCreate(context.Background(), opts)
	testutil.RequireNoError(t, err)
}

// --- Update tests ---

func TestRunUpdate(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "PUT", r.Method)
		testutil.Equal(t, "/rest/api/space/TEST", r.URL.Path)

		var req api.UpdateSpaceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.Equal(t, "TEST", req.Key)
		testutil.Equal(t, "Updated Name", req.Name)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(v1SpaceUpdateResponse))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	rootOpts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  stdout,
		Stderr:  &bytes.Buffer{},
	}
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &updateOptions{
		Options: rootOpts,
		name:    "Updated Name",
	}

	err := runUpdate(context.Background(), "TEST", opts)

	testutil.RequireNoError(t, err)
	output := stdout.String()
	testutil.Contains(t, output, "Updated space")
	testutil.Contains(t, output, "Updated Name")
}

func TestRunUpdate_JSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(v1SpaceUpdateResponse))
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	rootOpts := &root.Options{
		Output:  "json",
		NoColor: true,
		Stdout:  stdout,
		Stderr:  &bytes.Buffer{},
	}
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &updateOptions{
		Options: rootOpts,
		name:    "Updated Name",
	}

	err := runUpdate(context.Background(), "TEST", opts)

	testutil.RequireNoError(t, err)
	var result map[string]any
	err = json.Unmarshal(stdout.Bytes(), &result)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "TEST", result["key"])
}

func TestRunUpdate_NoFlags(t *testing.T) {
	t.Parallel()
	rootOpts := newTestRootOptions()

	opts := &updateOptions{
		Options: rootOpts,
	}

	err := runUpdate(context.Background(), "TEST", opts)

	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "at least one of --name or --description is required")
}

func TestRunUpdate_WithDescription(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req api.UpdateSpaceRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		testutil.RequireNoError(t, err)
		testutil.NotNil(t, req.Description)
		testutil.Equal(t, "New description", req.Description.Plain.Value)
		testutil.Equal(t, "plain", req.Description.Plain.Representation)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(v1SpaceUpdateResponse))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &updateOptions{
		Options:     rootOpts,
		description: "New description",
	}

	err := runUpdate(context.Background(), "TEST", opts)
	testutil.RequireNoError(t, err)
}

// --- Delete tests ---

func TestRunDelete_Force(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// GetSpaceByKey call
			testutil.Equal(t, "GET", r.Method)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(spaceListResponse))
			return
		}
		// DeleteSpace call
		testutil.Equal(t, "DELETE", r.Method)
		testutil.Equal(t, "/rest/api/space/TEST", r.URL.Path)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	rootOpts := &root.Options{
		Output:  "table",
		NoColor: true,
		Stdout:  stdout,
		Stderr:  &bytes.Buffer{},
	}
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   true,
	}

	err := runDelete(context.Background(), "TEST", opts)

	testutil.RequireNoError(t, err)
	output := stdout.String()
	testutil.Contains(t, output, "Deleted space")
	testutil.Contains(t, output, "Test Space")
}

func TestRunDelete_Force_JSON(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(spaceListResponse))
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	stdout := &bytes.Buffer{}
	rootOpts := &root.Options{
		Output:  "json",
		NoColor: true,
		Stdout:  stdout,
		Stderr:  &bytes.Buffer{},
	}
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   true,
	}

	err := runDelete(context.Background(), "TEST", opts)

	testutil.RequireNoError(t, err)
	var result map[string]string
	err = json.Unmarshal(stdout.Bytes(), &result)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "deleted", result["status"])
	testutil.Equal(t, "TEST", result["space_key"])
	testutil.Equal(t, "Test Space", result["name"])
}

func TestRunDelete_NoForce_Declined(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(spaceListResponse))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	rootOpts.Stdin = strings.NewReader("n\n")
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   false,
	}

	err := runDelete(context.Background(), "TEST", opts)

	testutil.RequireNoError(t, err)
}

func TestRunDelete_NoForce_Accepted(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(spaceListResponse))
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	rootOpts.Stdin = strings.NewReader("y\n")
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   false,
	}

	err := runDelete(context.Background(), "TEST", opts)

	testutil.RequireNoError(t, err)
	testutil.Equal(t, 2, callCount)
}

func TestRunDelete_NotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   true,
	}

	err := runDelete(context.Background(), "NONEXISTENT", opts)

	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "not found")
}
