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

func TestToConfluenceStorage_TOCMacro(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "simple TOC",
			input: "[TOC]",
			contains: []string{
				`<ac:structured-macro ac:name="toc" ac:schema-version="1">`,
				`</ac:structured-macro>`,
			},
		},
		{
			name:  "TOC with single parameter",
			input: "[TOC maxLevel=3]",
			contains: []string{
				`<ac:structured-macro ac:name="toc" ac:schema-version="1">`,
				`<ac:parameter ac:name="maxLevel">3</ac:parameter>`,
				`</ac:structured-macro>`,
			},
		},
		{
			name:  "TOC with multiple parameters",
			input: "[TOC maxLevel=3 minLevel=1]",
			contains: []string{
				`<ac:structured-macro ac:name="toc" ac:schema-version="1">`,
				`<ac:parameter ac:name="maxLevel">3</ac:parameter>`,
				`<ac:parameter ac:name="minLevel">1</ac:parameter>`,
				`</ac:structured-macro>`,
			},
		},
		{
			name:  "TOC case insensitive - lowercase",
			input: "[toc]",
			contains: []string{
				`<ac:structured-macro ac:name="toc" ac:schema-version="1">`,
			},
		},
		{
			name:  "TOC case insensitive - mixed case",
			input: "[Toc maxLevel=2]",
			contains: []string{
				`<ac:structured-macro ac:name="toc" ac:schema-version="1">`,
				`<ac:parameter ac:name="maxLevel">2</ac:parameter>`,
			},
		},
		{
			name:  "TOC with all common parameters",
			input: "[TOC maxLevel=4 minLevel=2 type=flat outline=true separator=pipe]",
			contains: []string{
				`<ac:parameter ac:name="maxLevel">4</ac:parameter>`,
				`<ac:parameter ac:name="minLevel">2</ac:parameter>`,
				`<ac:parameter ac:name="type">flat</ac:parameter>`,
				`<ac:parameter ac:name="outline">true</ac:parameter>`,
				`<ac:parameter ac:name="separator">pipe</ac:parameter>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToConfluenceStorage([]byte(tt.input))
			require.NoError(t, err)
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected, "should contain: %s", expected)
			}
		})
	}
}

func TestToConfluenceStorage_TOCMixedWithContent(t *testing.T) {
	input := `[TOC maxLevel=3]

# Heading 1

Some content here.

## Heading 2

More content.
`
	result, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)

	// Verify TOC macro is present
	assert.Contains(t, result, `<ac:structured-macro ac:name="toc" ac:schema-version="1">`)
	assert.Contains(t, result, `<ac:parameter ac:name="maxLevel">3</ac:parameter>`)
	assert.Contains(t, result, `</ac:structured-macro>`)

	// Verify other content is preserved
	assert.Contains(t, result, "<h1>Heading 1</h1>")
	assert.Contains(t, result, "Some content here.")
	assert.Contains(t, result, "<h2>Heading 2</h2>")
}

func TestToConfluenceStorage_TOCRoundtrip(t *testing.T) {
	// Test that TOC can survive a roundtrip conversion
	// Start with Confluence storage format with TOC
	originalXHTML := `<p>Before</p>
<ac:structured-macro ac:name="toc" ac:schema-version="1">
<ac:parameter ac:name="maxLevel">3</ac:parameter>
<ac:parameter ac:name="minLevel">1</ac:parameter>
</ac:structured-macro>
<h1>Title</h1>
<p>Content</p>`

	// Convert to markdown with ShowMacros
	opts := ConvertOptions{ShowMacros: true}
	markdown, err := FromConfluenceStorageWithOptions(originalXHTML, opts)
	require.NoError(t, err)

	// Verify markdown has TOC placeholder with params
	assert.Contains(t, markdown, "[TOC")
	assert.Contains(t, markdown, "maxLevel=3")
	assert.Contains(t, markdown, "minLevel=1")

	// Convert back to storage format
	resultXHTML, err := ToConfluenceStorage([]byte(markdown))
	require.NoError(t, err)

	// Verify TOC macro is restored
	assert.Contains(t, resultXHTML, `<ac:structured-macro ac:name="toc" ac:schema-version="1">`)
	assert.Contains(t, resultXHTML, `<ac:parameter ac:name="maxLevel">3</ac:parameter>`)
	assert.Contains(t, resultXHTML, `<ac:parameter ac:name="minLevel">1</ac:parameter>`)
}

func TestParseKeyValueParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single param",
			input:    "key=value",
			expected: []string{"key=value"},
		},
		{
			name:     "multiple params",
			input:    "key1=value1 key2=value2",
			expected: []string{"key1=value1", "key2=value2"},
		},
		{
			name:     "quoted value with spaces",
			input:    `title="Hello World"`,
			expected: []string{"title=Hello World"},
		},
		{
			name:     "mixed quoted and unquoted",
			input:    `maxLevel=3 title="My Title" type=flat`,
			expected: []string{"maxLevel=3", "title=My Title", "type=flat"},
		},
		{
			name:     "single quoted value",
			input:    `title='Hello World'`,
			expected: []string{"title=Hello World"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKeyValueParams(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToConfluenceStorage_PanelMacros(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "simple info panel",
			input: "[INFO]\nThis is info content.\n[/INFO]",
			contains: []string{
				`<ac:structured-macro ac:name="info" ac:schema-version="1">`,
				`<ac:rich-text-body>`,
				`This is info content.`,
				`</ac:rich-text-body>`,
				`</ac:structured-macro>`,
			},
		},
		{
			name:  "warning panel with title",
			input: `[WARNING title="Watch out"]` + "\nBe careful here.\n[/WARNING]",
			contains: []string{
				`<ac:structured-macro ac:name="warning" ac:schema-version="1">`,
				`<ac:parameter ac:name="title">Watch out</ac:parameter>`,
				`<ac:rich-text-body>`,
				`Be careful here.`,
				`</ac:rich-text-body>`,
				`</ac:structured-macro>`,
			},
		},
		{
			name:  "note panel",
			input: "[NOTE]\nNote content.\n[/NOTE]",
			contains: []string{
				`<ac:structured-macro ac:name="note" ac:schema-version="1">`,
				`Note content.`,
				`</ac:structured-macro>`,
			},
		},
		{
			name:  "tip panel",
			input: "[TIP]\nTip content.\n[/TIP]",
			contains: []string{
				`<ac:structured-macro ac:name="tip" ac:schema-version="1">`,
				`Tip content.`,
				`</ac:structured-macro>`,
			},
		},
		{
			name:  "expand panel with title",
			input: `[EXPAND title="Click to expand"]` + "\nHidden content.\n[/EXPAND]",
			contains: []string{
				`<ac:structured-macro ac:name="expand" ac:schema-version="1">`,
				`<ac:parameter ac:name="title">Click to expand</ac:parameter>`,
				`Hidden content.`,
				`</ac:structured-macro>`,
			},
		},
		{
			name:  "panel case insensitive - lowercase",
			input: "[info]\nContent.\n[/info]",
			contains: []string{
				`<ac:structured-macro ac:name="info" ac:schema-version="1">`,
				`Content.`,
				`</ac:structured-macro>`,
			},
		},
		{
			name:  "panel case insensitive - mixed case",
			input: "[Info]\nContent.\n[/Info]",
			contains: []string{
				`<ac:structured-macro ac:name="info" ac:schema-version="1">`,
				`Content.`,
				`</ac:structured-macro>`,
			},
		},
		{
			name:  "panel with markdown content",
			input: "[INFO]\nThis is **bold** and *italic*.\n[/INFO]",
			contains: []string{
				`<ac:structured-macro ac:name="info" ac:schema-version="1">`,
				`<strong>bold</strong>`,
				`<em>italic</em>`,
				`</ac:structured-macro>`,
			},
		},
		{
			name:  "panel with list content",
			input: "[NOTE]\n- Item 1\n- Item 2\n[/NOTE]",
			contains: []string{
				`<ac:structured-macro ac:name="note" ac:schema-version="1">`,
				`<li>Item 1</li>`,
				`<li>Item 2</li>`,
				`</ac:structured-macro>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToConfluenceStorage([]byte(tt.input))
			require.NoError(t, err)
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected, "should contain: %s", expected)
			}
		})
	}
}

