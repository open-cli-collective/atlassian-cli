package md

import (
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestFromConfluenceStorage(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result, err := FromConfluenceStorage(tt.input)
			testutil.RequireNoError(t, err)
			testutil.Equal(t, tt.expected, result)
		})
	}
}

func TestFromConfluenceStorage_ConfluenceCodeMacro(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result, err := FromConfluenceStorage(tt.input)
			testutil.RequireNoError(t, err)
			for _, expected := range tt.contains {
				testutil.Contains(t, result, expected)
			}
		})
	}
}

func TestFromConfluenceStorage_Tables(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result, err := FromConfluenceStorage(tt.input)
			testutil.RequireNoError(t, err)
			for _, expected := range tt.contains {
				testutil.Contains(t, result, expected)
			}
		})
	}
}

func TestFromConfluenceStorage_NonCodeMacrosStripped(t *testing.T) {
	t.Parallel()
	// Non-code macros should still be stripped
	input := `<p>Before</p>
	<ac:structured-macro ac:name="toc" ac:schema-version="1">
		<ac:parameter ac:name="maxLevel">3</ac:parameter>
	</ac:structured-macro>
	<p>After</p>`

	result, err := FromConfluenceStorage(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "Before")
	testutil.Contains(t, result, "After")
	testutil.NotContains(t, result, "toc")
	testutil.NotContains(t, result, "maxLevel")
}

func TestFromConfluenceStorage_TOCWithShowMacros(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			opts := ConvertOptions{ShowMacros: true}
			result, err := FromConfluenceStorageWithOptions(tt.input, opts)
			testutil.RequireNoError(t, err)
			testutil.Contains(t, result, tt.expected)
		})
	}
}

func TestFromConfluenceStorage_PanelMacrosWithShowMacros(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			opts := ConvertOptions{ShowMacros: true}
			result, err := FromConfluenceStorageWithOptions(tt.input, opts)
			testutil.RequireNoError(t, err)
			for _, expected := range tt.contains {
				testutil.Contains(t, result, expected)
			}
		})
	}
}

func TestFromConfluenceStorage_ComplexDocument(t *testing.T) {
	t.Parallel()
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
	testutil.RequireNoError(t, err)

	// Verify key elements are present
	testutil.Contains(t, result, "# Project README")
	testutil.Contains(t, result, "**introduction**")
	testutil.Contains(t, result, "## Features")
	testutil.Contains(t, result, "- Feature one")
	testutil.Contains(t, result, "## Code Example")
	testutil.Contains(t, result, "```")
	testutil.Contains(t, result, "fmt.Println")
	testutil.Contains(t, result, "[the docs](https://example.com)")
}

func TestFromConfluenceStorage_NestedMacros(t *testing.T) {
	t.Parallel()
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
	testutil.RequireNoError(t, err)

	// Should have both INFO panel and nested TOC
	testutil.Contains(t, result, "[INFO]")
	testutil.Contains(t, result, "[/INFO]")
	testutil.Contains(t, result, "[TOC maxLevel=2]")
	testutil.Contains(t, result, "# Title")

	// INFO should not have TOC's parameters
	testutil.NotContains(t, result, "[INFO maxLevel=2]")
}

// TestXHTMLToMD_NestedMacroPositionPreserved verifies that when converting XHTML back to
// Markdown, nested macros appear at their original position in the body content (not
// appended to the end).
func TestXHTMLToMD_NestedMacroPositionPreserved(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       string
		verifyOrder []string // These strings should appear in this order
		description string
	}{
		{
			name: "TOC between text in INFO",
			input: `<ac:structured-macro ac:name="info" ac:schema-version="1">
<ac:rich-text-body>
<p>Before</p>
<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>
<p>After</p>
</ac:rich-text-body>
</ac:structured-macro>`,
			verifyOrder: []string{"Before", "[TOC]", "After"},
			description: "TOC should be between Before and After",
		},
		{
			name: "multiple nested macros maintain order",
			input: `<ac:structured-macro ac:name="info" ac:schema-version="1">
<ac:rich-text-body>
<p>Start</p>
<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>
<p>Middle</p>
<ac:structured-macro ac:name="toc" ac:schema-version="1">
<ac:parameter ac:name="maxLevel">2</ac:parameter>
</ac:structured-macro>
<p>End</p>
</ac:rich-text-body>
</ac:structured-macro>`,
			verifyOrder: []string{"Start", "[TOC]", "Middle", "[TOC maxLevel=2]", "End"},
			description: "Multiple TOCs should maintain their positions",
		},
		{
			name: "deeply nested macros",
			input: `<ac:structured-macro ac:name="info" ac:schema-version="1">
<ac:rich-text-body>
<p>Outer start</p>
<ac:structured-macro ac:name="warning" ac:schema-version="1">
<ac:rich-text-body>
<p>Inner start</p>
<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>
<p>Inner end</p>
</ac:rich-text-body>
</ac:structured-macro>
<p>Outer end</p>
</ac:rich-text-body>
</ac:structured-macro>`,
			verifyOrder: []string{"Outer start", "[WARNING]", "Inner start", "[TOC]", "Inner end", "[/WARNING]", "Outer end"},
			description: "Deeply nested macros should maintain hierarchy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := ConvertOptions{ShowMacros: true}
			result, err := FromConfluenceStorageWithOptions(tt.input, opts)
			testutil.RequireNoError(t, err)

			// Verify all expected strings are present
			for _, expected := range tt.verifyOrder {
				testutil.Contains(t, result, expected)
			}

			// Verify order: each string should come after the previous one
			lastIdx := -1
			for _, expected := range tt.verifyOrder {
				idx := findStringIndex(result, expected)
				if idx == -1 {
					t.Errorf("string %q not found in result", expected)
					continue
				}
				if idx <= lastIdx {
					t.Errorf("order violation: %q (at %d) should come after position %d. %s", expected, idx, lastIdx, tt.description)
				}
				lastIdx = idx
			}

			// No placeholders should remain
			testutil.NotContains(t, result, "CFXMLCHILD")
			testutil.NotContains(t, result, "CFMACROOPEN")
			testutil.NotContains(t, result, "CFMACROCLOSE")
		})
	}
}

