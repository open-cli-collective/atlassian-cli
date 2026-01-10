package attachment

import (
	"path/filepath"
	"testing"
)

// TestSanitizeAttachmentFilename validates that filepath.Base correctly
// sanitizes malicious filenames that could be used for path traversal attacks.
func TestSanitizeAttachmentFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		valid    bool
	}{
		{
			name:     "normal filename",
			input:    "document.pdf",
			expected: "document.pdf",
			valid:    true,
		},
		{
			name:     "path traversal unix",
			input:    "../../../etc/passwd",
			expected: "passwd",
			valid:    true,
		},
		{
			name:     "path traversal with subdirectory",
			input:    "subdir/../../../etc/passwd",
			expected: "passwd",
			valid:    true,
		},
		{
			name:     "absolute path unix",
			input:    "/etc/passwd",
			expected: "passwd",
			valid:    true,
		},
		{
			name:     "filename with spaces",
			input:    "my document.pdf",
			expected: "my document.pdf",
			valid:    true,
		},
		{
			name:     "empty filename",
			input:    "",
			expected: ".",
			valid:    false,
		},
		{
			name:     "single dot",
			input:    ".",
			expected: ".",
			valid:    false,
		},
		{
			name:     "double dot",
			input:    "..",
			expected: "..",
			valid:    false,
		},
		{
			name:     "trailing slash",
			input:    "folder/",
			expected: "folder",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filepath.Base(tt.input)
			if result != tt.expected {
				t.Errorf("filepath.Base(%q) = %q, want %q", tt.input, result, tt.expected)
			}

			// Validate that our invalid filename check works
			isValid := result != "" && result != "." && result != ".."
			if isValid != tt.valid {
				t.Errorf("validity check for %q: got %v, want %v", tt.input, isValid, tt.valid)
			}
		})
	}
}
