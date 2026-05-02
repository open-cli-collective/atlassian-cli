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
	t.Parallel()
	t.Run("basic creation", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
		c := New("https://example.atlassian.net/", "user@example.com", "token", nil)

		if c.BaseURL != "https://example.atlassian.net" {
			t.Errorf("BaseURL = %v, should not have trailing slash", c.BaseURL)
		}
	})

	t.Run("with options", func(t *testing.T) {
		t.Parallel()
		verboseOut := &bytes.Buffer{}
		opts := &Options{
			Timeout:    90 * time.Second,
			Verbose:    true,
			VerboseOut: verboseOut,
		}

		c := New("https://example.atlassian.net", "user@example.com", "token", opts)

		if c.HTTPClient.Timeout != 90*time.Second {
			t.Errorf("Timeout = %v, want 90s", c.HTTPClient.Timeout)
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
	t.Parallel()
	t.Run("GET request", func(t *testing.T) {
		t.Parallel()
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
			_, _ = w.Write([]byte(`{"result": "success"}`))
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
		t.Parallel()
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
			_ = json.Unmarshal(body, &data)

			if data["name"] != "test" {
				t.Errorf("Body name = %v, want test", data["name"])
			}

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id": "123"}`))
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
		t.Parallel()
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
		t.Parallel()
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
		t.Parallel()
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

	t.Run("absolute URL bypasses BaseURL", func(t *testing.T) {
		t.Parallel()
		// Create a server that the client will actually hit
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/custom/endpoint" {
				t.Errorf("Path = %v, want /custom/endpoint", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success": true}`))
		}))
		defer server.Close()

		// Create client with a DIFFERENT base URL
		c := New("https://different-host.example.com", "user@example.com", "token", nil)
		// Override HTTPClient to use test server
		c.HTTPClient = server.Client()

		// Pass an absolute URL - should ignore BaseURL and use this directly
		absoluteURL := server.URL + "/custom/endpoint"
		body, err := c.Get(context.Background(), absoluteURL)

		if err != nil {
			t.Fatalf("Get() with absolute URL error = %v", err)
		}

		if !strings.Contains(string(body), "success") {
			t.Errorf("Body = %v, want to contain 'success'", string(body))
		}
	})
}

func TestClient_ErrorHandling(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			c := New(server.URL, "user@example.com", "token", nil)
			_, err := c.Get(context.Background(), "/api/test")

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if errors.IsNotFound(tt.wantErr) && !errors.IsNotFound(err) {
				t.Errorf("Expected ErrNotFound, got %v", err)
			}
			if errors.IsUnauthorized(tt.wantErr) && !errors.IsUnauthorized(err) {
				t.Errorf("Expected ErrUnauthorized, got %v", err)
			}
			if errors.IsForbidden(tt.wantErr) && !errors.IsForbidden(err) {
				t.Errorf("Expected ErrForbidden, got %v", err)
			}
		})
	}
}

func TestClient_VerboseOutput(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
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

func TestClient_VerboseLogsRequestBody(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	verboseOut := &bytes.Buffer{}
	c := New(server.URL, "user@example.com", "token", &Options{Verbose: true, VerboseOut: verboseOut})

	body := map[string]string{"hello": "world"}
	if _, err := c.Post(context.Background(), "/api/test", body); err != nil {
		t.Fatalf("Post() error = %v", err)
	}

	output := verboseOut.String()
	if !strings.Contains(output, `→ body: {"hello":"world"}`) {
		t.Errorf("expected request body in verbose output, got: %v", output)
	}
}

func TestClient_VerboseSkipsBodyWhenNil(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	verboseOut := &bytes.Buffer{}
	c := New(server.URL, "user@example.com", "token", &Options{Verbose: true, VerboseOut: verboseOut})

	if _, err := c.Get(context.Background(), "/api/test"); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if strings.Contains(verboseOut.String(), "→ body:") {
		t.Errorf("GET should not log a request body, got: %v", verboseOut.String())
	}
}

func TestClient_VerboseLogsErrorResponseBody(t *testing.T) {
	t.Parallel()
	errorPayload := `{"errorMessages":["INVALID_INPUT"],"errors":{"description":"bad ADF"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(errorPayload))
	}))
	defer server.Close()

	verboseOut := &bytes.Buffer{}
	c := New(server.URL, "user@example.com", "token", &Options{Verbose: true, VerboseOut: verboseOut})

	_, err := c.Post(context.Background(), "/api/test", map[string]string{"foo": "bar"})
	if err == nil {
		t.Fatal("expected error from 400 response")
	}

	output := verboseOut.String()
	if !strings.Contains(output, "← body: "+errorPayload) {
		t.Errorf("expected error response body in verbose output, got: %v", output)
	}
}

func TestClient_VerboseSkipsResponseBodyOn2xx(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	verboseOut := &bytes.Buffer{}
	c := New(server.URL, "user@example.com", "token", &Options{Verbose: true, VerboseOut: verboseOut})

	if _, err := c.Get(context.Background(), "/api/test"); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if strings.Contains(verboseOut.String(), "← body:") {
		t.Errorf("2xx should not log response body, got: %v", verboseOut.String())
	}
}

func TestClient_VerboseTruncatesLargeBodies(t *testing.T) {
	t.Parallel()
	bigField := strings.Repeat("A", 5000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"big":"` + bigField + `"}`))
	}))
	defer server.Close()

	verboseOut := &bytes.Buffer{}
	c := New(server.URL, "user@example.com", "token", &Options{Verbose: true, VerboseOut: verboseOut})

	_, _ = c.Post(context.Background(), "/api/test", map[string]string{"big": bigField})

	output := verboseOut.String()
	if !strings.Contains(output, "...[truncated]") {
		t.Errorf("expected truncation suffix in verbose output, got: %v", output)
	}
	// Both the request body line and the response body line should be capped.
	for _, prefix := range []string{"→ body: ", "← body: "} {
		idx := strings.Index(output, prefix)
		if idx < 0 {
			t.Errorf("expected %q in output, got: %v", prefix, output)
			continue
		}
		end := strings.Index(output[idx:], "\n")
		if end < 0 {
			t.Fatalf("no newline after %q", prefix)
		}
		line := output[idx+len(prefix) : idx+end]
		const maxLineLen = maxVerboseBodyLog + len("...[truncated]")
		if len(line) > maxLineLen {
			t.Errorf("%s line len = %d, want <= %d", prefix, len(line), maxLineLen)
		}
	}
}

