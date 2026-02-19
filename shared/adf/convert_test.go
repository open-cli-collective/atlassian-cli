package adf

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestToJSON_Paragraph(t *testing.T) {
	t.Parallel()
	input := "Hello world"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, doc.Type, "doc")
	testutil.Equal(t, doc.Version, 1)
	testutil.RequireEqual(t, len(doc.Content), 1)

	para := doc.Content[0]
	testutil.Equal(t, para.Type, "paragraph")
	testutil.RequireEqual(t, len(para.Content), 1)
	testutil.Equal(t, para.Content[0].Type, "text")
	testutil.Equal(t, para.Content[0].Text, "Hello world")
}

func TestToJSON_Headings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		markdown string
		level    int
		text     string
	}{
		{"h1", "# Heading 1", 1, "Heading 1"},
		{"h2", "## Heading 2", 2, "Heading 2"},
		{"h3", "### Heading 3", 3, "Heading 3"},
		{"h4", "#### Heading 4", 4, "Heading 4"},
		{"h5", "##### Heading 5", 5, "Heading 5"},
		{"h6", "###### Heading 6", 6, "Heading 6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToJSON([]byte(tt.markdown))
			testutil.RequireNoError(t, err)

			var doc Document
			err = json.Unmarshal([]byte(result), &doc)
			testutil.RequireNoError(t, err)

			testutil.RequireEqual(t, len(doc.Content), 1)
			heading := doc.Content[0]
			testutil.Equal(t, heading.Type, "heading")
			testutil.Equal(t, heading.Attrs["level"], float64(tt.level))
			testutil.RequireEqual(t, len(heading.Content), 1)
			testutil.Equal(t, heading.Content[0].Text, tt.text)
		})
	}
}

func TestToJSON_Formatting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		markdown string
		mark     string
	}{
		{"bold", "**bold**", "strong"},
		{"italic", "*italic*", "em"},
		{"inline_code", "`code`", "code"},
		{"strikethrough", "~~strike~~", "strike"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ToJSON([]byte(tt.markdown))
			testutil.RequireNoError(t, err)
			testutil.RequireNoError(t, err)

			var doc Document
			err = json.Unmarshal([]byte(result), &doc)
			testutil.RequireNoError(t, err)

			testutil.RequireEqual(t, len(doc.Content), 1)
			para := doc.Content[0]
			testutil.Equal(t, para.Type, "paragraph")

			var foundMark bool
			for _, node := range para.Content {
				if len(node.Marks) > 0 {
					for _, mark := range node.Marks {
						if mark.Type == tt.mark {
							foundMark = true
							break
						}
					}
				}
			}
			testutil.True(t, foundMark, fmt.Sprintf("expected to find mark %s", tt.mark))
		})
	}
}

func TestToJSON_Links(t *testing.T) {
	t.Parallel()
	input := "[Example](https://example.com)"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	para := doc.Content[0]

	var foundLink bool
	for _, node := range para.Content {
		for _, mark := range node.Marks {
			if mark.Type == "link" {
				foundLink = true
				testutil.Equal(t, mark.Attrs["href"], "https://example.com")
				testutil.Equal(t, node.Text, "Example")
			}
		}
	}
	testutil.True(t, foundLink, "expected to find link mark")
}

func TestToJSON_BulletList(t *testing.T) {
	t.Parallel()
	input := "- Item one\n- Item two\n- Item three"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	list := doc.Content[0]
	testutil.Equal(t, list.Type, "bulletList")
	testutil.Len(t, list.Content, 3)

	for i, item := range list.Content {
		testutil.Equal(t, item.Type, "listItem")
		testutil.RequireEqual(t, len(item.Content), 1)
		para := item.Content[0]
		testutil.Equal(t, para.Type, "paragraph")
		expected := []string{"Item one", "Item two", "Item three"}[i]
		testutil.RequireEqual(t, len(para.Content), 1)
		testutil.Equal(t, para.Content[0].Text, expected)
	}
}

