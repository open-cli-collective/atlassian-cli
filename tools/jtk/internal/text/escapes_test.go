package text

import "testing"

func TestInterpretEscapes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no escapes", "hello world", "hello world"},
		{"single newline", `first\nsecond`, "first\nsecond"},
		{"double newline", `first\n\nsecond`, "first\n\nsecond"},
		{"tab", `col1\tcol2`, "col1\tcol2"},
		{"escaped backslash", `path\\to\\file`, `path\to\file`},
		{"mixed escapes", `line1\n\tindented\n\\done`, "line1\n\tindented\n\\done"},
		{"trailing backslash", `ends with\`, `ends with\`},
		{"unknown escape", `\x is unknown`, `\x is unknown`},
		{"empty string", "", ""},
		{"only newlines", `\n\n\n`, "\n\n\n"},
		{"markdown with escapes", `# Title\n\nParagraph\n\n- Item 1\n- Item 2`, "# Title\n\nParagraph\n\n- Item 1\n- Item 2"},
		{"already real newlines", "real\nnewlines", "real\nnewlines"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InterpretEscapes(tt.input)
			if got != tt.want {
				t.Errorf("InterpretEscapes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
