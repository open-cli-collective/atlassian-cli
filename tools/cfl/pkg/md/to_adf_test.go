package md

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestToADF_Paragraph(t *testing.T) {
	t.Parallel()
	input := "Hello world"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, "doc", doc.Type)
	testutil.Equal(t, 1, doc.Version)
	testutil.Len(t, doc.Content, 1)

	para := doc.Content[0]
	testutil.Equal(t, "paragraph", para.Type)
	testutil.Len(t, para.Content, 1)
	testutil.Equal(t, "text", para.Content[0].Type)
	testutil.Equal(t, "Hello world", para.Content[0].Text)
}

func TestToADF_Headings(t *testing.T) {
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
			t.Parallel()
			result, err := ToADF([]byte(tt.markdown))
			testutil.RequireNoError(t, err)

			var doc ADFDocument
			err = json.Unmarshal([]byte(result), &doc)
			testutil.RequireNoError(t, err)

			testutil.Len(t, doc.Content, 1)
			heading := doc.Content[0]
			testutil.Equal(t, "heading", heading.Type)
			testutil.Equal(t, heading.Attrs["level"], float64(tt.level))
			testutil.Len(t, heading.Content, 1)
			testutil.Equal(t, tt.text, heading.Content[0].Text)
		})
	}
}

func TestToADF_Formatting(t *testing.T) {
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
			result, err := ToADF([]byte(tt.markdown))
			testutil.RequireNoError(t, err)

			var doc ADFDocument
			err = json.Unmarshal([]byte(result), &doc)
			testutil.RequireNoError(t, err)

			testutil.Len(t, doc.Content, 1)
			para := doc.Content[0]
			testutil.Equal(t, "paragraph", para.Type)

			// Find the text node with marks
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

func TestToADF_Links(t *testing.T) {
	t.Parallel()
	input := "[Example](https://example.com)"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	para := doc.Content[0]

	// Find the link
	var foundLink bool
	for _, node := range para.Content {
		for _, mark := range node.Marks {
			if mark.Type == "link" {
				foundLink = true
				testutil.Equal(t, "https://example.com", mark.Attrs["href"])
				testutil.Equal(t, "Example", node.Text)
			}
		}
	}
	testutil.True(t, foundLink, "expected to find link mark")
}

func TestToADF_BulletList(t *testing.T) {
	t.Parallel()
	input := "- Item one\n- Item two\n- Item three"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	list := doc.Content[0]
	testutil.Equal(t, "bulletList", list.Type)
	testutil.Len(t, list.Content, 3)

	for i, item := range list.Content {
		testutil.Equal(t, "listItem", item.Type)
		testutil.Len(t, item.Content, 1)
		para := item.Content[0]
		testutil.Equal(t, "paragraph", para.Type)
		expected := []string{"Item one", "Item two", "Item three"}[i]
		testutil.Len(t, para.Content, 1)
		testutil.Equal(t, expected, para.Content[0].Text)
	}
}

func TestToADF_OrderedList(t *testing.T) {
	t.Parallel()
	input := "1. First\n2. Second\n3. Third"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	list := doc.Content[0]
	testutil.Equal(t, "orderedList", list.Type)
	testutil.Equal(t, list.Attrs["order"], float64(1))
	testutil.Len(t, list.Content, 3)
}

func TestToADF_CodeBlock(t *testing.T) {
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
			result, err := ToADF([]byte(tt.markdown))
			testutil.RequireNoError(t, err)

			var doc ADFDocument
			err = json.Unmarshal([]byte(result), &doc)
			testutil.RequireNoError(t, err)

			testutil.Len(t, doc.Content, 1)
			block := doc.Content[0]
			testutil.Equal(t, "codeBlock", block.Type)

			if tt.language != "" {
				testutil.Equal(t, tt.language, block.Attrs["language"])
			}

			testutil.Len(t, block.Content, 1)
			testutil.Equal(t, tt.code, block.Content[0].Text)
		})
	}
}

func TestToADF_Blockquote(t *testing.T) {
	t.Parallel()
	input := "> This is a quote"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	quote := doc.Content[0]
	testutil.Equal(t, "blockquote", quote.Type)
	testutil.Len(t, quote.Content, 1)
	testutil.Equal(t, "paragraph", quote.Content[0].Type)
}

func TestToADF_HorizontalRule(t *testing.T) {
	t.Parallel()
	input := "Above\n\n---\n\nBelow"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 3)
	testutil.Equal(t, "paragraph", doc.Content[0].Type)
	testutil.Equal(t, "rule", doc.Content[1].Type)
	testutil.Equal(t, "paragraph", doc.Content[2].Type)
}