func TestToJSON_OrderedList(t *testing.T) {
	t.Parallel()
	input := "1. First\n2. Second\n3. Third"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	list := doc.Content[0]
	testutil.Equal(t, list.Type, "orderedList")
	testutil.Equal(t, list.Attrs["order"], float64(1))
	testutil.Len(t, list.Content, 3)
}

func TestToJSON_CodeBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		markdown string
		language string
		code     string
	}{
		{
			name:     "without_language",
			markdown: "```\ncode here\n```",
			language: "",
			code:     "code here",
		},
		{
			name:     "with_language",
			markdown: "```python\nprint(\"hello\")\n```",
			language: "python",
			code:     "print(\"hello\")",
		},
		{
			name:     "go_multiline",
			markdown: "```go\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```",
			language: "go",
			code:     "func main() {\n    fmt.Println(\"hello\")\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ToJSON([]byte(tt.markdown))
			testutil.RequireNoError(t, err)

			var doc Document
			err = json.Unmarshal([]byte(result), &doc)
			testutil.RequireNoError(t, err)

			testutil.RequireEqual(t, len(doc.Content), 1)
			block := doc.Content[0]
			testutil.Equal(t, block.Type, "codeBlock")

			if tt.language != "" {
				testutil.Equal(t, block.Attrs["language"], tt.language)
			}

			testutil.RequireEqual(t, len(block.Content), 1)
			testutil.Equal(t, block.Content[0].Text, tt.code)
		})
	}
}

func TestToJSON_Blockquote(t *testing.T) {
	t.Parallel()
	input := "> This is a quote"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	quote := doc.Content[0]
	testutil.Equal(t, quote.Type, "blockquote")
	testutil.RequireEqual(t, len(quote.Content), 1)
	testutil.Equal(t, quote.Content[0].Type, "paragraph")
}

func TestToJSON_HorizontalRule(t *testing.T) {
	t.Parallel()
	input := "Above\n\n---\n\nBelow"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 3)
	testutil.Equal(t, doc.Content[0].Type, "paragraph")
	testutil.Equal(t, doc.Content[1].Type, "rule")
	testutil.Equal(t, doc.Content[2].Type, "paragraph")
}

func TestToJSON_Table(t *testing.T) {
	t.Parallel()
	input := "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	table := doc.Content[0]
	testutil.Equal(t, table.Type, "table")
	testutil.Len(t, table.Content, 2)

	headerRow := table.Content[0]
	testutil.Equal(t, headerRow.Type, "tableRow")
	testutil.Len(t, headerRow.Content, 2)
	testutil.Equal(t, headerRow.Content[0].Type, "tableHeader")

	dataRow := table.Content[1]
	testutil.Equal(t, dataRow.Type, "tableRow")
	testutil.Len(t, dataRow.Content, 2)
	testutil.Equal(t, dataRow.Content[0].Type, "tableCell")
}

func TestToJSON_EmptyInput(t *testing.T) {
	t.Parallel()
	result, err := ToJSON([]byte(""))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, doc.Type, "doc")
	testutil.Equal(t, doc.Version, 1)
	testutil.Empty(t, doc.Content)
}

func TestToJSON_NestedList(t *testing.T) {
	t.Parallel()
	input := "- Item one\n  - Nested one\n  - Nested two\n- Item two"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	list := doc.Content[0]
	testutil.Equal(t, list.Type, "bulletList")

	firstItem := list.Content[0]
	testutil.Equal(t, firstItem.Type, "listItem")

	var foundNestedList bool
	for _, child := range firstItem.Content {
		if child.Type == "bulletList" {
			foundNestedList = true
			testutil.Len(t, child.Content, 2)
		}
	}
	testutil.True(t, foundNestedList, "expected nested bullet list")
}

func TestToJSON_BoldAndItalicCombined(t *testing.T) {
	t.Parallel()
	input := "***bold and italic***"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	para := doc.Content[0]

	var foundStrong, foundEm bool
	for _, node := range para.Content {
		for _, mark := range node.Marks {
			if mark.Type == "strong" {
				foundStrong = true
			}
			if mark.Type == "em" {
				foundEm = true
			}
		}
	}
	testutil.True(t, foundStrong, "expected strong mark")
	testutil.True(t, foundEm, "expected em mark")
}

