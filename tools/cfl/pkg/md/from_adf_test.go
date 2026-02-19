package md

import (
	"encoding/json"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestFromADF_EmptyInput(t *testing.T) {
	t.Parallel()
	result, err := FromADF("")
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "", result)
}

func TestFromADF_EmptyDocument(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "", result)
}

func TestFromADF_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := FromADF("{invalid")
	testutil.RequireError(t, err)
	testutil.Contains(t, err.Error(), "parsing ADF JSON")
}

func TestFromADF_Paragraph(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello world"}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "Hello world\n", result)
}

func TestFromADF_Headings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		level    int
		text     string
		expected string
	}{
		{"h1", 1, "Title", "# Title\n"},
		{"h2", 2, "Subtitle", "## Subtitle\n"},
		{"h3", 3, "Section", "### Section\n"},
		{"h4", 4, "Subsection", "#### Subsection\n"},
		{"h5", 5, "Minor", "##### Minor\n"},
		{"h6", 6, "Smallest", "###### Smallest\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			input := adfDoc(adfHeading(tt.level, tt.text))
			result, err := FromADF(input)
			testutil.RequireNoError(t, err)
			testutil.Equal(t, tt.expected, result)
		})
	}
}

func TestFromADF_Bold(t *testing.T) {
	t.Parallel()
	input := adfDoc(adfPara(adfMarkedText("bold text", "strong")))
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "**bold text**\n", result)
}

func TestFromADF_Italic(t *testing.T) {
	t.Parallel()
	input := adfDoc(adfPara(adfMarkedText("italic text", "em")))
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "*italic text*\n", result)
}

func TestFromADF_InlineCode(t *testing.T) {
	t.Parallel()
	input := adfDoc(adfPara(adfMarkedText("fmt.Println()", "code")))
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "`fmt.Println()`\n", result)
}

func TestFromADF_Strikethrough(t *testing.T) {
	t.Parallel()
	input := adfDoc(adfPara(adfMarkedText("deleted", "strike")))
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "~~deleted~~\n", result)
}

func TestFromADF_Link(t *testing.T) {
	t.Parallel()
	input := adfDoc(adfPara(`{"type":"text","text":"click here","marks":[{"type":"link","attrs":{"href":"https://example.com"}}]}`))
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "[click here](https://example.com)\n", result)
}

func TestFromADF_MixedInline(t *testing.T) {
	t.Parallel()
	input := adfDoc(adfPara(
		`{"type":"text","text":"Hello "}`,
		adfMarkedText("world", "strong"),
		`{"type":"text","text":" and "}`,
		adfMarkedText("code", "code"),
	))
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "Hello **world** and `code`\n", result)
}

func TestFromADF_BulletList(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Item one"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Item two"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Item three"}]}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "- Item one\n- Item two\n- Item three\n", result)
}

func TestFromADF_OrderedList(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"orderedList","attrs":{"order":1},"content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"First"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Second"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Third"}]}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "1. First\n2. Second\n3. Third\n", result)
}

func TestFromADF_NestedList(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Outer"}]},{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Inner"}]}]}]}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "- Outer")
	testutil.Contains(t, result, "  - Inner")
}

func TestFromADF_CodeBlock_NoLanguage(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"codeBlock","content":[{"type":"text","text":"hello world"}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "```\nhello world\n```\n", result)
}

func TestFromADF_CodeBlock_WithLanguage(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"fmt.Println(\"hello\")"}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "```go\nfmt.Println(\"hello\")\n```\n", result)
}

func TestFromADF_Blockquote(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"blockquote","content":[{"type":"paragraph","content":[{"type":"text","text":"Quoted text"}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "> Quoted text\n", result)
}

func TestFromADF_HorizontalRule(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Above"}]},{"type":"rule"},{"type":"paragraph","content":[{"type":"text","text":"Below"}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "Above")
	testutil.Contains(t, result, "---")
	testutil.Contains(t, result, "Below")
}

func TestFromADF_Table(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"table","attrs":{"layout":"default"},"content":[{"type":"tableRow","content":[{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"Name"}]}]},{"type":"tableHeader","content":[{"type":"paragraph","content":[{"type":"text","text":"Value"}]}]}]},{"type":"tableRow","content":[{"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"A"}]}]},{"type":"tableCell","content":[{"type":"paragraph","content":[{"type":"text","text":"1"}]}]}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "| Name")
	testutil.Contains(t, result, "| A")
	testutil.Contains(t, result, "---")
}

func TestFromADF_HardBreak(t *testing.T) {
	t.Parallel()
	input := adfDoc(adfPara(`{"type":"text","text":"Line one"}`, `{"type":"hardBreak"}`, `{"type":"text","text":"Line two"}`))
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "Line one  \nLine two\n", result)
}

func TestFromADF_Extension_TOC(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"extension","attrs":{"extensionType":"com.atlassian.confluence.macro.core","extensionKey":"toc","layout":"default"}}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "[TOC]\n", result)
}

