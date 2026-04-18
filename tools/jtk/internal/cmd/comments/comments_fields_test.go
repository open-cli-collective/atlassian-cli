package comments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// longBodyText is sized past PresentList's 100-char body truncation so the
// truncation marker is observable in table-mode tests.
const longBodyText = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"

func longBodyComment(id, author string) api.Comment {
	return plainComment(id, author, longBodyText)
}

// AC2 (table mode): --fields ID,AUTHOR drops the BODY column entirely. Long
// body text MUST NOT appear anywhere in the output.
func TestRunList_Fields_TableMode_DropsBodyColumn(t *testing.T) {
	t.Parallel()
	server := newTestCommentsServer(t, []api.Comment{longBodyComment("1", "Alice")})
	defer server.Close()

	opts, stdout, _ := newCommentsOpts(t, server)
	err := runList(context.Background(), opts, "TEST-1", 50, false, "ID,AUTHOR")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "ID")
	testutil.Contains(t, output, "AUTHOR")
	testutil.Contains(t, output, "Alice")
	if strings.Contains(output, "BODY") {
		t.Errorf("BODY header should be absent: %q", output)
	}
	if strings.Contains(output, "BBB") {
		t.Errorf("body text leaked into projected table output: %q", output)
	}
}

// AC3 (table mode): selecting BODY without --fulltext keeps the existing
// 100-char truncation. Projection must not silently bypass truncation.
func TestRunList_Fields_TableMode_BodyTruncatedWithoutFullText(t *testing.T) {
	t.Parallel()
	server := newTestCommentsServer(t, []api.Comment{longBodyComment("1", "Alice")})
	defer server.Close()

	opts, stdout, _ := newCommentsOpts(t, server)
	err := runList(context.Background(), opts, "TEST-1", 50, false, "ID,AUTHOR,BODY")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "BODY")
	testutil.Contains(t, output, "[truncated, use --fulltext for complete text]")
	testutil.NotContains(t, output, longBodyText)
}

// AC2 (block mode): --fields ID,Author with --fulltext drops Body from each
// per-comment DetailSection.
func TestRunList_Fields_BlockMode_DropsBodyField(t *testing.T) {
	t.Parallel()
	server := newTestCommentsServer(t, []api.Comment{longBodyComment("1", "Alice")})
	defer server.Close()

	opts, stdout, _ := newCommentsOpts(t, server)
	err := runList(context.Background(), opts, "TEST-1", 50, true, "ID,Author")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "ID:")
	testutil.Contains(t, output, "Author:")
	if strings.Contains(output, "Body:") {
		t.Errorf("Body label should be absent: %q", output)
	}
	if strings.Contains(output, "BBB") {
		t.Errorf("body text leaked into projected block output: %q", output)
	}
}

// AC1+AC3 (block mode): --fields Body with --fulltext renders the full body
// without the truncation marker.
func TestRunList_Fields_BlockMode_BodySelectedWithFullText(t *testing.T) {
	t.Parallel()
	server := newTestCommentsServer(t, []api.Comment{longBodyComment("1", "Alice")})
	defer server.Close()

	opts, stdout, _ := newCommentsOpts(t, server)
	err := runList(context.Background(), opts, "TEST-1", 50, true, "Body")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	testutil.Contains(t, output, "Body:")
	testutil.Contains(t, output, longBodyText)
	testutil.NotContains(t, output, "[truncated")
}

// AC3 (block mode): --fulltext is a no-op for unselected fields. Even with
// fulltext on, body text MUST NOT appear when Body is not in --fields.
func TestRunList_Fields_BlockMode_FullTextNoOp_WhenBodyNotSelected(t *testing.T) {
	t.Parallel()
	server := newTestCommentsServer(t, []api.Comment{longBodyComment("1", "Alice")})
	defer server.Close()

	opts, stdout, _ := newCommentsOpts(t, server)
	err := runList(context.Background(), opts, "TEST-1", 50, true, "ID,Author")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	if strings.Contains(output, "BBB") {
		t.Errorf("body text leaked even though Body not selected: %q", output)
	}
}