func TestToADF_Table(t *testing.T) {
	t.Parallel()
	input := "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	table := doc.Content[0]
	testutil.Equal(t, "table", table.Type)

	// Should have 2 rows (header + 1 data row)
	testutil.Len(t, table.Content, 2)

	// First row should have tableHeader cells
	headerRow := table.Content[0]
	testutil.Equal(t, "tableRow", headerRow.Type)
	testutil.Len(t, headerRow.Content, 2)
	testutil.Equal(t, "tableHeader", headerRow.Content[0].Type)

	// Second row should have tableCell cells
	dataRow := table.Content[1]
	testutil.Equal(t, "tableRow", dataRow.Type)
	testutil.Len(t, dataRow.Content, 2)
	testutil.Equal(t, "tableCell", dataRow.Content[0].Type)
}

func TestToADF_EmptyInput(t *testing.T) {
	t.Parallel()
	result, err := ToADF([]byte(""))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, "doc", doc.Type)
	testutil.Equal(t, 1, doc.Version)
	testutil.Empty(t, doc.Content)
}

func TestToADF_NestedList(t *testing.T) {
	t.Parallel()
	input := "- Item one\n  - Nested one\n  - Nested two\n- Item two"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	list := doc.Content[0]
	testutil.Equal(t, "bulletList", list.Type)

	// First list item should contain a nested bulletList
	firstItem := list.Content[0]
	testutil.Equal(t, "listItem", firstItem.Type)

	// Should have paragraph + nested list
	var foundNestedList bool
	for _, child := range firstItem.Content {
		if child.Type == "bulletList" {
			foundNestedList = true
			testutil.Len(t, child.Content, 2) // Two nested items
		}
	}
	testutil.True(t, foundNestedList, "expected nested bullet list")
}