func TestFromADF_Extension_TOC_WithParams(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"extension","attrs":{"extensionType":"com.atlassian.confluence.macro.core","extensionKey":"toc","parameters":{"maxLevel":{"value":"3"}},"layout":"default"}}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, "[TOC maxLevel=3]\n", result)
}

func TestFromADF_Panel_Info(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"panel","attrs":{"panelType":"info"},"content":[{"type":"paragraph","content":[{"type":"text","text":"Important info"}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "[INFO]")
	testutil.Contains(t, result, "Important info")
	testutil.Contains(t, result, "[/INFO]")
}

func TestFromADF_Panel_Warning(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"panel","attrs":{"panelType":"warning"},"content":[{"type":"paragraph","content":[{"type":"text","text":"Be careful"}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "[WARNING]")
	testutil.Contains(t, result, "Be careful")
	testutil.Contains(t, result, "[/WARNING]")
}

func TestFromADF_BodiedExtension_Expand(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"bodiedExtension","attrs":{"extensionType":"com.atlassian.confluence.macro.core","extensionKey":"expand","parameters":{"title":{"value":"Click me"}},"layout":"default"},"content":[{"type":"paragraph","content":[{"type":"text","text":"Hidden content"}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "[EXPAND title=Click me]")
	testutil.Contains(t, result, "Hidden content")
	testutil.Contains(t, result, "[/EXPAND]")
}

func TestFromADF_EmptyParagraph(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"paragraph"}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	// An empty paragraph produces just a newline, which gets trimmed to empty.
	testutil.Equal(t, "", result)
}

func TestFromADF_MultipleBlocks(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Title"}]},{"type":"paragraph","content":[{"type":"text","text":"Some text"}]},{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Item"}]}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "# Title")
	testutil.Contains(t, result, "Some text")
	testutil.Contains(t, result, "- Item")
}

func TestFromADF_UnknownNodeFallback(t *testing.T) {
	t.Parallel()
	input := `{"type":"doc","version":1,"content":[{"type":"customWidget","text":"fallback text"}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "fallback text")
}

func TestFromADF_InlineCard(t *testing.T) {
	t.Parallel()
	input := adfDoc(adfPara(`{"type":"inlineCard","attrs":{"url":"https://example.com/page"}}`))
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "https://example.com/page")
}

func TestFromADF_ListItem_ContinuationParagraph(t *testing.T) {
	t.Parallel()
	// List item with two paragraphs: first gets bullet prefix, second gets indent only.
	input := `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"First para"}]},{"type":"paragraph","content":[{"type":"text","text":"Second para"}]}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "- First para")
	testutil.Contains(t, result, "  Second para")
	// Second paragraph should NOT have a bullet prefix.
	testutil.NotContains(t, result, "- Second para")
}

func TestFromADF_ListItem_NestedOrderedList(t *testing.T) {
	t.Parallel()
	// Bullet list item containing a nested ordered list.
	input := `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Outer"}]},{"type":"orderedList","attrs":{"order":1},"content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Inner"}]}]}]}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "- Outer")
	testutil.Contains(t, result, "  1. Inner")
}

func TestFromADF_ListItem_WithCodeBlock(t *testing.T) {
	t.Parallel()
	// List item containing a paragraph followed by a code block.
	input := `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Item with code"}]},{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"fmt.Println()"}]}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "- Item with code")
	testutil.Contains(t, result, "```go")
	testutil.Contains(t, result, "fmt.Println()")
}

func TestFromADF_ListItem_DefaultChild(t *testing.T) {
	t.Parallel()
	// List item containing a blockquote (falls through to the default case).
	input := `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"blockquote","content":[{"type":"paragraph","content":[{"type":"text","text":"Quoted"}]}]}]}]}]}`
	result, err := FromADF(input)
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "- > Quoted")
}

// Test helpers for building ADF JSON strings.

func adfDoc(blocks ...string) string {
	return `{"type":"doc","version":1,"content":[` + join(blocks) + `]}`
}

func adfHeading(level int, text string) string {
	return `{"type":"heading","attrs":{"level":` + itoa(level) + `},"content":[{"type":"text","text":"` + text + `"}]}`
}

func adfPara(inlineNodes ...string) string {
	return `{"type":"paragraph","content":[` + join(inlineNodes) + `]}`
}

func adfMarkedText(text, markType string) string {
	return `{"type":"text","text":"` + text + `","marks":[{"type":"` + markType + `"}]}`
}

func join(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ","
		}
		result += p
	}
	return result
}

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}
