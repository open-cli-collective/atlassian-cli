package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestGetIssueChangelog_RequestAndDecode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/rest/api/3/issue/TEST-1/changelog", r.URL.EscapedPath())
		testutil.Equal(t, "5", r.URL.Query().Get("startAt"))
		testutil.Equal(t, "2", r.URL.Query().Get("maxResults"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"startAt": 5,
			"maxResults": 2,
			"total": 9,
			"values": [
				{
					"id": "10001",
					"created": "2026-06-20T15:04:05.000+0000",
					"author": {"accountId": "acct-1", "displayName": "Alice"},
					"items": [
						{
							"field": "status",
							"fieldtype": "jira",
							"fieldId": "status",
							"from": "1",
							"fromString": "Open",
							"to": "3",
							"toString": "Done"
						}
					]
				}
			]
		}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	page, err := client.GetIssueChangelog(context.Background(), "TEST-1", IssueChangelogOptions{StartAt: 5, MaxResults: 2})
	testutil.RequireNoError(t, err)

	testutil.Equal(t, 5, page.StartAt)
	testutil.Equal(t, 2, page.MaxResults)
	testutil.Equal(t, 9, page.Total)
	testutil.Equal(t, 1, len(page.Histories))
	testutil.Equal(t, "10001", page.Histories[0].ID)
	testutil.Equal(t, "Alice", page.Histories[0].Author.DisplayName)
	testutil.Equal(t, "Done", page.Histories[0].Items[0].ToString)
}

func TestGetIssueChangelog_PathEscapesIssueKey(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, "/rest/api/3/issue/TEST%2F1/changelog", r.URL.EscapedPath())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"startAt":0,"maxResults":50,"total":0,"values":[]}`))
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	_, err = client.GetIssueChangelog(context.Background(), "TEST/1", IssueChangelogOptions{})
	testutil.RequireNoError(t, err)
}

func TestGetIssueChangelog_EmptyIssueKey(t *testing.T) {
	t.Parallel()

	client, err := New(ClientConfig{URL: "https://example.atlassian.net", Email: "test@example.com", APIToken: "token"})
	testutil.RequireNoError(t, err)

	_, err = client.GetIssueChangelog(context.Background(), "", IssueChangelogOptions{})
	if !errors.Is(err, ErrIssueKeyRequired) {
		t.Fatalf("got %v, want ErrIssueKeyRequired", err)
	}
}