// AC2 + helper coverage: in block mode with hasMore=true, projecting Body
// out of every comment must not strip the trailing pagination MessageSection.
// Guards against projectAllDetailSectionsInModel accidentally rewriting
// non-Detail sections.
func TestRunList_Fields_BlockMode_PreservesPaginationHint(t *testing.T) {
	t.Parallel()
	// Total=2 with one comment returned forces commentsHasMore to true.
	server := commentsServerWithTotal([]api.Comment{longBodyComment("1", "Alice")}, 2)
	defer server.Close()

	opts, stdout, _ := newCommentsOpts(t, server)
	err := runList(context.Background(), opts, "TEST-1", 50, true, "ID,Author")
	testutil.RequireNoError(t, err)

	output := stdout.String()
	if strings.Contains(output, "Body:") {
		t.Errorf("Body label should be projected away: %q", output)
	}
	testutil.Contains(t, output, "More results available")
}

// Unknown --fields token must surface as UnknownFieldError. The no-op
// fetcher prevents UnrenderedFieldError or a real /field call.
func TestRunList_Fields_UnknownToken_Errors(t *testing.T) {
	t.Parallel()
	server := newTestCommentsServer(t, []api.Comment{longBodyComment("1", "Alice")})
	defer server.Close()

	opts, _, _ := newCommentsOpts(t, server)
	err := runList(context.Background(), opts, "TEST-1", 50, false, "bogus")
	var ufe *projection.UnknownFieldError
	if !errors.As(err, &ufe) {
		t.Fatalf("expected UnknownFieldError, got %v", err)
	}
}

// Validation (Resolve) must happen BEFORE the GetComments network call.
// An invalid --fields token must produce UnknownFieldError without making
// any API requests — the flag is a display contract validated on the
// client before any fetch, mirroring the runGet ordering.
func TestRunList_Fields_InvalidToken_ErrorsBeforeFetch(t *testing.T) {
	t.Parallel()
	var commentCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/comment") {
			commentCalls++
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(api.CommentsResponse{})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{URL: server.URL, Email: "e@x", APIToken: "t"})
	testutil.RequireNoError(t, err)
	opts := &root.Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	opts.SetAPIClient(client)

	runErr := runList(context.Background(), opts, "TEST-1", 50, false, "bogus")
	var ufe *projection.UnknownFieldError
	if !errors.As(runErr, &ufe) {
		t.Fatalf("expected UnknownFieldError, got %v", runErr)
	}
	if commentCalls != 0 {
		t.Errorf("GetComments should not be called when --fields is invalid; got %d call(s)", commentCalls)
	}
}

// --id wins over --fields: bare IDs only, no projection or metadata work.
func TestRunList_IDOnly_OverridesFields(t *testing.T) {
	t.Parallel()
	server := newTestCommentsServer(t, []api.Comment{longBodyComment("1", "Alice")})
	defer server.Close()

	opts, stdout, _ := newCommentsOpts(t, server)
	opts.IDOnly = true
	err := runList(context.Background(), opts, "TEST-1", 50, false, "Body")
	testutil.RequireNoError(t, err)

	if stdout.String() != "1\n" {
		t.Errorf("expected bare ID, got %q", stdout.String())
	}
}

// --fields combined with --output json is rejected, mirroring runGet.
func TestRunList_Fields_WithJSON_Errors(t *testing.T) {
	t.Parallel()
	server := newTestCommentsServer(t, []api.Comment{longBodyComment("1", "Alice")})
	defer server.Close()

	opts, _, _ := newCommentsOpts(t, server)
	opts.Output = "json"
	err := runList(context.Background(), opts, "TEST-1", 50, false, "ID")
	if err == nil {
		t.Fatalf("expected error when --fields combined with --output json")
	}
	testutil.Contains(t, err.Error(), "not supported with --output json")
}

// --id combined with --output json + --fields must let --id win without
// triggering the JSON-vs-fields rejection (parity with runGet).
func TestRunList_IDOnly_BypassesJSONFieldsRejection(t *testing.T) {
	t.Parallel()
	server := newTestCommentsServer(t, []api.Comment{longBodyComment("1", "Alice")})
	defer server.Close()

	opts, stdout, _ := newCommentsOpts(t, server)
	opts.IDOnly = true
	opts.Output = "json"
	err := runList(context.Background(), opts, "TEST-1", 50, false, "Body")
	testutil.RequireNoError(t, err)
	if stdout.String() != "1\n" {
		t.Errorf("expected bare ID, got %q", stdout.String())
	}
}
