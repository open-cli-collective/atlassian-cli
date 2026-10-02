package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestUpdateComment(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issue/PROJ-123/comment/10001")
		testutil.Equal(t, r.Method, http.MethodPut)
		capturedBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "10001",
			"author":  map[string]any{"displayName": "Alice"},
			"created": "2024-06-15T10:00:00.000+0000",
			"updated": "2024-06-16T10:00:00.000+0000",
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	comment, err := client.UpdateComment(context.Background(), "PROJ-123", "10001", "Corrected text")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, comment.ID, "10001")
	testutil.Equal(t, comment.Author.DisplayName, "Alice")

	var req map[string]any
	testutil.RequireNoError(t, json.Unmarshal(capturedBody, &req))
	body := req["body"].(map[string]any)
	testutil.Equal(t, body["type"], "doc")
	testutil.Len(t, body["content"].([]any), 1)
}

func TestUpdateComment_EmptyIssueKey(t *testing.T) {
	_, err := (&Client{}).UpdateComment(context.Background(), "", "10001", "x")
	testutil.Equal(t, err, ErrIssueKeyRequired)
}

func TestUpdateComment_EmptyCommentID(t *testing.T) {
	_, err := (&Client{}).UpdateComment(context.Background(), "PROJ-123", "", "x")
	testutil.Equal(t, err, ErrCommentIDRequired)
}
