package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToConfluenceStorage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "basic paragraph",
			input:    "Hello world",
			expected: "<p>Hello world</p>\n",
		},
		{
			name:     "multiple paragraphs",
			input:    "First paragraph.\n\nSecond paragraph.",
			expected: "<p>First paragraph.</p>\n<p>Second paragraph.</p>\n",
		},
		{
			name:     "h1 header",
			input:    "# Title",
			expected: "<h1>Title</h1>\n",
		},
		{
			name:     "h2 header",
			input:    "## Subtitle",
			expected: "<h2>Subtitle</h2>\n",
		},
		{
			name:     "h3 header",
			input:    "### Section",
			expected: "<h3>Section</h3>\n",
		},
		{
			name:     "bold text",
			input:    "This is **bold** text",
			expected: "<p>This is <strong>bold</strong> text</p>\n",
		},
		{
			name:     "italic text",
			input:    "This is *italic* text",
			expected: "<p>This is <em>italic</em> text</p>\n",
		},
		{
			name:     "bold and italic",
			input:    "**bold** and *italic*",
			expected: "<p><strong>bold</strong> and <em>italic</em></p>\n",
		},
		{
			name:     "unordered list",
			input:    "- Item 1\n- Item 2\n- Item 3",
			expected: "<ul>\n<li>Item 1</li>\n<li>Item 2</li>\n<li>Item 3</li>\n</ul>\n",
		},
		{
			name:     "ordered list",
			input:    "1. First\n2. Second\n3. Third",
			expected: "<ol>\n<li>First</li>\n<li>Second</li>\n<li>Third</li>\n</ol>\n",
		},
		{
			name:     "inline code",
			input:    "Use `code` here",
			expected: "<p>Use <code>code</code> here</p>\n",
		},
		{
			name:     "code block",
			input:    "```\ncode here\n```",
			expected: "<pre><code>code here\n</code></pre>\n",
		},
		{
			name:     "code block with language",
			input:    "```go\nfunc main() {}\n```",
			expected: "<pre><code class=\"language-go\">func main() {}\n</code></pre>\n",
		},
		{
			name:     "link",
			input:    "[Google](https://google.com)",
			expected: "<p><a href=\"https://google.com\">Google</a></p>\n",
		},
		{
			name:     "blockquote",
			input:    "> This is a quote",
			expected: "<blockquote>\n<p>This is a quote</p>\n</blockquote>\n",
		},
		{
			name:     "horizontal rule",
			input:    "---",
			expected: "<hr>\n",
		},
		{
			name:     "simple table",
			input:    "| A | B |\n|---|---|\n| 1 | 2 |",
			expected: "<table>\n<thead>\n<tr>\n<th>A</th>\n<th>B</th>\n</tr>\n</thead>\n<tbody>\n<tr>\n<td>1</td>\n<td>2</td>\n</tr>\n</tbody>\n</table>\n",
		},
		{
			name:     "table with multiple rows",
			input:    "| Name | Age |\n|------|-----|\n| Alice | 30 |\n| Bob | 25 |",
			expected: "<table>\n<thead>\n<tr>\n<th>Name</th>\n<th>Age</th>\n</tr>\n</thead>\n<tbody>\n<tr>\n<td>Alice</td>\n<td>30</td>\n</tr>\n<tr>\n<td>Bob</td>\n<td>25</td>\n</tr>\n</tbody>\n</table>\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToConfluenceStorage([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToConfluenceStorage_ComplexDocument(t *testing.T) {
	input := `# Project README

This is the **introduction** to the project.

## Features

- Feature one
- Feature two
- Feature three

## Code Example

` + "```go" + `
func hello() {
    fmt.Println("Hello")
}
` + "```" + `

For more info, see [the docs](https://example.com).
`

	result, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)

	// Verify key elements are present
	assert.Contains(t, result, "<h1>Project README</h1>")
	assert.Contains(t, result, "<strong>introduction</strong>")
	assert.Contains(t, result, "<h2>Features</h2>")
	assert.Contains(t, result, "<li>Feature one</li>")
	assert.Contains(t, result, "<h2>Code Example</h2>")
	assert.Contains(t, result, "language-go")
	assert.Contains(t, result, "fmt.Println")
	assert.Contains(t, result, `<a href="https://example.com">the docs</a>`)
}