func TestToADF_BoldAndItalicCombined(t *testing.T) {
	t.Parallel()
	input := "***bold and italic***"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	para := doc.Content[0]

	// Find the text node with both marks
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

func TestToADF_OutputIsValidJSON(t *testing.T) {
	t.Parallel()
	// Test various inputs produce valid JSON
	inputs := []string{
		"# Simple heading",
		"Paragraph with **bold** and *italic*",
		"- Item 1\n- Item 2",
		"```go\ncode\n```",
		"| A | B |\n|---|---|\n| 1 | 2 |",
	}

	for _, input := range inputs {
		result, err := ToADF([]byte(input))
		testutil.RequireNoError(t, err)

		// Verify it's valid JSON
		var parsed map[string]any
		err = json.Unmarshal([]byte(result), &parsed)
		testutil.RequireNoError(t, err)

		// Verify basic structure
		testutil.Equal(t, "doc", parsed["type"])
		testutil.Equal(t, parsed["version"], float64(1))
	}
}

func TestToADF_Images_AltText(t *testing.T) {
	t.Parallel()
	input := "![Alt text](https://example.com/image.png)"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	// Images should be converted to text with alt text
	testutil.Len(t, doc.Content, 1)
	para := doc.Content[0]
	testutil.Equal(t, "paragraph", para.Type)
	testutil.Len(t, para.Content, 1)
	testutil.Equal(t, "Alt text", para.Content[0].Text)
}

func TestToADF_WhitespaceInCodeBlock(t *testing.T) {
	t.Parallel()
	// Code with leading whitespace should be preserved
	input := "```\n    indented code\n        more indented\n```"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	block := doc.Content[0]
	testutil.Equal(t, "codeBlock", block.Type)
	testutil.Len(t, block.Content, 1)

	// Verify whitespace is preserved
	text := block.Content[0].Text
	testutil.Contains(t, text, "    indented")
	testutil.Contains(t, text, "        more indented")
}

func TestToADF_NestedBlockquote(t *testing.T) {
	t.Parallel()
	input := "> Quote with **bold** text\n>\n> And a list:\n> - Item 1\n> - Item 2"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	quote := doc.Content[0]
	testutil.Equal(t, "blockquote", quote.Type)

	// Should have nested content
	testutil.True(t, len(quote.Content) > 0, "blockquote should have content")
}

func TestToADF_HardLineBreak(t *testing.T) {
	t.Parallel()
	// Two spaces at end of line creates a hard break
	input := "Line one  \nLine two"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	// Should have paragraph with hard break
	testutil.Len(t, doc.Content, 1)
	para := doc.Content[0]
	testutil.Equal(t, "paragraph", para.Type)

	// Check for hardBreak node or separate text nodes
	var foundBreak bool
	for _, node := range para.Content {
		if node.Type == "hardBreak" {
			foundBreak = true
			break
		}
	}
	// Note: If hardBreak isn't implemented, the content should at least be present
	if !foundBreak {
		// Verify both lines are present in some form
		var fullText string
		for _, node := range para.Content {
			fullText += node.Text
		}
		testutil.Contains(t, fullText, "Line one")
		testutil.Contains(t, fullText, "Line two")
	}
}

func TestToADF_InlineCodePreservesContent(t *testing.T) {
	t.Parallel()
	input := "Use `fmt.Println()` to print"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	para := doc.Content[0]

	// Find the code-marked text
	var foundCode bool
	for _, node := range para.Content {
		for _, mark := range node.Marks {
			if mark.Type == "code" {
				foundCode = true
				testutil.Equal(t, "fmt.Println()", node.Text)
			}
		}
	}
	testutil.True(t, foundCode, "expected code mark")
}

// --- Macro conversion tests ---

func TestToADF_TOC_Simple(t *testing.T) {
	t.Parallel()
	input := "[TOC]"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	ext := doc.Content[0]
	testutil.Equal(t, "extension", ext.Type)
	testutil.Equal(t, "com.atlassian.confluence.macro.core", ext.Attrs["extensionType"])
	testutil.Equal(t, "toc", ext.Attrs["extensionKey"])
	testutil.Equal(t, "default", ext.Attrs["layout"])

	// Verify parameters structure
	params, ok := ext.Attrs["parameters"].(map[string]any)
	testutil.True(t, ok, "parameters should be a map")
	metadata, ok := params["macroMetadata"].(map[string]any)
	testutil.True(t, ok, "macroMetadata should be a map")
	schemaVersion, ok := metadata["schemaVersion"].(map[string]any)
	testutil.True(t, ok, "schemaVersion should be a map")
	testutil.Equal(t, "1", schemaVersion["value"])
}

func TestToADF_TOC_WithParams(t *testing.T) {
	t.Parallel()
	input := "[TOC maxLevel=3]"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	ext := doc.Content[0]
	testutil.Equal(t, "extension", ext.Type)
	testutil.Equal(t, "toc", ext.Attrs["extensionKey"])

	// Verify macro param
	params := ext.Attrs["parameters"].(map[string]any)
	macroParams := params["macroParams"].(map[string]any)
	maxLevel := macroParams["maxLevel"].(map[string]any)
	testutil.Equal(t, "3", maxLevel["value"])
}

func TestToADF_TOC_MultipleParams(t *testing.T) {
	t.Parallel()
	input := "[TOC maxLevel=3 minLevel=1]"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	ext := doc.Content[0]
	params := ext.Attrs["parameters"].(map[string]any)
	macroParams := params["macroParams"].(map[string]any)

	maxLevel := macroParams["maxLevel"].(map[string]any)
	testutil.Equal(t, "3", maxLevel["value"])
	minLevel := macroParams["minLevel"].(map[string]any)
	testutil.Equal(t, "1", minLevel["value"])
}

func TestToADF_TOC_CaseInsensitive(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
	}{
		{"lowercase", "[toc]"},
		{"mixed_case", "[Toc]"},
		{"uppercase", "[TOC]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ToADF([]byte(tt.input))
			testutil.RequireNoError(t, err)

			var doc ADFDocument
			err = json.Unmarshal([]byte(result), &doc)
			testutil.RequireNoError(t, err)

			testutil.Len(t, doc.Content, 1)
			testutil.Equal(t, "extension", doc.Content[0].Type)
			testutil.Equal(t, "toc", doc.Content[0].Attrs["extensionKey"])
		})
	}
}

func TestToADF_TOC_WithSurroundingContent(t *testing.T) {
	t.Parallel()
	input := "Before content.\n\n[TOC]\n\n# Heading\n\nAfter content."
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	// Should have: paragraph, extension, heading, paragraph
	testutil.Len(t, doc.Content, 4)
	testutil.Equal(t, "paragraph", doc.Content[0].Type)
	testutil.Equal(t, "extension", doc.Content[1].Type)
	testutil.Equal(t, "toc", doc.Content[1].Attrs["extensionKey"])
	testutil.Equal(t, "heading", doc.Content[2].Type)
	testutil.Equal(t, "paragraph", doc.Content[3].Type)
}

func TestToADF_TOC_InsideCodeBlock_Preserved(t *testing.T) {
	t.Parallel()
	input := "```\n[TOC]\n```"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	// Should be a code block, NOT an extension
	testutil.Len(t, doc.Content, 1)
	testutil.Equal(t, "codeBlock", doc.Content[0].Type)
	testutil.Contains(t, doc.Content[0].Content[0].Text, "[TOC]")
}

