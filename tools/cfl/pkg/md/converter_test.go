package md

import (
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestToConfluenceStorage(t *testing.T) {
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
			t.Parallel()
			result, err := ToConfluenceStorage([]byte(tt.input))
			testutil.RequireNoError(t, err)
			testutil.Equal(t, tt.expected, result)
		})
	}
}

func TestToConfluenceStorage_ComplexDocument(t *testing.T) {
	t.Parallel()
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
	testutil.RequireNoError(t, err)

	// Verify key elements are present
	testutil.Contains(t, result, "<h1>Project README</h1>")
	testutil.Contains(t, result, "<strong>introduction</strong>")
	testutil.Contains(t, result, "<h2>Features</h2>")
	testutil.Contains(t, result, "<li>Feature one</li>")
	testutil.Contains(t, result, "<h2>Code Example</h2>")
	testutil.Contains(t, result, "language-go")
	testutil.Contains(t, result, "fmt.Println")
	testutil.Contains(t, result, `<a href="https://example.com">the docs</a>`)
}

func TestToConfluenceStorage_TOCMacro(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result, err := ToConfluenceStorage([]byte(tt.input))
			testutil.RequireNoError(t, err)
			for _, expected := range tt.contains {
				testutil.Contains(t, result, expected)
			}
		})
	}
}

func TestToConfluenceStorage_TOCMixedWithContent(t *testing.T) {
	t.Parallel()
	input := `[TOC maxLevel=3]

# Heading 1

Some content here.

## Heading 2

More content.
`
	result, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)

	// Verify TOC macro is present
	testutil.Contains(t, result, `<ac:structured-macro ac:name="toc" ac:schema-version="1">`)
	testutil.Contains(t, result, `<ac:parameter ac:name="maxLevel">3</ac:parameter>`)
	testutil.Contains(t, result, `</ac:structured-macro>`)

	// Verify other content is preserved
	testutil.Contains(t, result, "<h1>Heading 1</h1>")
	testutil.Contains(t, result, "Some content here.")
	testutil.Contains(t, result, "<h2>Heading 2</h2>")
}

func TestToConfluenceStorage_TOCRoundtrip(t *testing.T) {
	t.Parallel()
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
	testutil.RequireNoError(t, err)

	// Verify markdown has TOC placeholder with params
	testutil.Contains(t, markdown, "[TOC")
	testutil.Contains(t, markdown, "maxLevel=3")
	testutil.Contains(t, markdown, "minLevel=1")

	// Convert back to storage format
	resultXHTML, err := ToConfluenceStorage([]byte(markdown))
	testutil.RequireNoError(t, err)

	// Verify TOC macro is restored
	testutil.Contains(t, resultXHTML, `<ac:structured-macro ac:name="toc" ac:schema-version="1">`)
	testutil.Contains(t, resultXHTML, `<ac:parameter ac:name="maxLevel">3</ac:parameter>`)
	testutil.Contains(t, resultXHTML, `<ac:parameter ac:name="minLevel">1</ac:parameter>`)
}

func TestParseKeyValueParams(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := parseKeyValueParams(tt.input)
			testutil.Equal(t, tt.expected, result)
		})
	}
}

func TestToConfluenceStorage_PanelMacros(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result, err := ToConfluenceStorage([]byte(tt.input))
			testutil.RequireNoError(t, err)
			for _, expected := range tt.contains {
				testutil.Contains(t, result, expected)
			}
		})
	}
}

func TestToConfluenceStorage_PanelMixedWithContent(t *testing.T) {
	t.Parallel()
	input := `# Heading

Some intro text.

[WARNING title="Important"]
This is a warning.
[/WARNING]

More text after.
`
	result, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)

	// Verify all parts are present
	testutil.Contains(t, result, "<h1>Heading</h1>")
	testutil.Contains(t, result, "Some intro text.")
	testutil.Contains(t, result, `<ac:structured-macro ac:name="warning" ac:schema-version="1">`)
	testutil.Contains(t, result, `<ac:parameter ac:name="title">Important</ac:parameter>`)
	testutil.Contains(t, result, "This is a warning.")
	testutil.Contains(t, result, "</ac:structured-macro>")
	testutil.Contains(t, result, "More text after.")
}

func TestToConfluenceStorage_PanelRoundtrip(t *testing.T) {
	t.Parallel()
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
	testutil.RequireNoError(t, err)

	// Verify markdown has panel placeholder (brackets may be escaped by markdown converter)
	testutil.Contains(t, markdown, "INFO")
	testutil.Contains(t, markdown, "title=Important")
	testutil.Contains(t, markdown, "Panel content")

	// Convert back to storage format
	resultXHTML, err := ToConfluenceStorage([]byte(markdown))
	testutil.RequireNoError(t, err)

	// Verify panel macro is restored
	testutil.Contains(t, resultXHTML, `<ac:structured-macro ac:name="info" ac:schema-version="1">`)
	testutil.Contains(t, resultXHTML, `<ac:parameter ac:name="title">Important</ac:parameter>`)
	testutil.Contains(t, resultXHTML, `<ac:rich-text-body>`)
	testutil.Contains(t, resultXHTML, `Panel content`)
	testutil.Contains(t, resultXHTML, `</ac:rich-text-body>`)
}

