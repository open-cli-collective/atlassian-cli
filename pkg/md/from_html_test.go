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
