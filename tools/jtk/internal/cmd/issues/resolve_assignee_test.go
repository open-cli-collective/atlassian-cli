package issues

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestResolveAssignee_RawAccountID(t *testing.T) {
	client, err := api.New(api.ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	accountID, err := resolveAssignee(client, "5b10ac8d82e05b22cc7d4ef5")
	require.NoError(t, err)
	assert.Equal(t, "5b10ac8d82e05b22cc7d4ef5", accountID)
}

func TestResolveAssignee_Me(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/myself", r.URL.Path)
		json.NewEncoder(w).Encode(api.User{
			AccountID:   "abc123",
			DisplayName: "Test User",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	accountID, err := resolveAssignee(client, "me")
	require.NoError(t, err)
	assert.Equal(t, "abc123", accountID)
}

func TestResolveAssignee_MeCaseInsensitive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.User{
			AccountID: "abc123",
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	accountID, err := resolveAssignee(client, "ME")
	require.NoError(t, err)
	assert.Equal(t, "abc123", accountID)
}

func TestResolveAssignee_Email(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/user/search", r.URL.Path)
		assert.Equal(t, "jane@example.com", r.URL.Query().Get("query"))
		json.NewEncoder(w).Encode([]api.User{
			{AccountID: "other123", DisplayName: "Other User", EmailAddress: "other@example.com"},
			{AccountID: "jane456", DisplayName: "Jane Doe", EmailAddress: "jane@example.com"},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	accountID, err := resolveAssignee(client, "jane@example.com")
	require.NoError(t, err)
	assert.Equal(t, "jane456", accountID)
}

func TestResolveAssignee_EmailNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.User{})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	_, err = resolveAssignee(client, "nobody@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no user found with email")
}

func TestResolveAssignee_EmailNoExactMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.User{
			{AccountID: "other123", DisplayName: "Other User", EmailAddress: "similar@example.com"},
		})
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	_, err = resolveAssignee(client, "exact@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no user found with email")
}
