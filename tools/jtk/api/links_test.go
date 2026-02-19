package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestGetIssueLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issue/PROJ-123")
		testutil.Equal(t, r.URL.Query().Get("fields"), "issuelinks")

		_ = json.NewEncoder(w).Encode(map[string]any{
			"fields": map[string]any{
				"issuelinks": []map[string]any{
					{
						"id":   "10001",
						"type": map[string]string{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
						"outwardIssue": map[string]any{
							"key": "PROJ-456",
							"fields": map[string]any{
								"summary": "Other issue",
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	links, err := client.GetIssueLinks("PROJ-123")
	testutil.RequireNoError(t, err)
	testutil.Len(t, links, 1)
	testutil.Equal(t, links[0].ID, "10001")
	testutil.Equal(t, links[0].Type.Name, "Blocks")
	testutil.NotNil(t, links[0].OutwardIssue)
	testutil.Equal(t, links[0].OutwardIssue.Key, "PROJ-456")
}

func TestGetIssueLinks_EmptyKey(t *testing.T) {
	_, err := (&Client{}).GetIssueLinks("")
	testutil.Equal(t, err, ErrIssueKeyRequired)
}

func TestCreateIssueLink(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issueLink")
		testutil.Equal(t, r.Method, "POST")
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	err = client.CreateIssueLink("PROJ-123", "PROJ-456", "Blocks")
	testutil.RequireNoError(t, err)

	var req CreateIssueLinkRequest
	err = json.Unmarshal(capturedBody, &req)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, req.Type.Name, "Blocks")
	testutil.Equal(t, req.OutwardIssue.Key, "PROJ-123")
	testutil.Equal(t, req.InwardIssue.Key, "PROJ-456")
}

func TestCreateIssueLink_EmptyKeys(t *testing.T) {
	testutil.Error(t, (&Client{}).CreateIssueLink("", "B", "t"))
	testutil.Error(t, (&Client{}).CreateIssueLink("A", "", "t"))
	testutil.Error(t, (&Client{}).CreateIssueLink("A", "B", ""))
}

func TestDeleteIssueLink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issueLink/10001")
		testutil.Equal(t, r.Method, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	err = client.DeleteIssueLink("10001")
	testutil.RequireNoError(t, err)
}

func TestDeleteIssueLink_EmptyID(t *testing.T) {
	testutil.Error(t, (&Client{}).DeleteIssueLink(""))
}

func TestGetIssueLinkTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/issueLinkType")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issueLinkTypes": []map[string]string{
				{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
				{"id": "2", "name": "Relates", "inward": "relates to", "outward": "relates to"},
			},
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	testutil.RequireNoError(t, err)

	types, err := client.GetIssueLinkTypes()
	testutil.RequireNoError(t, err)
	testutil.Len(t, types, 2)
	testutil.Equal(t, types[0].Name, "Blocks")
	testutil.Equal(t, types[1].Name, "Relates")
}