func TestToJSON_OutputIsValidJSON(t *testing.T) {
	t.Parallel()
	inputs := []string{
		"# Simple heading",
		"Paragraph with **bold** and *italic*",
		"- Item 1\n- Item 2",
		"```go\ncode\n```",
		"| A | B |\n|---|---|\n| 1 | 2 |",
	}

	for _, input := range inputs {
		result, err := ToJSON([]byte(input))
		testutil.RequireNoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(result), &parsed)
		testutil.RequireNoError(t, err)

		testutil.Equal(t, parsed["type"], "doc")
		testutil.Equal(t, parsed["version"], float64(1))
	}
}

func TestToJSON_Images_AltText(t *testing.T) {
	t.Parallel()
	input := "![Alt text](https://example.com/image.png)"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	para := doc.Content[0]
	testutil.Equal(t, para.Type, "paragraph")
	testutil.RequireEqual(t, len(para.Content), 1)
	testutil.Equal(t, para.Content[0].Text, "Alt text")
}

func TestToJSON_WhitespaceInCodeBlock(t *testing.T) {
	t.Parallel()
	input := "```\n    indented code\n        more indented\n```"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	block := doc.Content[0]
	testutil.Equal(t, block.Type, "codeBlock")
	testutil.RequireEqual(t, len(block.Content), 1)

	text := block.Content[0].Text
	testutil.Contains(t, text, "    indented")
	testutil.Contains(t, text, "        more indented")
}

func TestToJSON_NestedBlockquote(t *testing.T) {
	t.Parallel()
	input := "> Quote with **bold** text\n>\n> And a list:\n> - Item 1\n> - Item 2"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	quote := doc.Content[0]
	testutil.Equal(t, quote.Type, "blockquote")
	testutil.True(t, len(quote.Content) > 0, "blockquote should have content")
}

func TestToJSON_HardLineBreak(t *testing.T) {
	t.Parallel()
	input := "Line one  \nLine two"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	para := doc.Content[0]
	testutil.Equal(t, para.Type, "paragraph")

	var foundBreak bool
	for _, node := range para.Content {
		if node.Type == "hardBreak" {
			foundBreak = true
			break
		}
	}
	testutil.True(t, foundBreak, "expected hardBreak node")
}

func TestToJSON_InlineCodePreservesContent(t *testing.T) {
	t.Parallel()
	input := "Use `fmt.Println()` to print"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	para := doc.Content[0]

	var foundCode bool
	for _, node := range para.Content {
		for _, mark := range node.Marks {
			if mark.Type == "code" {
				foundCode = true
				testutil.Equal(t, node.Text, "fmt.Println()")
			}
		}
	}
	testutil.True(t, foundCode, "expected code mark")
}

func TestToDocument_Empty(t *testing.T) {
	t.Parallel()
	doc := ToDocument("")
	testutil.Nil(t, doc)
}

func TestToDocument_PlainText(t *testing.T) {
	t.Parallel()
	doc := ToDocument("Hello world")
	testutil.NotNil(t, doc)
	testutil.Equal(t, doc.Type, "doc")
	testutil.Equal(t, doc.Version, 1)
	testutil.RequireEqual(t, len(doc.Content), 1)
	testutil.Equal(t, doc.Content[0].Type, "paragraph")
	testutil.RequireEqual(t, len(doc.Content[0].Content), 1)
	testutil.Equal(t, doc.Content[0].Content[0].Text, "Hello world")
}

func TestToDocument_ToPlainText(t *testing.T) {
	t.Parallel()
	doc := ToDocument("# Title\n\nSome text\n\n- Item 1\n- Item 2")
	testutil.NotNil(t, doc)

	text := doc.ToPlainText()
	testutil.Contains(t, text, "Title")
	testutil.Contains(t, text, "Some text")
	testutil.Contains(t, text, "Item 1")
	testutil.Contains(t, text, "Item 2")
}

func TestToPlainText_Nil(t *testing.T) {
	t.Parallel()
	var doc *Document
	testutil.Equal(t, doc.ToPlainText(), "")
}

