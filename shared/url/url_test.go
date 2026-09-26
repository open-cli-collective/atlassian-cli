package url //nolint:revive // test file for url package

import "testing"

func TestNormalizeURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "domain only",
			input: "example.atlassian.net",
			want:  "https://example.atlassian.net",
		},
		{
			name:  "with https scheme",
			input: "https://example.atlassian.net",
			want:  "https://example.atlassian.net",
		},
		{
			name:  "with http scheme",
			input: "http://localhost:8080",
			want:  "http://localhost:8080",
		},
		{
			name:  "with trailing slash",
			input: "https://example.com/",
			want:  "https://example.com",
		},
		{
			name:  "with multiple trailing slashes",
			input: "https://example.com///",
			want:  "https://example.com",
		},
		{
			name:  "domain with trailing slash",
			input: "example.atlassian.net/",
			want:  "https://example.atlassian.net",
		},
		{
			name:  "with path",
			input: "https://example.com/wiki",
			want:  "https://example.com/wiki",
		},
		{
			name:  "with path and trailing slash",
			input: "https://example.com/wiki/",
			want:  "https://example.com/wiki",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizeURL(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasScheme(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"example.com", false},
		{"ftp://example.com", false},
		{"", false},
		{"httpx://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := HasScheme(tt.input)
			if got != tt.want {
				t.Errorf("HasScheme(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTrimTrailingSlashes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com/", "https://example.com"},
		{"https://example.com///", "https://example.com"},
		{"https://example.com", "https://example.com"},
		{"https://example.com/path/", "https://example.com/path"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := TrimTrailingSlashes(tt.input)
			if got != tt.want {
				t.Errorf("TrimTrailingSlashes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsLoopbackHTTP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"http://localhost:8080/atlassian", true},
		{"http://LOCALHOST/atlassian", true},
		{"http://127.0.0.1:8080/atlassian", true},
		{"http://127.42.0.1:8080/atlassian", true},
		{"http://[::1]:8080/atlassian", true},
		{"https://localhost:8080/atlassian", false},
		{"http://example.com/atlassian", false},
		{"http://10.0.0.1/atlassian", false},
		{"localhost:8080/atlassian", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := IsLoopbackHTTP(tt.input); got != tt.want {
				t.Errorf("IsLoopbackHTTP(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