// Helper function to find the first occurrence of a substring
func findStringIndex(s, substr string) int {
	return strings.Index(s, substr)
}

// TestXHTMLToMD_NestedMacroOrderPreserved verifies exact ordering of content and nested
// macros using index comparisons.
func TestXHTMLToMD_NestedMacroOrderPreserved(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1">
<ac:rich-text-body>
<p>Before</p>
<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>
<p>After</p>
</ac:rich-text-body>
</ac:structured-macro>`

	opts := ConvertOptions{ShowMacros: true}
	result, err := FromConfluenceStorageWithOptions(input, opts)
	testutil.RequireNoError(t, err)

	beforeIdx := strings.Index(result, "Before")
	tocIdx := strings.Index(result, "[TOC]")
	afterIdx := strings.Index(result, "After")

	testutil.True(t, beforeIdx >= 0, "Before should be found")
	testutil.True(t, tocIdx >= 0, "[TOC] should be found")
	testutil.True(t, afterIdx >= 0, "After should be found")

	testutil.True(t, beforeIdx < tocIdx, "Before should come before [TOC]")
	testutil.True(t, tocIdx < afterIdx, "[TOC] should come before After")
}

// TestFromConfluenceStorage_NestedMacroInParagraph tests the exact bug scenario from issue #56.
// When a self-closing nested macro is wrapped in a <p> tag, the parser should correctly
// identify both macros and their nesting relationship.
func TestFromConfluenceStorage_NestedMacroInParagraph(t *testing.T) {
	t.Parallel()
	// This is the exact XHTML structure from issue #56
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1">
<ac:rich-text-body>
<p><ac:structured-macro ac:name="toc" ac:schema-version="1" /></p>
</ac:rich-text-body>
</ac:structured-macro>
<h1>Header 1</h1>`

	opts := ConvertOptions{ShowMacros: true}
	result, err := FromConfluenceStorageWithOptions(input, opts)
	testutil.RequireNoError(t, err)

	// Should contain both INFO and TOC macros without "unclosed macro" warning
	testutil.Contains(t, result, "[INFO]")
	testutil.Contains(t, result, "[TOC]")
	testutil.Contains(t, result, "[/INFO]")
	testutil.Contains(t, result, "# Header 1")

	// TOC should be inside INFO (between [INFO] and [/INFO])
	infoStart := strings.Index(result, "[INFO]")
	tocPos := strings.Index(result, "[TOC]")
	infoEnd := strings.Index(result, "[/INFO]")

	testutil.True(t, infoStart >= 0, "[INFO] should be found")
	testutil.True(t, tocPos >= 0, "[TOC] should be found")
	testutil.True(t, infoEnd >= 0, "[/INFO] should be found")

	testutil.True(t, infoStart < tocPos, "[INFO] should come before [TOC]")
	testutil.True(t, tocPos < infoEnd, "[TOC] should come before [/INFO]")
}

// TestFromConfluenceStorage_MultipleSelfClosingNestedMacros tests multiple self-closing
// macros nested inside a body macro.
func TestFromConfluenceStorage_MultipleSelfClosingNestedMacros(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1">
<ac:rich-text-body>
<p>First paragraph</p>
<p><ac:structured-macro ac:name="toc" ac:schema-version="1" /></p>
<p>Middle text</p>
<p><ac:structured-macro ac:name="anchor" ac:schema-version="1"><ac:parameter ac:name="">bookmark</ac:parameter></ac:structured-macro></p>
<p>Last paragraph</p>
</ac:rich-text-body>
</ac:structured-macro>`

	opts := ConvertOptions{ShowMacros: true}
	result, err := FromConfluenceStorageWithOptions(input, opts)
	testutil.RequireNoError(t, err)

	// All macros should be present
	testutil.Contains(t, result, "[INFO]")
	testutil.Contains(t, result, "[TOC]")
	testutil.Contains(t, result, "[/INFO]")

	// Content should be preserved
	testutil.Contains(t, result, "First paragraph")
	testutil.Contains(t, result, "Middle text")
	testutil.Contains(t, result, "Last paragraph")
}
