package space

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

// TestRunDelete_NonInteractive_WithoutForce_ShortCircuits pins the §3.4
// early-fail contract: cfl space delete must surface
// ErrConfirmationRequired BEFORE the API GetSpaceByKey call AND before
// view.ValidateFormat — neither should win precedence over the
// confirmation gate.
func TestRunDelete_NonInteractive_WithoutForce_ShortCircuits(t *testing.T) {
	t.Parallel()
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rootOpts := &root.Options{
		Output:         "table",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{Options: rootOpts, force: false}
	err := runDelete(context.Background(), "SPACE", opts)
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
	if rootOpts.Stdout.(*bytes.Buffer).Len() != 0 {
		t.Fatalf("stdout must be empty: %q", rootOpts.Stdout.(*bytes.Buffer).String())
	}
}

// TestRunDelete_NonInteractive_WithoutForce_ShortCircuitsBeforeValidateFormat
// pins the positioning: even with an invalid --output, the
// confirmation error wins first under --non-interactive without --force.
// A regression that re-orders the checks would cause this test to
// surface a "format not supported" error instead.
func TestRunDelete_NonInteractive_WithoutForce_ShortCircuitsBeforeValidateFormat(t *testing.T) {
	t.Parallel()
	rootOpts := &root.Options{
		Output:         "bogus-format",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	opts := &deleteOptions{Options: rootOpts, force: false}
	err := runDelete(context.Background(), "SPACE", opts)
	if !errors.Is(err, prompt.ErrConfirmationRequired) {
		t.Fatalf("expected prompt.ErrConfirmationRequired (must win over format-validation), got %v", err)
	}
}

// TestRunDelete_NonInteractive_WithForce_Proceeds — --force still
// bypasses confirmation under --non-interactive.
func TestRunDelete_NonInteractive_WithForce_Proceeds(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			// GetSpaceByKey internally calls ListSpaces — must return a list.
			_, _ = w.Write([]byte(`{"results":[{"id":"123","key":"SPACE","name":"Test Space"}]}`))
			return
		}
		// DELETE
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	rootOpts := &root.Options{
		Output:         "table",
		NoColor:        true,
		NonInteractive: true,
		Stdin:          strings.NewReader(""),
		Stdout:         &bytes.Buffer{},
		Stderr:         &bytes.Buffer{},
	}
	client := api.NewClient(server.URL, "test@example.com", "token")
	rootOpts.SetAPIClient(client)

	opts := &deleteOptions{Options: rootOpts, force: true}
	err := runDelete(context.Background(), "SPACE", opts)
	testutil.RequireNoError(t, err)
}
