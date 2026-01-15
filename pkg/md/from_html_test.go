package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromConfluenceStorage(t *testing.T) {
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
			input:    "<p>Hello world</p>",
			expected: "Hello world",
		},
		{
			name:     "multiple paragraphs",
			input:    "<p>First paragraph.</p><p>Second paragraph.</p>",
			expected: "First paragraph.\n\nSecond paragraph.",
		},
		{
			name:     "h1 header",
			input:    "<h1>Title</h1>",
			expected: "# Title",
		},
		{
			name:     "h2 header",
			input:    "<h2>Subtitle</h2>",
			expected: "## Subtitle",
		},
		{
			name:     "h3 header",
			input:    "<h3>Section</h3>",
			expected: "### Section",
		},
		{
			name:     "bold text",
			input:    "<p>This is <strong>bold</strong> text</p>",
			expected: "This is **bold** text",
		},
		{
			name:     "italic text",
			input:    "<p>This is <em>italic</em> text</p>",
			expected: "This is *italic* text",
		},
		{
			name:     "unordered list",
			input:    "<ul><li>Item 1</li><li>Item 2</li></ul>",
			expected: "- Item 1\n- Item 2",
		},
		{
			name:     "ordered list",
			input:    "<ol><li>First</li><li>Second</li></ol>",
			expected: "1. First\n2. Second",
		},
		{
			name:     "inline code",
			input:    "<p>Use <code>code</code> here</p>",
			expected: "Use `code` here",
		},
		{
			name:     "code block",
			input:    "<pre><code>code here</code></pre>",
			expected: "```\ncode here\n```",
		},
		{
			name:     "link",
			input:    `<p><a href="https://google.com">Google</a></p>`,
			expected: "[Google](https://google.com)",
		},
		{
			name:     "blockquote",
			input:    "<blockquote><p>This is a quote</p></blockquote>",
			expected: "> This is a quote",
		},
		{
			name:     "simple table",
			input:    "<table><tr><th>Name</th><th>Age</th></tr><tr><td>Alice</td><td>30</td></tr></table>",
			expected: "| Name  | Age |\n|-------|-----|\n| Alice | 30  |",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FromConfluenceStorage(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromConfluenceStorage_ConfluenceCodeMacro(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "code macro with language",
			input: `<ac:structured-macro ac:name="code" ac:schema-version="1">
				<ac:parameter ac:name="language">python</ac:parameter>
				<ac:plain-text-body><![CDATA[print("Hello World")]]></ac:plain-text-body>
			</ac:structured-macro>`,
			contains: []string{"```python", `print("Hello World")`, "```"},
		},
		{
			name: "code macro without language",
			input: `<ac:structured-macro ac:name="code" ac:schema-version="1">
				<ac:plain-text-body><![CDATA[some code here]]></ac:plain-text-body>
			</ac:structured-macro>`,
			contains: []string{"```", "some code here"},
		},
		{
			name: "code macro with special characters",
			input: `<ac:structured-macro ac:name="code" ac:schema-version="1">
				<ac:parameter ac:name="language">go</ac:parameter>
				<ac:plain-text-body><![CDATA[if x < 10 && y > 5 {
	fmt.Println("test")
}]]></ac:plain-text-body>
			</ac:structured-macro>`,
			contains: []string{"```go", "if x < 10 && y > 5", "```"},
		},
		{
			name: "multiple code macros",
			input: `<p>First code:</p>
			<ac:structured-macro ac:name="code" ac:schema-version="1">
				<ac:parameter ac:name="language">bash</ac:parameter>
				<ac:plain-text-body><![CDATA[echo "hello"]]></ac:plain-text-body>
			</ac:structured-macro>
			<p>Second code:</p>
			<ac:structured-macro ac:name="code" ac:schema-version="1">
				<ac:parameter ac:name="language">python</ac:parameter>
				<ac:plain-text-body><![CDATA[print("world")]]></ac:plain-text-body>
			</ac:structured-macro>`,
			contains: []string{"```bash", `echo "hello"`, "```python", `print("world")`},
		},
		{
			name: "code macro mixed with other content",
			input: `<h2>Example</h2>
			<ac:structured-macro ac:name="code" ac:schema-version="1">
				<ac:parameter ac:name="language">javascript</ac:parameter>
				<ac:plain-text-body><![CDATA[console.log("test");]]></ac:plain-text-body>
			</ac:structured-macro>
			<p>That's the code.</p>`,
			contains: []string{"## Example", "```javascript", `console.log("test");`, "That's the code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FromConfluenceStorage(tt.input)
			require.NoError(t, err)
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected, "should contain: %s", expected)
			}
		})
	}
}

