package attachment

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/confluence-cli/api"
)

// mockAttachmentServer creates a test server that handles attachment get and delete
func mockAttachmentServer(t *testing.T, getHandler, deleteHandler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/v2/attachments/") {
			if getHandler != nil {
				getHandler(w, r)
				return
			}
			// Default: return a valid attachment
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": "att123", "title": "test-file.txt", "mediaType": "text/plain", "fileSize": 100}`))
			return
		}
		if r.Method == "DELETE" && strings.HasPrefix(r.URL.Path, "/api/v2/attachments/") {
			if deleteHandler != nil {
				deleteHandler(w, r)
				return
			}
			// Default: successful delete
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestRunDeleteAttachment_ForceDelete(t *testing.T) {
	server := mockAttachmentServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/api/v2/attachments/att123", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	client := api.NewClient(server.URL, "user@example.com", "token")
	opts := &deleteOptions{
		force: true,
		stdin: strings.NewReader(""), // Not used with force
	}

	err := runDeleteAttachment("att123", opts, client)
	require.NoError(t, err)
}

func TestRunDeleteAttachment_ConfirmWithY(t *testing.T) {
	deleted := false
	server := mockAttachmentServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	client := api.NewClient(server.URL, "user@example.com", "token")
	opts := &deleteOptions{
		force: false,
		stdin: strings.NewReader("y\n"),
	}

	err := runDeleteAttachment("att123", opts, client)
	require.NoError(t, err)
	assert.True(t, deleted, "attachment should have been deleted")
}

func TestRunDeleteAttachment_ConfirmWithUpperY(t *testing.T) {
	deleted := false
	server := mockAttachmentServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	client := api.NewClient(server.URL, "user@example.com", "token")
	opts := &deleteOptions{
		force: false,
		stdin: strings.NewReader("Y\n"),
	}

	err := runDeleteAttachment("att123", opts, client)
	require.NoError(t, err)
	assert.True(t, deleted, "attachment should have been deleted")
}

func TestRunDeleteAttachment_CancelWithN(t *testing.T) {
	deleted := false
	server := mockAttachmentServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	client := api.NewClient(server.URL, "user@example.com", "token")
	opts := &deleteOptions{
		force: false,
		stdin: strings.NewReader("n\n"),
	}

	err := runDeleteAttachment("att123", opts, client)
	require.NoError(t, err)
	assert.False(t, deleted, "attachment should NOT have been deleted")
}

func TestRunDeleteAttachment_CancelWithEmpty(t *testing.T) {
	deleted := false
	server := mockAttachmentServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	client := api.NewClient(server.URL, "user@example.com", "token")
	opts := &deleteOptions{
		force: false,
		stdin: strings.NewReader("\n"),
	}

	err := runDeleteAttachment("att123", opts, client)
	require.NoError(t, err)
	assert.False(t, deleted, "attachment should NOT have been deleted")
}

func TestRunDeleteAttachment_CancelWithOther(t *testing.T) {
	deleted := false
	server := mockAttachmentServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	client := api.NewClient(server.URL, "user@example.com", "token")
	opts := &deleteOptions{
		force: false,
		stdin: strings.NewReader("maybe\n"),
	}

	err := runDeleteAttachment("att123", opts, client)
	require.NoError(t, err)
	assert.False(t, deleted, "attachment should NOT have been deleted")
}

func TestRunDeleteAttachment_GetAttachmentFails(t *testing.T) {
	server := mockAttachmentServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Attachment not found"}`))
		},
		nil,
	)
	defer server.Close()

	client := api.NewClient(server.URL, "user@example.com", "token")
	opts := &deleteOptions{
		force: true,
		stdin: strings.NewReader(""),
	}

	err := runDeleteAttachment("invalid", opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get attachment")
}

func TestRunDeleteAttachment_DeleteFails(t *testing.T) {
	server := mockAttachmentServer(t, nil,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message": "Permission denied"}`))
		},
	)
	defer server.Close()

	client := api.NewClient(server.URL, "user@example.com", "token")
	opts := &deleteOptions{
		force: true,
		stdin: strings.NewReader(""),
	}

	err := runDeleteAttachment("att123", opts, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete attachment")
}
