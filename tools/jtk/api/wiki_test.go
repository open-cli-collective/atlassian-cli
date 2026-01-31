package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsWikiMarkup(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "h1 heading",
			input:    "h1. This is a heading",
			expected: true,
		},
		{
			name:     "h2 heading",
			input:    "h2. Another heading",
			expected: true,
		},
		{
			name:     "monospace",
			input:    "Some {{inline code}} here",
			expected: true,
		},
		{
			name:     "code block",
			input:    "{code:java}\npublic class Test {}\n{code}",
			expected: true,
		},
		{
			name:     "wiki link",
			input:    "Check out [this link|https://example.com]",
			expected: true,
		},
		{
			name:     "wiki image",
			input:    "See !screenshot.png!",
			expected: true,
		},
		{
			name:     "blockquote",
			input:    "bq. This is a quote",
			expected: true,
		},
		{
			name:     "noformat block",
			input:    "{noformat}some text{noformat}",
			expected: true,
		},
		{
			name:     "quote block",
			input:    "{quote}quoted text{quote}",
			expected: true,
		},
		{
			name:     "plain markdown",
			input:    "# Heading\n\nSome **bold** text",
			expected: false,
		},
		{
			name:     "markdown link",
			input:    "Check [this](https://example.com)",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "plain text",
			input:    "Just some plain text without any formatting",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWikiMarkup(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWikiToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "h1 heading",
			input:    "h1. Main Title",
			expected: "# Main Title",
		},
		{
			name:     "h2 heading",
			input:    "h2. Section",
			expected: "## Section",
		},
		{
			name:     "h3 heading",
			input:    "h3. Subsection",
			expected: "### Subsection",
		},
		{
			name:     "multiple headings",
			input:    "h1. Title\nh2. Section\nh3. Subsection",
			expected: "# Title\n## Section\n### Subsection",
		},
		{
			name:     "monospace",
			input:    "Use {{git status}} to check",
			expected: "Use `git status` to check",
		},
		{
			name:     "code block without language",
			input:    "{code}\nfunc main() {}\n{code}",
			expected: "```\nfunc main() {}\n```",
		},
		{
			name:     "code block with language",
			input:    "{code:go}\nfunc main() {}\n{code}",
			expected: "```go\nfunc main() {}\n```",
		},
		{
			name:     "noformat block",
			input:    "{noformat}\nsome preformatted text\n{noformat}",
			expected: "```\nsome preformatted text\n```",
		},
		{
			name:     "wiki link",
			input:    "See [Google|https://google.com] for more",
			expected: "See [Google](https://google.com) for more",
		},
		{
			name:     "wiki image",
			input:    "Screenshot: !image.png!",
			expected: "Screenshot: ![](image.png)",
		},
		{
			name:     "wiki image with alt",
			input:    "!diagram.png|alt=Architecture!",
			expected: "![Architecture](diagram.png)",
		},
		{
			name:     "blockquote line",
			input:    "bq. This is quoted",
			expected: "> This is quoted",
		},
		{
			name:     "quote block",
			input:    "{quote}\nFirst line\nSecond line\n{quote}",
			expected: "> First line\n> Second line",
		},
		{
			name:     "bullet list",
			input:    "* Item 1\n* Item 2\n* Item 3",
			expected: "- Item 1\n- Item 2\n- Item 3",
		},
		{
			name:     "nested bullet list",
			input:    "* Item 1\n** Nested 1\n** Nested 2\n* Item 2",
			expected: "- Item 1\n  - Nested 1\n  - Nested 2\n- Item 2",
		},
		{
			name:     "horizontal rule",
			input:    "Before\n----\nAfter",
			expected: "Before\n---\nAfter",
		},
		{
			name:     "complex document",
			input:    "h1. Guide\n\nThis is about {{code}}.\n\n{code:python}\nprint('hello')\n{code}\n\nSee [docs|https://example.com].",
			expected: "# Guide\n\nThis is about `code`.\n\n```python\nprint('hello')\n```\n\nSee [docs](https://example.com).",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WikiToMarkdown(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWikiToMarkdownPreservesMarkdown(t *testing.T) {
	// Markdown input should pass through mostly unchanged
	// (some edge cases may have minor differences)
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "markdown heading",
			input: "# Title",
		},
		{
			name:  "markdown bold",
			input: "Some **bold** text",
		},
		{
			name:  "markdown code",
			input: "Use `code` here",
		},
		{
			name:  "markdown link",
			input: "[Google](https://google.com)",
		},
		{
			name:  "markdown list",
			input: "- Item 1\n- Item 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WikiToMarkdown(tt.input)
			assert.Equal(t, tt.input, result)
		})
	}
}

func TestMarkdownToADFWithWikiMarkup(t *testing.T) {
	// Test that wiki markup is properly converted when passed to MarkdownToADF
	tests := []struct {
		name      string
		input     string
		checkType string
		checkAttr interface{}
	}{
		{
			name:      "wiki h1 becomes ADF heading",
			input:     "h1. Hello World",
			checkType: "heading",
			checkAttr: 1,
		},
		{
			name:      "wiki h2 becomes ADF heading",
			input:     "h2. Section Title",
			checkType: "heading",
			checkAttr: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := MarkdownToADF(tt.input)
			assert.NotNil(t, doc)
			assert.Equal(t, "doc", doc.Type)
			assert.NotEmpty(t, doc.Content)

			if tt.checkType == "heading" {
				assert.Equal(t, "heading", doc.Content[0].Type)
				assert.Equal(t, tt.checkAttr, doc.Content[0].Attrs["level"])
			}
		})
	}
}
