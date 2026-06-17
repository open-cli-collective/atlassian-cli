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

func TestGetRemoteLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issue/PROJ-123/remotelink")
		testutil.Equal(t, r.Method, http.MethodGet)

		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":           10001,
				"self":         "https://acme.atlassian.net/rest/api/3/issue/PROJ-123/remotelink/10001",
				"relationship": "mentioned in",
				"object": map[string]any{
					"url":     "https://github.com/owner/repo/issues/456",
					"title":   "GitHub #456",
					"summary": "Some issue",
				},
			},
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	links, err := client.GetRemoteLinks(context.Background(), "PROJ-123")
	testutil.RequireNoError(t, err)
	testutil.Len(t, links, 1)
	testutil.Equal(t, links[0].ID, 10001)
	testutil.Equal(t, links[0].Object.URL, "https://github.com/owner/repo/issues/456")
	testutil.Equal(t, links[0].Object.Title, "GitHub #456")
	testutil.Equal(t, links[0].Relationship, "mentioned in")
}

func TestGetRemoteLinks_EmptyKey(t *testing.T) {
	_, err := (&Client{}).GetRemoteLinks(context.Background(), "")
	testutil.Equal(t, err, ErrIssueKeyRequired)
}

func TestAddRemoteLink(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issue/PROJ-123/remotelink")
		testutil.Equal(t, r.Method, http.MethodPost)
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   10010,
			"self": "https://acme.atlassian.net/rest/api/3/issue/PROJ-123/remotelink/10010",
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	req := CreateRemoteLinkRequest{
		Object: RemoteLinkObject{
			URL:   "https://example.com/page",
			Title: "Example",
		},
	}
	link, err := client.AddRemoteLink(context.Background(), "PROJ-123", req)
	testutil.RequireNoError(t, err)
	// The create response is slim; the returned link echoes the request object.
	testutil.Equal(t, link.ID, 10010)
	testutil.Equal(t, link.Object.URL, "https://example.com/page")
	testutil.Equal(t, link.Object.Title, "Example")

	var sent CreateRemoteLinkRequest
	err = json.Unmarshal(capturedBody, &sent)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, sent.Object.URL, "https://example.com/page")
	testutil.Equal(t, sent.Object.Title, "Example")
}

func TestAddRemoteLink_EmptyKey(t *testing.T) {
	_, err := (&Client{}).AddRemoteLink(context.Background(), "", CreateRemoteLinkRequest{
		Object: RemoteLinkObject{URL: "https://example.com"},
	})
	testutil.Equal(t, err, ErrIssueKeyRequired)
}

func TestAddRemoteLink_EmptyURL(t *testing.T) {
	_, err := (&Client{}).AddRemoteLink(context.Background(), "PROJ-123", CreateRemoteLinkRequest{})
	testutil.Equal(t, err, ErrRemoteLinkURLRequired)
}

func TestAddRemoteLink_EmptyTitle(t *testing.T) {
	_, err := (&Client{}).AddRemoteLink(context.Background(), "PROJ-123", CreateRemoteLinkRequest{
		Object: RemoteLinkObject{URL: "https://example.com"},
	})
	testutil.Equal(t, err, ErrRemoteLinkTitleRequired)
}

func TestDeleteRemoteLink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issue/PROJ-123/remotelink/10001")
		testutil.Equal(t, r.Method, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	err = client.DeleteRemoteLink(context.Background(), "PROJ-123", 10001)
	testutil.RequireNoError(t, err)
}

func TestDeleteRemoteLink_EmptyArgs(t *testing.T) {
	testutil.Equal(t, (&Client{}).DeleteRemoteLink(context.Background(), "", 10001), ErrIssueKeyRequired)
	testutil.Equal(t, (&Client{}).DeleteRemoteLink(context.Background(), "PROJ-123", 0), ErrRemoteLinkIDRequired)
	testutil.Equal(t, (&Client{}).DeleteRemoteLink(context.Background(), "PROJ-123", -1), ErrRemoteLinkIDRequired)
}