func TestToConfluenceStorage_NestedMacros(t *testing.T) {
	t.Parallel()
	// Test nested TOC inside INFO panel
	input := `[INFO]
Check out the table of contents: [TOC]
[/INFO]`

	result, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)

	// The result should have both the panel macro and TOC macro
	testutil.Contains(t, result, `<ac:structured-macro ac:name="info"`)
	testutil.Contains(t, result, `<ac:structured-macro ac:name="toc"`)

	// Make sure placeholders are not left behind
	testutil.NotContains(t, result, "CFMACRO")
	testutil.NotContains(t, result, "END")
}

// TestPanelMacro_CloseTagConsumed verifies that panel macro close tags like [/INFO]
// are properly consumed during MD→XHTML conversion and don't appear as literal text.
func TestPanelMacro_CloseTagConsumed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple INFO panel",
			input: "[INFO]content[/INFO]",
		},
		{
			name:  "INFO with newlines",
			input: "[INFO]\ncontent\n[/INFO]",
		},
		{
			name:  "WARNING panel",
			input: "[WARNING]warning content[/WARNING]",
		},
		{
			name:  "NOTE panel",
			input: "[NOTE]note content[/NOTE]",
		},
		{
			name:  "mixed case close tag",
			input: "[INFO]content[/info]",
		},
		{
			name:  "lowercase open and close",
			input: "[info]content[/info]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ToConfluenceStorage([]byte(tt.input))
			testutil.RequireNoError(t, err)

			// Close tag should NOT appear as literal text
			testutil.NotContains(t, result, "[/INFO]")
			testutil.NotContains(t, result, "[/info]")
			testutil.NotContains(t, result, "[/WARNING]")
			testutil.NotContains(t, result, "[/NOTE]")

			// Content should appear exactly once
			testutil.Equal(t, 1, strings.Count(result, "content"))
		})
	}
}

// TestNestedMacros_ProcessedAtCorrectLevel verifies that nested macros like [TOC] inside
// [INFO]...[/INFO] are converted to XML at the correct nesting level (inside the parent's
// rich-text-body), not as siblings at the top level.
func TestNestedMacros_ProcessedAtCorrectLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		input             string
		verifyContains    []string
		verifyNotContains []string
	}{
		{
			name:  "TOC inside INFO",
			input: "[INFO][TOC][/INFO]",
			verifyContains: []string{
				`ac:name="info"`,
				`ac:name="toc"`,
			},
			verifyNotContains: []string{
				"[TOC]", // should not be literal text
				"[/INFO]",
			},
		},
		{
			name:  "TOC with text inside INFO",
			input: "[INFO]Before [TOC] After[/INFO]",
			verifyContains: []string{
				`ac:name="info"`,
				`ac:name="toc"`,
				"Before",
				"After",
			},
			verifyNotContains: []string{
				"[TOC]",
				"[/INFO]",
			},
		},
		{
			name:  "multiple nested macros",
			input: "[INFO]Start [TOC] middle [TOC] end[/INFO]",
			verifyContains: []string{
				`ac:name="info"`,
				"Start",
				"middle",
				"end",
			},
		},
		{
			name:  "deeply nested - TOC inside WARNING inside INFO",
			input: "[INFO]Outer [WARNING]Inner [TOC][/WARNING][/INFO]",
			verifyContains: []string{
				`ac:name="info"`,
				`ac:name="warning"`,
				`ac:name="toc"`,
				"Outer",
				"Inner",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ToConfluenceStorage([]byte(tt.input))
			testutil.RequireNoError(t, err)

			for _, expected := range tt.verifyContains {
				testutil.Contains(t, result, expected)
			}
			for _, notExpected := range tt.verifyNotContains {
				testutil.NotContains(t, result, notExpected)
			}

			// No placeholders should remain
			testutil.NotContains(t, result, "CFMACRO")
			testutil.NotContains(t, result, "CFCHILD")
		})
	}
}