func TestToJSON_IndentedCodeBlock(t *testing.T) {
	t.Parallel()
	input := "    code line one\n    code line two"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	block := doc.Content[0]
	testutil.Equal(t, block.Type, "codeBlock")
	testutil.Nil(t, block.Attrs)
	testutil.RequireEqual(t, len(block.Content), 1)
	testutil.Contains(t, block.Content[0].Text, "code line one")
	testutil.Contains(t, block.Content[0].Text, "code line two")
}

func TestToJSON_AutoLink(t *testing.T) {
	t.Parallel()
	input := "Visit <https://example.com> for info"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	para := doc.Content[0]

	var foundAutoLink bool
	for _, node := range para.Content {
		for _, mark := range node.Marks {
			if mark.Type == "link" {
				foundAutoLink = true
				testutil.Equal(t, mark.Attrs["href"], "https://example.com")
				testutil.Equal(t, node.Text, "https://example.com")
			}
		}
	}
	testutil.True(t, foundAutoLink, "expected to find auto-linked URL")
}

func TestToJSON_RawHTMLDropped(t *testing.T) {
	t.Parallel()
	input := "Before <span>raw</span> after"
	result, err := ToJSON([]byte(input))
	testutil.RequireNoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.RequireEqual(t, len(doc.Content), 1)
	para := doc.Content[0]

	// Raw HTML should be dropped; surrounding text should remain
	var allText string
	for _, node := range para.Content {
		allText += node.Text
	}
	testutil.Contains(t, allText, "Before")
	testutil.Contains(t, allText, "after")
	testutil.NotContains(t, allText, "<span>")
}

func TestSplitLines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"trailing newline stripped", "a\nb\n", []string{"a", "b"}},
		{"single line", "single", []string{"single"}},
		{"empty string", "", []string{}},
		{"multiple trailing newlines", "a\n\n\n", []string{"a"}},
		{"no trailing newline", "a\nb", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := splitLines(tt.input)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func TestToPlainText_CodeBlock(t *testing.T) {
	t.Parallel()
	doc := &Document{
		Type:    "doc",
		Version: 1,
		Content: []*Node{
			{
				Type: "codeBlock",
				Content: []*Node{
					{Type: "text", Text: "fmt.Println()"},
				},
			},
		},
	}

	text := doc.ToPlainText()
	testutil.Contains(t, text, "fmt.Println()")
}

func TestToPlainText_Blockquote(t *testing.T) {
	t.Parallel()
	doc := &Document{
		Type:    "doc",
		Version: 1,
		Content: []*Node{
			{
				Type: "blockquote",
				Content: []*Node{
					{
						Type: "paragraph",
						Content: []*Node{
							{Type: "text", Text: "Quoted"},
						},
					},
				},
			},
		},
	}

	text := doc.ToPlainText()
	testutil.Contains(t, text, "> Quoted")
}

func TestToPlainText_Rule(t *testing.T) {
	t.Parallel()
	doc := &Document{
		Type:    "doc",
		Version: 1,
		Content: []*Node{
			{Type: "rule"},
		},
	}

	text := doc.ToPlainText()
	testutil.Contains(t, text, "---")
}

func TestToPlainText_UnknownNodeType(t *testing.T) {
	t.Parallel()
	doc := &Document{
		Type:    "doc",
		Version: 1,
		Content: []*Node{
			{Type: "unknownWidget", Text: "fallback text"},
		},
	}

	text := doc.ToPlainText()
	testutil.Contains(t, text, "fallback text")
}

func TestToPlainText_UnknownNodeWithChildren(t *testing.T) {
	t.Parallel()
	doc := &Document{
		Type:    "doc",
		Version: 1,
		Content: []*Node{
			{
				Type: "panel",
				Content: []*Node{
					{
						Type: "paragraph",
						Content: []*Node{
							{Type: "text", Text: "inner"},
						},
					},
				},
			},
		},
	}

	text := doc.ToPlainText()
	testutil.Contains(t, text, "inner")
}
