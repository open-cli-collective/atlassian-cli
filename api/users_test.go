package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUser(t *testing.T) {
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/rest/api/3/user", r.URL.Path)
				assert.Equal(t, tt.accountID, r.URL.Query().Get("accountId"))
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client, err := New(ClientConfig{
				URL:      "https://test.atlassian.net",
				Email:    "test@example.com",
				APIToken: "test-token",
			})
			require.NoError(t, err)
			client.BaseURL = server.URL + "/rest/api/3"

			user, err := client.GetUser(tt.accountID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantDisplay, user.DisplayName)
		})
	}
}

func TestGetCurrentUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/myself", r.URL.Path)
		user := User{
			AccountID:    "5b10ac8d82e05b22cc7d4ef5",
			DisplayName:  "Current User",
			EmailAddress: "current@example.com",
			Active:       true,
		}
		json.NewEncoder(w).Encode(user)
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	require.NoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	user, err := client.GetCurrentUser()
	require.NoError(t, err)
	assert.Equal(t, "Current User", user.DisplayName)
	assert.Equal(t, "5b10ac8d82e05b22cc7d4ef5", user.AccountID)
}

func TestSearchUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/3/user/search", r.URL.Path)
		assert.Equal(t, "john", r.URL.Query().Get("query"))
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
		json.NewEncoder(w).Encode(users)
	}))
	defer server.Close()

	client, err := New(ClientConfig{
		URL:      "https://test.atlassian.net",
		Email:    "test@example.com",
		APIToken: "test-token",
	})
	require.NoError(t, err)
	client.BaseURL = server.URL + "/rest/api/3"

	users, err := client.SearchUsers("john", 0)
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "John Smith", users[0].DisplayName)
	assert.Equal(t, "John Doe", users[1].DisplayName)
}
