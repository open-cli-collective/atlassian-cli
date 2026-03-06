package auth

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestBasicAuthHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		email    string
		apiToken string
		want     string
	}{
		{
			name:     "standard credentials",
			email:    "user@example.com",
			apiToken: "secret-token",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:secret-token")),
		},
		{
			name:     "empty email",
			email:    "",
			apiToken: "token",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte(":token")),
		},
		{
			name:     "empty token",
			email:    "user@example.com",
			apiToken: "",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:")),
		},
		{
			name:     "special characters in token",
			email:    "user@example.com",
			apiToken: "token+with/special=chars",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:token+with/special=chars")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BasicAuthHeader(tt.email, tt.apiToken)
			if got != tt.want {
				t.Errorf("BasicAuthHeader() = %v, want %v", got, tt.want)
			}

			// Verify it starts with "Basic "
			if !strings.HasPrefix(got, "Basic ") {
				t.Error("BasicAuthHeader() should start with 'Basic '")
			}

			// Verify the encoded part is valid base64
			encoded := strings.TrimPrefix(got, "Basic ")
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				t.Errorf("BasicAuthHeader() returned invalid base64: %v", err)
			}

			// Verify the decoded value contains the email and token
			expectedDecoded := tt.email + ":" + tt.apiToken
			if string(decoded) != expectedDecoded {
				t.Errorf("Decoded value = %v, want %v", string(decoded), expectedDecoded)
			}
		})
	}
}

func TestBearerAuthHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		apiToken string
		want     string
	}{
		{
			name:     "standard token",
			apiToken: "secret-token",
			want:     "Bearer secret-token",
		},
		{
			name:     "empty token",
			apiToken: "",
			want:     "Bearer ",
		},
		{
			name:     "special characters in token",
			apiToken: "token+with/special=chars",
			want:     "Bearer token+with/special=chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BearerAuthHeader(tt.apiToken)
			if got != tt.want {
				t.Errorf("BearerAuthHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAuthMethod(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{name: "basic is valid", method: "basic", wantErr: false},
		{name: "bearer is valid", method: "bearer", wantErr: false},
		{name: "empty string is invalid", method: "", wantErr: true},
		{name: "capitalized Bearer is invalid", method: "Bearer", wantErr: true},
		{name: "unknown method is invalid", method: "oauth", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateAuthMethod(tt.method)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAuthMethod(%q) = nil, want error", tt.method)
				}
				if !errors.Is(err, ErrInvalidAuthMethod) {
					t.Errorf("ValidateAuthMethod(%q) error = %v, want ErrInvalidAuthMethod", tt.method, err)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAuthMethod(%q) = %v, want nil", tt.method, err)
				}
			}
		})
	}
}

func TestAuthMethodConstants(t *testing.T) {
	t.Parallel()
	if AuthMethodBasic != "basic" {
		t.Errorf("AuthMethodBasic = %q, want %q", AuthMethodBasic, "basic")
	}
	if AuthMethodBearer != "bearer" {
		t.Errorf("AuthMethodBearer = %q, want %q", AuthMethodBearer, "bearer")
	}
}