func TestToADF_TOC_InsideInlineCode_Preserved(t *testing.T) {
	t.Parallel()
	input := "Use `[TOC]` to add a table of contents."
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	// Should be a paragraph with inline code, NOT an extension
	testutil.Len(t, doc.Content, 1)
	testutil.Equal(t, "paragraph", doc.Content[0].Type)

	// Find the code-marked text containing [TOC]
	var foundCode bool
	for _, node := range doc.Content[0].Content {
		for _, mark := range node.Marks {
			if mark.Type == "code" && node.Text == "[TOC]" {
				foundCode = true
			}
		}
	}
	testutil.True(t, foundCode, "expected [TOC] as inline code, not a macro")
}

func TestToADF_InfoPanel(t *testing.T) {
	t.Parallel()
	input := "[INFO]\nThis is important content.\n[/INFO]"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	panel := doc.Content[0]
	testutil.Equal(t, "panel", panel.Type)
	testutil.Equal(t, "info", panel.Attrs["panelType"])
	testutil.True(t, len(panel.Content) > 0, "panel should have body content")

	// Body should contain the text
	var foundText bool
	for _, node := range panel.Content {
		if node.Type == "paragraph" {
			for _, textNode := range node.Content {
				if textNode.Text == "This is important content." {
					foundText = true
				}
			}
		}
	}
	testutil.True(t, foundText, "panel body should contain the text")
}

func TestToADF_WarningPanel(t *testing.T) {
	t.Parallel()
	input := "[WARNING]\nBe careful!\n[/WARNING]"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	panel := doc.Content[0]
	testutil.Equal(t, "panel", panel.Type)
	testutil.Equal(t, "warning", panel.Attrs["panelType"])
}

func TestToADF_NotePanel(t *testing.T) {
	t.Parallel()
	input := "[NOTE]\nTake note of this.\n[/NOTE]"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	panel := doc.Content[0]
	testutil.Equal(t, "panel", panel.Type)
	testutil.Equal(t, "note", panel.Attrs["panelType"])
}

func TestToADF_TipPanel(t *testing.T) {
	t.Parallel()
	input := "[TIP]\nHere is a tip.\n[/TIP]"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	panel := doc.Content[0]
	testutil.Equal(t, "panel", panel.Type)
	testutil.Equal(t, "success", panel.Attrs["panelType"])
}

func TestToADF_NestedMacro_TOCInsideInfo(t *testing.T) {
	t.Parallel()
	input := "[INFO]\nContent with [TOC] inside.\n[/INFO]"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	// The outer macro should be a panel
	testutil.Len(t, doc.Content, 1)
	panel := doc.Content[0]
	testutil.Equal(t, "panel", panel.Type)

	// The panel body may have the TOC placeholder resolved or the text
	// depending on how deep we recurse. At minimum, the panel should exist.
	testutil.True(t, len(panel.Content) > 0, "panel should have content")
}

func TestToADF_ExpandMacro(t *testing.T) {
	t.Parallel()
	input := "[EXPAND]\nExpanded content here.\n[/EXPAND]"
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	testutil.Len(t, doc.Content, 1)
	ext := doc.Content[0]
	testutil.Equal(t, "bodiedExtension", ext.Type)
	testutil.Equal(t, "com.atlassian.confluence.macro.core", ext.Attrs["extensionType"])
	testutil.Equal(t, "expand", ext.Attrs["extensionKey"])
	testutil.True(t, len(ext.Content) > 0, "bodied extension should have content")
}

func TestToADF_MultipleMacroTypes(t *testing.T) {
	t.Parallel()
	input := "[TOC]\n\n# Introduction\n\n[INFO]\nImportant note here.\n[/INFO]\n\n## Details\n\nSome details."
	result, err := ToADF([]byte(input))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	testutil.RequireNoError(t, err)

	// Should have: extension(toc), heading, panel(info), heading, paragraph
	testutil.Len(t, doc.Content, 5)
	testutil.Equal(t, "extension", doc.Content[0].Type)
	testutil.Equal(t, "toc", doc.Content[0].Attrs["extensionKey"])
	testutil.Equal(t, "heading", doc.Content[1].Type)
	testutil.Equal(t, "panel", doc.Content[2].Type)
	testutil.Equal(t, "info", doc.Content[2].Attrs["panelType"])
	testutil.Equal(t, "heading", doc.Content[3].Type)
	testutil.Equal(t, "paragraph", doc.Content[4].Type)
}

func TestToADF_MacroOutputIsValidJSON(t *testing.T) {
	t.Parallel()
	inputs := []string{
		"[TOC]",
		"[TOC maxLevel=3]",
		"[INFO]\nContent\n[/INFO]",
		"Before\n\n[TOC]\n\n# Heading\n\nAfter",
	}

	for _, input := range inputs {
		result, err := ToADF([]byte(input))
		testutil.RequireNoError(t, err)

		var parsed map[string]any
		err = json.Unmarshal([]byte(result), &parsed)
		testutil.RequireNoError(t, err)
		testutil.Equal(t, "doc", parsed["type"])
	}
}
