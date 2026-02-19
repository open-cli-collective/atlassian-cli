package page

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// mockPageServer creates a test server that handles GetPage and DeletePage requests
func mockPageServer(t *testing.T, pageID, title string, deleteStatus int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/"+pageID):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "` + pageID + `",
				"title": "` + title + `",
				"spaceId": "123456",
				"version": {"number": 1}
			}`))
		case r.Method == "DELETE" && strings.Contains(r.URL.Path, "/pages/"+pageID):
			w.WriteHeader(deleteStatus)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func newDeleteTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdin:   strings.NewReader(""),
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRunDelete_ConfirmYes(t *testing.T) {
	t.Parallel()
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	rootOpts := newDeleteTestRootOptions()
	rootOpts.Stdin = strings.NewReader("y\n")
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   false,
	}

	err := runDelete(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
}

func TestRunDelete_ConfirmYesUppercase(t *testing.T) {
	t.Parallel()
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	rootOpts := newDeleteTestRootOptions()
	rootOpts.Stdin = strings.NewReader("Y\n")
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   false,
	}

	err := runDelete(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
}

func TestRunDelete_ConfirmNo(t *testing.T) {
	t.Parallel()
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	rootOpts := newDeleteTestRootOptions()
	rootOpts.Stdin = strings.NewReader("n\n")
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   false,
	}

	err := runDelete(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err) // Cancellation is not an error
}

func TestRunDelete_ConfirmEmpty(t *testing.T) {
	t.Parallel()
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	rootOpts := newDeleteTestRootOptions()
	rootOpts.Stdin = strings.NewReader("\n")
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   false,
	}

	err := runDelete(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err) // Empty input should cancel
}

func TestRunDelete_ConfirmOther(t *testing.T) {
	t.Parallel()
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	rootOpts := newDeleteTestRootOptions()
	rootOpts.Stdin = strings.NewReader("maybe\n")
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   false,
	}

	err := runDelete(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err) // Any non-y/Y input should cancel
}

func TestRunDelete_Force(t *testing.T) {
	t.Parallel()
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	rootOpts := newDeleteTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   true,
	}

	err := runDelete(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
}

func TestRunDelete_PageNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	rootOpts := newDeleteTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   true,
	}

	err := runDelete(context.Background(), "99999", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "getting page")
}

func TestRunDelete_DeleteFailed(t *testing.T) {
	t.Parallel()
	server := mockPageServer(t, "12345", "Test Page", http.StatusForbidden)
	defer server.Close()

	rootOpts := newDeleteTestRootOptions()
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   true,
	}

	err := runDelete(context.Background(), "12345", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "deleting page")
}

func TestRunDelete_JSONOutput(t *testing.T) {
	t.Parallel()
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	rootOpts := newDeleteTestRootOptions()
	rootOpts.Output = "json"
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   true,
	}

	err := runDelete(context.Background(), "12345", opts)
	testutil.RequireNoError(t, err)
}

func TestRunDelete_ConfirmationInputs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		input         string
		shouldProceed bool
	}{
		{"lowercase y", "y\n", true},
		{"uppercase Y", "Y\n", true},
		{"lowercase n", "n\n", false},
		{"uppercase N", "N\n", false},
		{"empty input", "\n", false},
		{"other input", "yes\n", false},
		{"whitespace", "  \n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Track if delete was called
			deleteCalled := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "DELETE" {
					deleteCalled = true
					w.WriteHeader(http.StatusNoContent)
					return
				}
				// GET request for page info
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id": "12345", "title": "Test", "version": {"number": 1}}`))
			}))
			defer server.Close()

			rootOpts := newDeleteTestRootOptions()
			rootOpts.Stdin = strings.NewReader(tt.input)
			client := api.NewClient(server.URL, "test@example.com", "token")
			rootOpts.SetAPIClient(client)

			opts := &deleteOptions{
				Options: rootOpts,
				force:   false,
			}

			err := runDelete(context.Background(), "12345", opts)
			testutil.RequireNoError(t, err)
			testutil.Equal(t, deleteCalled, tt.shouldProceed)
		})
	}
}
