package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetIssueLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issue/PROJ-123", r.URL.Path)
		assert.Equal(t, "issuelinks", r.URL.Query().Get("fields"))

		json.NewEncoder(w).Encode(map[string]interface{}{
			"fields": map[string]interface{}{
				"issuelinks": []map[string]interface{}{
					{
						"id":   "10001",
						"type": map[string]string{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
						"outwardIssue": map[string]interface{}{
							"key": "PROJ-456",
							"fields": map[string]interface{}{
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
	require.NoError(t, err)

	links, err := client.GetIssueLinks("PROJ-123")
	require.NoError(t, err)
	require.Len(t, links, 1)
	assert.Equal(t, "10001", links[0].ID)
	assert.Equal(t, "Blocks", links[0].Type.Name)
	require.NotNil(t, links[0].OutwardIssue)
	assert.Equal(t, "PROJ-456", links[0].OutwardIssue.Key)
}

func TestGetIssueLinks_EmptyKey(t *testing.T) {
	_, err := (&Client{}).GetIssueLinks("")
	assert.Equal(t, ErrIssueKeyRequired, err)
}

func TestCreateIssueLink(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issueLink", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	err = client.CreateIssueLink("PROJ-123", "PROJ-456", "Blocks")
	require.NoError(t, err)

	var req CreateIssueLinkRequest
	err = json.Unmarshal(capturedBody, &req)
	require.NoError(t, err)
	assert.Equal(t, "Blocks", req.Type.Name)
	assert.Equal(t, "PROJ-123", req.OutwardIssue.Key)
	assert.Equal(t, "PROJ-456", req.InwardIssue.Key)
}

func TestCreateIssueLink_EmptyKeys(t *testing.T) {
	assert.Error(t, (&Client{}).CreateIssueLink("", "B", "t"))
	assert.Error(t, (&Client{}).CreateIssueLink("A", "", "t"))
	assert.Error(t, (&Client{}).CreateIssueLink("A", "B", ""))
}

func TestDeleteIssueLink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issueLink/10001", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	err = client.DeleteIssueLink("10001")
	require.NoError(t, err)
}

func TestDeleteIssueLink_EmptyID(t *testing.T) {
	assert.Error(t, (&Client{}).DeleteIssueLink(""))
}

func TestGetIssueLinkTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/issueLinkType", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issueLinkTypes": []map[string]string{
				{"id": "1", "name": "Blocks", "inward": "is blocked by", "outward": "blocks"},
				{"id": "2", "name": "Relates", "inward": "relates to", "outward": "relates to"},
			},
		})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: server.URL, Email: "t@t.com", APIToken: "tok"})
	require.NoError(t, err)

	types, err := client.GetIssueLinkTypes()
	require.NoError(t, err)
	require.Len(t, types, 2)
	assert.Equal(t, "Blocks", types[0].Name)
	assert.Equal(t, "Relates", types[1].Name)
}
