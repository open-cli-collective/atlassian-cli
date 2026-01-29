package auth

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBasicAuthHeader(t *testing.T) {
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
