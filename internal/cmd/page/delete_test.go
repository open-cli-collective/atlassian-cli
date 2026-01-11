package page

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rianjs/confluence-cli/api"
)

// mockPageServer creates a test server that handles GetPage and DeletePage requests
func mockPageServer(t *testing.T, pageID, title string, deleteStatus int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/pages/"+pageID):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
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

func TestRunDelete_ConfirmYes(t *testing.T) {
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &deleteOptions{
		force:   false,
		output:  "",
		noColor: true,
		stdin:   strings.NewReader("y\n"),
	}

	err := runDelete("12345", opts, client)
	require.NoError(t, err)
}

func TestRunDelete_ConfirmYesUppercase(t *testing.T) {
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &deleteOptions{
		force:   false,
		output:  "",
		noColor: true,
		stdin:   strings.NewReader("Y\n"),
	}

	err := runDelete("12345", opts, client)
	require.NoError(t, err)
}

func TestRunDelete_ConfirmNo(t *testing.T) {
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &deleteOptions{
		force:   false,
		output:  "",
		noColor: true,
		stdin:   strings.NewReader("n\n"),
	}

	err := runDelete("12345", opts, client)
	require.NoError(t, err) // Cancellation is not an error
}

func TestRunDelete_ConfirmEmpty(t *testing.T) {
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &deleteOptions{
		force:   false,
		output:  "",
		noColor: true,
		stdin:   strings.NewReader("\n"),
	}

	err := runDelete("12345", opts, client)
	require.NoError(t, err) // Empty input should cancel
}

func TestRunDelete_ConfirmOther(t *testing.T) {
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &deleteOptions{
		force:   false,
		output:  "",
		noColor: true,
		stdin:   strings.NewReader("maybe\n"),
	}

	err := runDelete("12345", opts, client)
	require.NoError(t, err) // Any non-y/Y input should cancel
}

func TestRunDelete_Force(t *testing.T) {
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &deleteOptions{
		force:   true,
		output:  "",
		noColor: true,
		stdin:   nil, // stdin not used when force=true
	}

	err := runDelete("12345", opts, client)
	require.NoError(t, err)
}

func TestRunDelete_PageNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Page not found"}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &deleteOptions{
		force:   true,
		output:  "",
		noColor: true,
	}

	err := runDelete("99999", opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get page")
}

func TestRunDelete_DeleteFailed(t *testing.T) {
	server := mockPageServer(t, "12345", "Test Page", http.StatusForbidden)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &deleteOptions{
		force:   true,
		output:  "",
		noColor: true,
	}

	err := runDelete("12345", opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete page")
}

func TestRunDelete_JSONOutput(t *testing.T) {
	server := mockPageServer(t, "12345", "Test Page", http.StatusNoContent)
	defer server.Close()

	client := api.NewClient(server.URL, "test@example.com", "token")
	opts := &deleteOptions{
		force:   true,
		output:  "json",
		noColor: true,
	}

	err := runDelete("12345", opts, client)
	require.NoError(t, err)
}

func TestRunDelete_ConfirmationInputs(t *testing.T) {
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
				w.Write([]byte(`{"id": "12345", "title": "Test", "version": {"number": 1}}`))
			}))
			defer server.Close()

			client := api.NewClient(server.URL, "test@example.com", "token")
			opts := &deleteOptions{
				force:   false,
				noColor: true,
				stdin:   strings.NewReader(tt.input),
			}

			err := runDelete("12345", opts, client)
			require.NoError(t, err)
			assert.Equal(t, tt.shouldProceed, deleteCalled, "delete should have been called: %v", tt.shouldProceed)
		})
	}
}
