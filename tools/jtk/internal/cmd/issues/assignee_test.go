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

	id, err := resolveAssignee(client, "61292e4c4f29230069621c5f")
	require.NoError(t, err)
	assert.Equal(t, "61292e4c4f29230069621c5f", id)
}

func TestResolveAssignee_Me(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/myself" {
			json.NewEncoder(w).Encode(api.User{
				AccountID:   "me-account-id",
				DisplayName: "Current User",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	id, err := resolveAssignee(client, "me")
	require.NoError(t, err)
	assert.Equal(t, "me-account-id", id)
}

func TestResolveAssignee_MeCaseInsensitive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/myself" {
			json.NewEncoder(w).Encode(api.User{
				AccountID:   "me-account-id",
				DisplayName: "Current User",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	id, err := resolveAssignee(client, "Me")
	require.NoError(t, err)
	assert.Equal(t, "me-account-id", id)
}

func TestResolveAssignee_Email(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/user/search" {
			json.NewEncoder(w).Encode([]api.User{
				{AccountID: "email-account-id", DisplayName: "Email User"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	id, err := resolveAssignee(client, "user@example.com")
	require.NoError(t, err)
	assert.Equal(t, "email-account-id", id)
}

func TestResolveAssignee_EmailNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/user/search" {
			json.NewEncoder(w).Encode([]api.User{})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := api.New(api.ClientConfig{
		URL:      server.URL,
		Email:    "test@example.com",
		APIToken: "token",
	})
	require.NoError(t, err)

	_, err = resolveAssignee(client, "nobody@example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no user found")
}