func TestFromConfluenceStorage_Tables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "table with thead and tbody",
			input: `<table>
				<thead><tr><th>Header 1</th><th>Header 2</th></tr></thead>
				<tbody><tr><td>Cell 1</td><td>Cell 2</td></tr></tbody>
			</table>`,
			contains: []string{"| Header 1", "| Header 2", "| Cell 1", "| Cell 2", "|----------|"},
		},
		{
			name: "table with multiple rows",
			input: `<table>
				<tr><th>Name</th><th>Age</th><th>City</th></tr>
				<tr><td>Alice</td><td>30</td><td>NYC</td></tr>
				<tr><td>Bob</td><td>25</td><td>LA</td></tr>
				<tr><td>Charlie</td><td>35</td><td>Chicago</td></tr>
			</table>`,
			contains: []string{"| Name", "| Age", "| City", "| Alice", "| Bob", "| Charlie"},
		},
		{
			name: "table mixed with other content",
			input: `<h2>Data Table</h2>
			<p>Here is some data:</p>
			<table>
				<tr><th>Item</th><th>Price</th></tr>
				<tr><td>Apple</td><td>$1.00</td></tr>
			</table>
			<p>End of table.</p>`,
			contains: []string{"## Data Table", "Here is some data", "| Item", "| Price", "| Apple", "$1.00", "End of table"},
		},
		{
			name: "empty table cells",
			input: `<table>
				<tr><th>A</th><th>B</th></tr>
				<tr><td></td><td>Value</td></tr>
			</table>`,
			contains: []string{"| A", "| B", "| Value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FromConfluenceStorage(tt.input)
			require.NoError(t, err)
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected, "should contain: %s", expected)
			}
		})
	}
}

func TestFromConfluenceStorage_NonCodeMacrosStripped(t *testing.T) {
	// Non-code macros should still be stripped
	input := `<p>Before</p>
	<ac:structured-macro ac:name="toc" ac:schema-version="1">
		<ac:parameter ac:name="maxLevel">3</ac:parameter>
	</ac:structured-macro>
	<p>After</p>`

	result, err := FromConfluenceStorage(input)
	require.NoError(t, err)
	assert.Contains(t, result, "Before")
	assert.Contains(t, result, "After")
	assert.NotContains(t, result, "toc")
	assert.NotContains(t, result, "maxLevel")
}

