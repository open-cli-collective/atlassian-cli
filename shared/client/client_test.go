package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/errors"
)

func TestNew(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		c := New("https://example.atlassian.net", "user@example.com", "token", nil)

		if c.BaseURL != "https://example.atlassian.net" {
			t.Errorf("BaseURL = %v, want https://example.atlassian.net", c.BaseURL)
		}

		if !strings.HasPrefix(c.AuthHeader, "Basic ") {
			t.Error("AuthHeader should start with 'Basic '")
		}

		if c.HTTPClient.Timeout != DefaultTimeout {
			t.Errorf("Timeout = %v, want %v", c.HTTPClient.Timeout, DefaultTimeout)
		}
	})

	t.Run("trims trailing slash", func(t *testing.T) {
		c := New("https://example.atlassian.net/", "user@example.com", "token", nil)

		if c.BaseURL != "https://example.atlassian.net" {
			t.Errorf("BaseURL = %v, should not have trailing slash", c.BaseURL)
		}
	})

	t.Run("with options", func(t *testing.T) {
		verboseOut := &bytes.Buffer{}
		opts := &Options{
			Timeout:    60 * time.Second,
			Verbose:    true,
			VerboseOut: verboseOut,
		}

		c := New("https://example.atlassian.net", "user@example.com", "token", opts)

		if c.HTTPClient.Timeout != 60*time.Second {
			t.Errorf("Timeout = %v, want 60s", c.HTTPClient.Timeout)
		}

		if !c.Verbose {
			t.Error("Verbose should be true")
		}

		if c.VerboseOut != verboseOut {
			t.Error("VerboseOut not set correctly")
		}
	})
}

func TestClient_Do(t *testing.T) {
	t.Run("GET request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify headers
			if r.Method != http.MethodGet {
				t.Errorf("Method = %v, want GET", r.Method)
			}

			if auth := r.Header.Get("Authorization"); !strings.HasPrefix(auth, "Basic ") {
				t.Errorf("Authorization header missing or invalid: %v", auth)
			}

			if accept := r.Header.Get("Accept"); accept != "application/json" {
				t.Errorf("Accept = %v, want application/json", accept)
			}

			if r.URL.Path != "/api/test" {
				t.Errorf("Path = %v, want /api/test", r.URL.Path)
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result": "success"}`))
		}))
		defer server.Close()

		c := New(server.URL, "user@example.com", "token", nil)
		body, err := c.Get(context.Background(), "/api/test")

		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if !strings.Contains(string(body), "success") {
			t.Errorf("Body = %v, want to contain 'success'", string(body))
		}
	})

	t.Run("POST request with body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Method = %v, want POST", r.Method)
			}

			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %v, want application/json", ct)
			}

			// Read and verify body
			body, _ := io.ReadAll(r.Body)
			var data map[string]string
			json.Unmarshal(body, &data)

			if data["name"] != "test" {
				t.Errorf("Body name = %v, want test", data["name"])
			}

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id": "123"}`))
		}))
		defer server.Close()

		c := New(server.URL, "user@example.com", "token", nil)
		body, err := c.Post(context.Background(), "/api/create", map[string]string{"name": "test"})

		if err != nil {
			t.Fatalf("Post() error = %v", err)
		}

		if !strings.Contains(string(body), "123") {
			t.Errorf("Body = %v, want to contain '123'", string(body))
		}
	})

	t.Run("PUT request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("Method = %v, want PUT", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := New(server.URL, "user@example.com", "token", nil)
		_, err := c.Put(context.Background(), "/api/update", map[string]string{"name": "updated"})

		if err != nil {
			t.Fatalf("Put() error = %v", err)
		}
	})

	t.Run("DELETE request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("Method = %v, want DELETE", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		c := New(server.URL, "user@example.com", "token", nil)
		_, err := c.Delete(context.Background(), "/api/delete")

		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
	})

	t.Run("path without leading slash", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/test" {
				t.Errorf("Path = %v, want /api/test", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		c := New(server.URL, "user@example.com", "token", nil)
		_, err := c.Get(context.Background(), "api/test") // No leading slash

		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
	})
}

func TestClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    error
	}{
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"message": "Invalid credentials"}`,
			wantErr:    errors.ErrUnauthorized,
		},
		{
			name:       "403 forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"message": "Access denied"}`,
			wantErr:    errors.ErrForbidden,
		},
		{
			name:       "404 not found",
			statusCode: http.StatusNotFound,
			body:       `{"message": "Resource not found"}`,
			wantErr:    errors.ErrNotFound,
		},
		{
			name:       "400 bad request",
			statusCode: http.StatusBadRequest,
			body:       `{"errorMessages": ["Invalid input"]}`,
			wantErr:    errors.ErrBadRequest,
		},
		{
			name:       "429 rate limited",
			statusCode: http.StatusTooManyRequests,
			body:       `{}`,
			wantErr:    errors.ErrRateLimited,
		},
		{
			name:       "500 server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"message": "Internal error"}`,
			wantErr:    errors.ErrServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			c := New(server.URL, "user@example.com", "token", nil)
			_, err := c.Get(context.Background(), "/api/test")

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if !errors.IsNotFound(err) && tt.wantErr == errors.ErrNotFound {
				t.Errorf("Expected ErrNotFound, got %v", err)
			}
			if !errors.IsUnauthorized(err) && tt.wantErr == errors.ErrUnauthorized {
				t.Errorf("Expected ErrUnauthorized, got %v", err)
			}
			if !errors.IsForbidden(err) && tt.wantErr == errors.ErrForbidden {
				t.Errorf("Expected ErrForbidden, got %v", err)
			}
		})
	}
}

func TestClient_VerboseOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	verboseOut := &bytes.Buffer{}
	opts := &Options{
		Verbose:    true,
		VerboseOut: verboseOut,
	}

	c := New(server.URL, "user@example.com", "token", opts)
	_, err := c.Get(context.Background(), "/api/test")

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	output := verboseOut.String()

	if !strings.Contains(output, "→ GET") {
		t.Errorf("Verbose output should contain request: %v", output)
	}

	if !strings.Contains(output, "← 200") {
		t.Errorf("Verbose output should contain response: %v", output)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(server.URL, "user@example.com", "token", nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := c.Get(ctx, "/api/test")

	if err == nil {
		t.Error("Expected error due to cancelled context")
	}
}

func TestOptions_timeoutOrDefault(t *testing.T) {
	t.Run("nil options", func(t *testing.T) {
		var opts *Options
		if got := opts.timeoutOrDefault(); got != DefaultTimeout {
			t.Errorf("timeoutOrDefault() = %v, want %v", got, DefaultTimeout)
		}
	})

	t.Run("zero timeout", func(t *testing.T) {
		opts := &Options{}
		if got := opts.timeoutOrDefault(); got != DefaultTimeout {
			t.Errorf("timeoutOrDefault() = %v, want %v", got, DefaultTimeout)
		}
	})

	t.Run("custom timeout", func(t *testing.T) {
		opts := &Options{Timeout: 60 * time.Second}
		if got := opts.timeoutOrDefault(); got != 60*time.Second {
			t.Errorf("timeoutOrDefault() = %v, want 60s", got)
		}
	})
}