func TestToConfluenceStorage_PanelMixedWithContent(t *testing.T) {
	input := `# Heading

Some intro text.

[WARNING title="Important"]
This is a warning.
[/WARNING]

More text after.
`
	result, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)

	// Verify all parts are present
	assert.Contains(t, result, "<h1>Heading</h1>")
	assert.Contains(t, result, "Some intro text.")
	assert.Contains(t, result, `<ac:structured-macro ac:name="warning" ac:schema-version="1">`)
	assert.Contains(t, result, `<ac:parameter ac:name="title">Important</ac:parameter>`)
	assert.Contains(t, result, "This is a warning.")
	assert.Contains(t, result, "</ac:structured-macro>")
	assert.Contains(t, result, "More text after.")
}

func TestToConfluenceStorage_PanelRoundtrip(t *testing.T) {
	// Test that panel can survive a roundtrip conversion
	// Use a simple title without spaces to avoid quoting complexity
	originalXHTML := `<p>Before</p>
<ac:structured-macro ac:name="info" ac:schema-version="1">
<ac:parameter ac:name="title">Important</ac:parameter>
<ac:rich-text-body><p>Panel content here.</p></ac:rich-text-body>
</ac:structured-macro>
<p>After</p>`

	// Convert to markdown with ShowMacros
	opts := ConvertOptions{ShowMacros: true}
	markdown, err := FromConfluenceStorageWithOptions(originalXHTML, opts)
	require.NoError(t, err)

	// Verify markdown has panel placeholder (brackets may be escaped by markdown converter)
	assert.Contains(t, markdown, "INFO")
	assert.Contains(t, markdown, "title=Important")
	assert.Contains(t, markdown, "Panel content")

	// Convert back to storage format
	resultXHTML, err := ToConfluenceStorage([]byte(markdown))
	require.NoError(t, err)

	// Verify panel macro is restored
	assert.Contains(t, resultXHTML, `<ac:structured-macro ac:name="info" ac:schema-version="1">`)
	assert.Contains(t, resultXHTML, `<ac:parameter ac:name="title">Important</ac:parameter>`)
	assert.Contains(t, resultXHTML, `<ac:rich-text-body>`)
	assert.Contains(t, resultXHTML, `Panel content`)
	assert.Contains(t, resultXHTML, `</ac:rich-text-body>`)
}

func TestToConfluenceStorage_NestedMacros(t *testing.T) {
	// Test nested TOC inside INFO panel
	input := `[INFO]
Check out the table of contents: [TOC]
[/INFO]`

	result, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)

	// The result should have both the panel macro and TOC macro
	assert.Contains(t, result, `<ac:structured-macro ac:name="info"`)
	assert.Contains(t, result, `<ac:structured-macro ac:name="toc"`)

	// Make sure placeholders are not left behind
	assert.NotContains(t, result, "CFMACRO")
	assert.NotContains(t, result, "END")
}