func TestFromConfluenceStorage_TOCWithShowMacros(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "TOC without parameters",
			input: `<p>Before</p>
			<ac:structured-macro ac:name="toc" ac:schema-version="1">
			</ac:structured-macro>
			<p>After</p>`,
			expected: "[TOC]",
		},
		{
			name: "TOC with single parameter",
			input: `<p>Before</p>
			<ac:structured-macro ac:name="toc" ac:schema-version="1">
				<ac:parameter ac:name="maxLevel">3</ac:parameter>
			</ac:structured-macro>
			<p>After</p>`,
			expected: "[TOC maxLevel=3]",
		},
		{
			name: "TOC with multiple parameters",
			input: `<p>Before</p>
			<ac:structured-macro ac:name="toc" ac:schema-version="1">
				<ac:parameter ac:name="maxLevel">3</ac:parameter>
				<ac:parameter ac:name="minLevel">1</ac:parameter>
				<ac:parameter ac:name="type">flat</ac:parameter>
			</ac:structured-macro>
			<p>After</p>`,
			expected: "[TOC maxLevel=3 minLevel=1 type=flat]",
		},
		{
			name: "TOC with outline parameter",
			input: `<ac:structured-macro ac:name="toc" ac:schema-version="1">
				<ac:parameter ac:name="outline">true</ac:parameter>
				<ac:parameter ac:name="separator">pipe</ac:parameter>
			</ac:structured-macro>`,
			expected: "[TOC outline=true separator=pipe]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ConvertOptions{ShowMacros: true}
			result, err := FromConfluenceStorageWithOptions(tt.input, opts)
			require.NoError(t, err)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestFromConfluenceStorage_PanelMacrosWithShowMacros(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "info macro with body",
			input: `<ac:structured-macro ac:name="info" ac:schema-version="1">
				<ac:rich-text-body><p>Info content</p></ac:rich-text-body>
			</ac:structured-macro>`,
			contains: []string{"[INFO]", "Info content", "[/INFO]"},
		},
		{
			name: "warning macro with title and body",
			input: `<ac:structured-macro ac:name="warning" ac:schema-version="1">
				<ac:parameter ac:name="title">Watch out</ac:parameter>
				<ac:rich-text-body><p>Warning content</p></ac:rich-text-body>
			</ac:structured-macro>`,
			// Title with space gets quoted
			contains: []string{`[WARNING title="Watch out"]`, "Warning content", "[/WARNING]"},
		},
		{
			name: "note macro",
			input: `<ac:structured-macro ac:name="note" ac:schema-version="1">
				<ac:rich-text-body><p>Note content</p></ac:rich-text-body>
			</ac:structured-macro>`,
			contains: []string{"[NOTE]", "Note content", "[/NOTE]"},
		},
		{
			name: "tip macro",
			input: `<ac:structured-macro ac:name="tip" ac:schema-version="1">
				<ac:rich-text-body><p>Tip content</p></ac:rich-text-body>
			</ac:structured-macro>`,
			contains: []string{"[TIP]", "Tip content", "[/TIP]"},
		},
		{
			name: "expand macro with title and body",
			input: `<ac:structured-macro ac:name="expand" ac:schema-version="1">
				<ac:parameter ac:name="title">Click to expand</ac:parameter>
				<ac:rich-text-body><p>Hidden content</p></ac:rich-text-body>
			</ac:structured-macro>`,
			contains: []string{`[EXPAND title="Click to expand"]`, "Hidden content", "[/EXPAND]"},
		},
		{
			name: "panel with title containing spaces",
			input: `<ac:structured-macro ac:name="info" ac:schema-version="1">
				<ac:parameter ac:name="title">Important Information</ac:parameter>
				<ac:rich-text-body><p>Content here</p></ac:rich-text-body>
			</ac:structured-macro>`,
			contains: []string{`[INFO title="Important Information"]`, "Content here", "[/INFO]"},
		},
		{
			name: "panel with empty body",
			input: `<ac:structured-macro ac:name="info" ac:schema-version="1">
				<ac:rich-text-body></ac:rich-text-body>
			</ac:structured-macro>`,
			contains: []string{"[INFO]", "[/INFO]"},
		},
		{
			name: "panel with formatted body content",
			input: `<ac:structured-macro ac:name="warning" ac:schema-version="1">
				<ac:rich-text-body><p>This is <strong>bold</strong> and <em>italic</em> text.</p></ac:rich-text-body>
			</ac:structured-macro>`,
			// Body HTML is converted to markdown
			contains: []string{"[WARNING]", "**bold**", "*italic*", "[/WARNING]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ConvertOptions{ShowMacros: true}
			result, err := FromConfluenceStorageWithOptions(tt.input, opts)
			require.NoError(t, err)
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected, "should contain: %s", expected)
			}
		})
	}
}

func TestFromConfluenceStorage_ComplexDocument(t *testing.T) {
	input := `<h1>Project README</h1>
<p>This is the <strong>introduction</strong> to the project.</p>
<h2>Features</h2>
<ul>
<li>Feature one</li>
<li>Feature two</li>
<li>Feature three</li>
</ul>
<h2>Code Example</h2>
<pre><code class="language-go">func hello() {
    fmt.Println("Hello")
}</code></pre>
<p>For more info, see <a href="https://example.com">the docs</a>.</p>`

	result, err := FromConfluenceStorage(input)
	require.NoError(t, err)

	// Verify key elements are present
	assert.Contains(t, result, "# Project README")
	assert.Contains(t, result, "**introduction**")
	assert.Contains(t, result, "## Features")
	assert.Contains(t, result, "- Feature one")
	assert.Contains(t, result, "## Code Example")
	assert.Contains(t, result, "```")
	assert.Contains(t, result, "fmt.Println")
	assert.Contains(t, result, "[the docs](https://example.com)")
}

func TestFromConfluenceStorage_NestedMacros(t *testing.T) {
	// Test nested TOC inside INFO panel
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1">
<ac:rich-text-body>
<p>Here is the table of contents:</p>
<p><ac:structured-macro ac:name="toc" ac:schema-version="1">
<ac:parameter ac:name="maxLevel">2</ac:parameter>
</ac:structured-macro></p>
</ac:rich-text-body>
</ac:structured-macro>
<h1>Title</h1>`

	opts := ConvertOptions{ShowMacros: true}
	result, err := FromConfluenceStorageWithOptions(input, opts)
	require.NoError(t, err)

	// Should have both INFO panel and nested TOC
	assert.Contains(t, result, "[INFO]")
	assert.Contains(t, result, "[/INFO]")
	assert.Contains(t, result, "[TOC maxLevel=2]")
	assert.Contains(t, result, "# Title")

	// INFO should not have TOC's parameters
	assert.NotContains(t, result, "[INFO maxLevel=2]")
}
