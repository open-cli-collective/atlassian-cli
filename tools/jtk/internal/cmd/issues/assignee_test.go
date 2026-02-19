package issues

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestResolveAssignee_RawAccountID(t *testing.T) {
	t.Parallel()
	client, err := api.New(api.ClientConfig{
		URL:      "http://unused",
		Email:    "test@example.com",
		APIToken: "token",
	})
	testutil.RequireNoError(t, err)

	id, err := resolveAssignee(context.Background(), client, "61292e4c4f29230069621c5f")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, id, "61292e4c4f29230069621c5f")
}

func TestResolveAssignee_Me(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/myself" {
			_ = json.NewEncoder(w).Encode(api.User{
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
	testutil.RequireNoError(t, err)

	id, err := resolveAssignee(context.Background(), client, "me")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, id, "me-account-id")
}

func TestResolveAssignee_MeCaseInsensitive(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/myself" {
			_ = json.NewEncoder(w).Encode(api.User{
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
	testutil.RequireNoError(t, err)

	id, err := resolveAssignee(context.Background(), client, "Me")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, id, "me-account-id")
}

func TestResolveAssignee_Email(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/user/search" {
			_ = json.NewEncoder(w).Encode([]api.User{
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
	testutil.RequireNoError(t, err)

	id, err := resolveAssignee(context.Background(), client, "user@example.com")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, id, "email-account-id")
}

func TestResolveAssignee_EmailNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/3/user/search" {
			_ = json.NewEncoder(w).Encode([]api.User{})
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
	testutil.RequireNoError(t, err)

	_, err = resolveAssignee(context.Background(), client, "nobody@example.com")
	testutil.Error(t, err)
	testutil.Contains(t, err.Error(), "no user found")
}