func TestTruncateForLog(t *testing.T) {
	t.Parallel()
	if got := truncateForLog([]byte("short")); string(got) != "short" {
		t.Errorf("under cap: got %q, want %q", got, "short")
	}
	// Exact-cap boundary: input length == maxVerboseBodyLog must NOT be truncated.
	atCap := bytes.Repeat([]byte("Y"), maxVerboseBodyLog)
	if got := truncateForLog(atCap); len(got) != maxVerboseBodyLog || bytes.Contains(got, []byte("[truncated]")) {
		t.Errorf("at cap: expected pass-through (len=%d, no suffix), got len=%d truncated=%v", maxVerboseBodyLog, len(got), bytes.Contains(got, []byte("[truncated]")))
	}
	big := bytes.Repeat([]byte("X"), maxVerboseBodyLog+10)
	got := truncateForLog(big)
	if !bytes.HasSuffix(got, []byte("...[truncated]")) {
		t.Errorf("expected truncated suffix, got %q", got)
	}
	if len(got) != maxVerboseBodyLog+len("...[truncated]") {
		t.Errorf("len = %d, want %d", len(got), maxVerboseBodyLog+len("...[truncated]"))
	}
}

func TestClient_VerboseLogsResponseBodyOn5xx(t *testing.T) {
	t.Parallel()
	errorPayload := `{"errorMessages":["Internal server error"]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(errorPayload))
	}))
	defer server.Close()

	verboseOut := &bytes.Buffer{}
	c := New(server.URL, "user@example.com", "token", &Options{Verbose: true, VerboseOut: verboseOut})

	if _, err := c.Get(context.Background(), "/api/test"); err == nil {
		t.Fatal("expected error from 500 response")
	}

	if !strings.Contains(verboseOut.String(), "← body: "+errorPayload) {
		t.Errorf("expected 5xx response body in verbose output, got: %v", verboseOut.String())
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func TestNew_AuthHeaderOverride(t *testing.T) {
	t.Parallel()
	t.Run("options AuthHeader overrides Basic auth", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer my-scoped-token" {
				t.Errorf("Authorization = %v, want Bearer my-scoped-token", auth)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}))
		defer server.Close()

		opts := &Options{AuthHeader: "Bearer my-scoped-token"}
		c := New(server.URL, "", "", opts)

		if c.AuthHeader != "Bearer my-scoped-token" {
			t.Errorf("AuthHeader = %v, want Bearer my-scoped-token", c.AuthHeader)
		}

		_, err := c.Get(context.Background(), "/api/test")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
	})

	t.Run("empty options AuthHeader falls back to Basic auth", func(t *testing.T) {
		t.Parallel()
		opts := &Options{}
		c := New("https://example.atlassian.net", "user@example.com", "token", opts)

		if !strings.HasPrefix(c.AuthHeader, "Basic ") {
			t.Errorf("AuthHeader = %v, should start with 'Basic ' when Options.AuthHeader is empty", c.AuthHeader)
		}
	})

	t.Run("nil options uses Basic auth", func(t *testing.T) {
		t.Parallel()
		c := New("https://example.atlassian.net", "user@example.com", "token", nil)

		if !strings.HasPrefix(c.AuthHeader, "Basic ") {
			t.Errorf("AuthHeader = %v, should start with 'Basic ' when opts is nil", c.AuthHeader)
		}
	})
}

func TestOptions_timeoutOrDefault(t *testing.T) {
	t.Parallel()
	t.Run("nil options", func(t *testing.T) {
		t.Parallel()
		var opts *Options
		if got := opts.timeoutOrDefault(); got != DefaultTimeout {
			t.Errorf("timeoutOrDefault() = %v, want %v", got, DefaultTimeout)
		}
	})

	t.Run("zero timeout", func(t *testing.T) {
		t.Parallel()
		opts := &Options{}
		if got := opts.timeoutOrDefault(); got != DefaultTimeout {
			t.Errorf("timeoutOrDefault() = %v, want %v", got, DefaultTimeout)
		}
	})

	t.Run("custom timeout", func(t *testing.T) {
		t.Parallel()
		opts := &Options{Timeout: 90 * time.Second}
		if got := opts.timeoutOrDefault(); got != 90*time.Second {
			t.Errorf("timeoutOrDefault() = %v, want 90s", got)
		}
	})
}
