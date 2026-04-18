package api //nolint:revive // package name is intentional

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestGetUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		accountID   string
		response    string
		statusCode  int
		wantErr     bool
		wantDisplay string
	}{
		{
			name:      "successful user lookup",
			accountID: "5b10ac8d82e05b22cc7d4ef5",
			response: `{
				"accountId": "5b10ac8d82e05b22cc7d4ef5",
				"displayName": "John Smith",
				"emailAddress": "john@example.com",
				"active": true
			}`,
			statusCode:  http.StatusOK,
			wantErr:     false,
			wantDisplay: "John Smith",
		},
		{
			name:       "user not found",
			accountID:  "nonexistent",
			response:   `{"errorMessages":["User does not exist"]}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				testutil.Equal(t, r.URL.Path, "/rest/api/3/user")
				testutil.Equal(t, r.URL.Query().Get("accountId"), tt.accountID)
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client, err := New(ClientConfig{
				URL:      "https://test.atlassian.net",
				Email:    "test@example.com",
				APIToken: "test-token",
			})
			testutil.RequireNoError(t, err)
			client.BaseURL = server.URL + "/rest/api/3"

			user, err := client.GetUser(context.Background(), tt.accountID, "")
			if tt.wantErr {
				testutil.Error(t, err)
				return
			}

			testutil.RequireNoError(t, err)
			testutil.Equal(t, user.DisplayName, tt.wantDisplay)
		})
	}
}

func TestGetCurrentUser(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/myself")
		user := User{
			AccountID:    "5b10ac8d82e05b22cc7d4ef5",
			DisplayName:  "Current User",
			EmailAddress: "current@example.com",
			Active:       true,
		}
		_ = json.NewEncoder(w).Encode(user)
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	testutil.RequireNoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	user, err := client.GetCurrentUser(context.Background(), "")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, user.DisplayName, "Current User")
	testutil.Equal(t, user.AccountID, "5b10ac8d82e05b22cc7d4ef5")
}

func TestListUsersPage(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/users")
		testutil.Equal(t, r.URL.Query().Get("startAt"), "50")
		testutil.Equal(t, r.URL.Query().Get("maxResults"), "50")
		users := []User{
			{AccountID: "a1", DisplayName: "User One", Active: true},
			{AccountID: "a2", DisplayName: "User Two", Active: true},
		}
		_ = json.NewEncoder(w).Encode(users)
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	testutil.RequireNoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	users, err := client.ListUsersPage(context.Background(), 50, 50)
	testutil.RequireNoError(t, err)
	testutil.Len(t, users, 2)
	testutil.Equal(t, users[0].AccountID, "a1")
}

func TestSearchUsers(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testutil.Equal(t, r.URL.Path, "/rest/api/3/user/search")
		testutil.Equal(t, r.URL.Query().Get("query"), "john")
		users := []User{
			{
				AccountID:    "5b10ac8d82e05b22cc7d4ef5",
				DisplayName:  "John Smith",
				EmailAddress: "john@example.com",
				Active:       true,
			},
			{
				AccountID:    "5b10ac8d82e05b22cc7d4ef6",
				DisplayName:  "John Doe",
				EmailAddress: "johnd@example.com",
				Active:       true,
			},
		}
		_ = json.NewEncoder(w).Encode(users)
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	testutil.RequireNoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	users, err := client.SearchUsers(context.Background(), "john", 0, 0)
	testutil.RequireNoError(t, err)
	testutil.Len(t, users, 2)
	testutil.Equal(t, users[0].DisplayName, "John Smith")
	testutil.Equal(t, users[1].DisplayName, "John Doe")
}

func TestSearchUsers_SendsStartAtWhenSet(t *testing.T) {
	t.Parallel()
	// The new startAt int parameter gates the "startAt" query param behind a
	// `> 0` check so a zero offset doesn't pollute the URL. This test asserts
	// the positive branch directly at the API layer (TestSearchUsers covers
	// the zero branch implicitly via its omitted startAt query param).
	var capturedStartAt, capturedMaxResults string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedStartAt = r.URL.Query().Get("startAt")
		capturedMaxResults = r.URL.Query().Get("maxResults")
		_ = json.NewEncoder(w).Encode([]User{})
	}))
	defer server.Close()

	client, err := New(ClientConfig{URL: "https://test.atlassian.net", Email: "test@example.com", APIToken: "t"})
	testutil.RequireNoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	_, err = client.SearchUsers(context.Background(), "q", 25, 10)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, capturedStartAt, "25")
	testutil.Equal(t, capturedMaxResults, "10")
}
