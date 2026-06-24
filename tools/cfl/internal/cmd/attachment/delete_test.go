package attachment

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/prompt"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
	"github.com/open-cli-collective/confluence-cli/internal/cmd/root"
)

// mockAttachmentServer creates a test server that handles attachment get and delete
func mockAttachmentServer(_ *testing.T, getHandler, deleteHandler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
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

func newTestRootOptions() *root.Options {
	return &root.Options{
		Output:  "table",
		NoColor: true,
		Stdin:   strings.NewReader(""),
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}
}

func TestRunDeleteAttachment_ForceDelete(t *testing.T) {
	t.Parallel()
	server := mockAttachmentServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "DELETE", r.Method)
		testutil.Equal(t, "/api/v2/attachments/att123", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   true,
	}

	err := runDeleteAttachment(context.Background(), "att123", opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "Deleted attachment: test-file.txt (ID: att123)\n", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "", rootOpts.Stderr.(*bytes.Buffer).String())
}

func TestRunDeleteAttachment_ConfirmWithY(t *testing.T) {
	t.Parallel()
	deleted := false
	server := mockAttachmentServer(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	rootOpts := newTestRootOptions()
	rootOpts.Stdin = strings.NewReader("y\n")
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   false,
	}

	err := runDeleteAttachment(context.Background(), "att123", opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "Deleted attachment: test-file.txt (ID: att123)\n", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "About to delete attachment: test-file.txt (ID: att123)\nAre you sure? [y/N]: ", rootOpts.Stderr.(*bytes.Buffer).String())
	testutil.True(t, deleted, "attachment should have been deleted")
}

func TestRunDeleteAttachment_ConfirmWithUpperY(t *testing.T) {
	t.Parallel()
	deleted := false
	server := mockAttachmentServer(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	rootOpts := newTestRootOptions()
	rootOpts.Stdin = strings.NewReader("Y\n")
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   false,
	}

	err := runDeleteAttachment(context.Background(), "att123", opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "Deleted attachment: test-file.txt (ID: att123)\n", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "About to delete attachment: test-file.txt (ID: att123)\nAre you sure? [y/N]: ", rootOpts.Stderr.(*bytes.Buffer).String())
	testutil.True(t, deleted, "attachment should have been deleted")
}

func TestRunDeleteAttachment_CancelWithN(t *testing.T) {
	t.Parallel()
	deleted := false
	server := mockAttachmentServer(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	rootOpts := newTestRootOptions()
	rootOpts.Stdin = strings.NewReader("n\n")
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   false,
	}

	err := runDeleteAttachment(context.Background(), "att123", opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "About to delete attachment: test-file.txt (ID: att123)\nAre you sure? [y/N]: Deletion cancelled.\n", rootOpts.Stderr.(*bytes.Buffer).String())
	testutil.False(t, deleted, "attachment should NOT have been deleted")
}

func TestRunDeleteAttachment_CancelWithEmpty(t *testing.T) {
	t.Parallel()
	deleted := false
	server := mockAttachmentServer(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	rootOpts := newTestRootOptions()
	rootOpts.Stdin = strings.NewReader("\n")
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   false,
	}

	err := runDeleteAttachment(context.Background(), "att123", opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "About to delete attachment: test-file.txt (ID: att123)\nAre you sure? [y/N]: Deletion cancelled.\n", rootOpts.Stderr.(*bytes.Buffer).String())
	testutil.False(t, deleted, "attachment should NOT have been deleted")
}

func TestRunDeleteAttachment_CancelWithOther(t *testing.T) {
	t.Parallel()
	deleted := false
	server := mockAttachmentServer(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	rootOpts := newTestRootOptions()
	rootOpts.Stdin = strings.NewReader("maybe\n")
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   false,
	}

	err := runDeleteAttachment(context.Background(), "att123", opts)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "", rootOpts.Stdout.(*bytes.Buffer).String())
	testutil.Equal(t, "About to delete attachment: test-file.txt (ID: att123)\nAre you sure? [y/N]: Deletion cancelled.\n", rootOpts.Stderr.(*bytes.Buffer).String())
	testutil.False(t, deleted, "attachment should NOT have been deleted")
}

func TestRunDeleteAttachment_GetAttachmentFails(t *testing.T) {
	t.Parallel()
	server := mockAttachmentServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Attachment not found"}`))
		},
		nil,
	)
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   true,
	}

	err := runDeleteAttachment(context.Background(), "invalid", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "getting attachment")
}

func TestRunDeleteAttachment_DeleteFails(t *testing.T) {
	t.Parallel()
	server := mockAttachmentServer(t, nil,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message": "Permission denied"}`))
		},
	)
	defer server.Close()

	rootOpts := newTestRootOptions()
	client := api.NewClient(server.URL, "user@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{
		Options: rootOpts,
		force:   true,
	}

	err := runDeleteAttachment(context.Background(), "att123", opts)
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "deleting attachment")
}

// TestRunDeleteAttachment_NonInteractive_WithoutForce_ShortCircuits
// pins the §3.4 early-fail contract: the API must NOT be called when
// --non-interactive is set without --force.
func TestRunDeleteAttachment_NonInteractive_WithoutForce_ShortCircuits(t *testing.T) {
	t.Parallel()
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rootOpts := newTestRootOptions()
	rootOpts.NonInteractive = true
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{Options: rootOpts, force: false}
	err := runDeleteAttachment(context.Background(), "att123", opts)
	if err == nil {
		t.Fatal("expected ErrConfirmationRequired")
	}
	if !errors.Is(err, prompt.ErrConfirmationRequired) {
		t.Fatalf("expected prompt.ErrConfirmationRequired, got %v", err)
	}
	if hits != 0 {
		t.Fatalf("API must not be called; got %d hits", hits)
	}
	if rootOpts.Stderr.(*bytes.Buffer).Len() != 0 {
		t.Fatalf("stderr must be empty: %q", rootOpts.Stderr.(*bytes.Buffer).String())
	}
}