// TestNestedMacros_XMLStructureCorrect verifies that the generated XML has correct
// structure: nested macros appear inside the parent's rich-text-body element.
func TestNestedMacros_XMLStructureCorrect(t *testing.T) {
	t.Parallel()
	input := "[INFO]Before [TOC] After[/INFO]"
	result, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)

	// Verify TOC is inside INFO's rich-text-body
	infoStart := strings.Index(result, `ac:name="info"`)
	richTextStart := strings.Index(result, `<ac:rich-text-body>`)
	tocStart := strings.Index(result, `ac:name="toc"`)
	richTextEnd := strings.Index(result, `</ac:rich-text-body>`)
	// Use LastIndex for INFO's closing tag since TOC also uses </ac:structured-macro>
	infoEnd := strings.LastIndex(result, `</ac:structured-macro>`)

	testutil.True(t, infoStart < richTextStart, "INFO should start before rich-text-body")
	testutil.True(t, richTextStart < tocStart, "rich-text-body should start before TOC")
	testutil.True(t, tocStart < richTextEnd, "TOC should be before rich-text-body end")
	testutil.True(t, richTextEnd < infoEnd, "rich-text-body should end before INFO")
}

// TestToConfluenceStorage_MacrosInCodeBlock verifies that bracket macros inside fenced
// code blocks are preserved as literal text and not expanded to Confluence XML.
func TestToConfluenceStorage_MacrosInCodeBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       string
		contains    []string
		notContains []string
	}{
		{
			name:     "TOC in fenced code block",
			input:    "```\n[TOC]\n```",
			contains: []string{"<code>[TOC]\n</code>"},
			notContains: []string{
				`ac:name="toc"`,
				"CFMACRO",
			},
		},
		{
			name:     "INFO panel in fenced code block",
			input:    "```\n[INFO]\nSome content\n[/INFO]\n```",
			contains: []string{"[INFO]", "[/INFO]"},
			notContains: []string{
				`ac:name="info"`,
			},
		},
		{
			name:  "macro outside code still expanded",
			input: "```\n[TOC]\n```\n\n[TOC]",
			contains: []string{
				"<code>[TOC]\n</code>",
				`ac:name="toc"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ToConfluenceStorage([]byte(tt.input))
			testutil.RequireNoError(t, err)
			for _, s := range tt.contains {
				testutil.Contains(t, result, s)
			}
			for _, s := range tt.notContains {
				testutil.NotContains(t, result, s)
			}
		})
	}
}

// TestToConfluenceStorage_DeterministicNestedMacros verifies that deeply nested macro
// conversion is deterministic. This catches issues with non-deterministic map iteration
// order in Go which previously caused flaky test failures. See issue #68.
func TestToConfluenceStorage_DeterministicNestedMacros(t *testing.T) {
	t.Parallel()
	input := `[INFO]Outer
[WARNING]Inner
[TOC]
More inner
[/WARNING]
More outer
[/INFO]`

	// Run 100 times to catch non-deterministic behavior
	for i := 0; i < 100; i++ {
		result, err := ToConfluenceStorage([]byte(input))
		testutil.RequireNoError(t, err)
		testutil.Contains(t, result, `ac:name="toc"`)
		testutil.Contains(t, result, `ac:name="warning"`)
		testutil.Contains(t, result, `ac:name="info"`)
		testutil.NotContains(t, result, "CFMACRO")
	}
}

// TestToConfluenceStorage_FailingBracketLinkDoesNotDuplicateOrLeakPlaceholders
// is a regression test for a tokenizer bug where markdown links whose link
// text contained characters outside [A-Za-z0-9_-] caused the text between
// the failing '[' and the next valid bracket to be emitted twice. When the
// duplicated chunk contained inline code, the second copy's CFCODE…ENDC
// placeholder was never restored (restoreCodeRegions used single-match
// Replace), so it leaked into the final HTML as literal text.
//
// This test covers the end-to-end path a user of the CLI hits: a markdown
// document with a "bad" link text followed by content that contains inline
// code. The output must contain the original content exactly once and have
// no CFCODE / CFMACRO placeholder residue.
func TestToConfluenceStorage_FailingBracketLinkDoesNotDuplicateOrLeakPlaceholders(t *testing.T) {
	t.Parallel()
	input := "# Repro\n\n" +
		"Related MR: [signalft!116](https://example.com/mr/116) · Ticket: [MON-4791](https://example.com/browse/MON-4791)\n\n" +
		"Some text with `inline code` and more.\n"

	result, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)

	// No placeholder residue.
	testutil.NotContains(t, result, "CFCODE")
	testutil.NotContains(t, result, "CFMACRO")
	testutil.NotContains(t, result, "CFCHILD")

	// Exactly one rendered heading.
	testutil.Equal(t, 1, strings.Count(result, "<h1>Repro</h1>"))
	// Exactly one occurrence of the inline code.
	testutil.Equal(t, 1, strings.Count(result, "<code>inline code</code>"))
	// Exactly one "Some text with" paragraph.
	testutil.Equal(t, 1, strings.Count(result, "Some text with"))
	// The "# Repro" string must NOT appear anywhere outside the h1 — the
	// bug caused a second copy of "# Repro" to be flushed as literal text
	// inside the "Related MR:" paragraph.
	testutil.NotContains(t, result, "# Repro")
}
