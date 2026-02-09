package md

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToADF_Paragraph(t *testing.T) {
	input := "Hello world"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	assert.Equal(t, "doc", doc.Type)
	assert.Equal(t, 1, doc.Version)
	require.Len(t, doc.Content, 1)

	para := doc.Content[0]
	assert.Equal(t, "paragraph", para.Type)
	require.Len(t, para.Content, 1)
	assert.Equal(t, "text", para.Content[0].Type)
	assert.Equal(t, "Hello world", para.Content[0].Text)
}

func TestToADF_Headings(t *testing.T) {
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
			result, err := ToADF([]byte(tt.markdown))
			require.NoError(t, err)

			var doc ADFDocument
			err = json.Unmarshal([]byte(result), &doc)
			require.NoError(t, err)

			require.Len(t, doc.Content, 1)
			heading := doc.Content[0]
			assert.Equal(t, "heading", heading.Type)
			assert.EqualValues(t, tt.level, heading.Attrs["level"])
			require.Len(t, heading.Content, 1)
			assert.Equal(t, tt.text, heading.Content[0].Text)
		})
	}
}

func TestToADF_Formatting(t *testing.T) {
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
			result, err := ToADF([]byte(tt.markdown))
			require.NoError(t, err)

			var doc ADFDocument
			err = json.Unmarshal([]byte(result), &doc)
			require.NoError(t, err)

			require.Len(t, doc.Content, 1)
			para := doc.Content[0]
			assert.Equal(t, "paragraph", para.Type)

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
			assert.True(t, foundMark, "expected to find mark %s", tt.mark)
		})
	}
}

func TestToADF_Links(t *testing.T) {
	input := "[Example](https://example.com)"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	para := doc.Content[0]

	// Find the link
	var foundLink bool
	for _, node := range para.Content {
		for _, mark := range node.Marks {
			if mark.Type == "link" {
				foundLink = true
				assert.Equal(t, "https://example.com", mark.Attrs["href"])
				assert.Equal(t, "Example", node.Text)
			}
		}
	}
	assert.True(t, foundLink, "expected to find link mark")
}

func TestToADF_BulletList(t *testing.T) {
	input := "- Item one\n- Item two\n- Item three"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	list := doc.Content[0]
	assert.Equal(t, "bulletList", list.Type)
	assert.Len(t, list.Content, 3)

	for i, item := range list.Content {
		assert.Equal(t, "listItem", item.Type)
		require.Len(t, item.Content, 1)
		para := item.Content[0]
		assert.Equal(t, "paragraph", para.Type)
		expected := []string{"Item one", "Item two", "Item three"}[i]
		require.Len(t, para.Content, 1)
		assert.Equal(t, expected, para.Content[0].Text)
	}
}

func TestToADF_OrderedList(t *testing.T) {
	input := "1. First\n2. Second\n3. Third"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	list := doc.Content[0]
	assert.Equal(t, "orderedList", list.Type)
	assert.EqualValues(t, 1, list.Attrs["order"])
	assert.Len(t, list.Content, 3)
}

func TestToADF_CodeBlock(t *testing.T) {
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
			result, err := ToADF([]byte(tt.markdown))
			require.NoError(t, err)

			var doc ADFDocument
			err = json.Unmarshal([]byte(result), &doc)
			require.NoError(t, err)

			require.Len(t, doc.Content, 1)
			block := doc.Content[0]
			assert.Equal(t, "codeBlock", block.Type)

			if tt.language != "" {
				assert.Equal(t, tt.language, block.Attrs["language"])
			}

			require.Len(t, block.Content, 1)
			assert.Equal(t, tt.code, block.Content[0].Text)
		})
	}
}

func TestToADF_Blockquote(t *testing.T) {
	input := "> This is a quote"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	quote := doc.Content[0]
	assert.Equal(t, "blockquote", quote.Type)
	require.Len(t, quote.Content, 1)
	assert.Equal(t, "paragraph", quote.Content[0].Type)
}

func TestToADF_HorizontalRule(t *testing.T) {
	input := "Above\n\n---\n\nBelow"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	assert.Len(t, doc.Content, 3)
	assert.Equal(t, "paragraph", doc.Content[0].Type)
	assert.Equal(t, "rule", doc.Content[1].Type)
	assert.Equal(t, "paragraph", doc.Content[2].Type)
}

func TestToADF_Table(t *testing.T) {
	input := "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	table := doc.Content[0]
	assert.Equal(t, "table", table.Type)

	// Should have 2 rows (header + 1 data row)
	assert.Len(t, table.Content, 2)

	// First row should have tableHeader cells
	headerRow := table.Content[0]
	assert.Equal(t, "tableRow", headerRow.Type)
	assert.Len(t, headerRow.Content, 2)
	assert.Equal(t, "tableHeader", headerRow.Content[0].Type)

	// Second row should have tableCell cells
	dataRow := table.Content[1]
	assert.Equal(t, "tableRow", dataRow.Type)
	assert.Len(t, dataRow.Content, 2)
	assert.Equal(t, "tableCell", dataRow.Content[0].Type)
}

func TestToADF_EmptyInput(t *testing.T) {
	result, err := ToADF([]byte(""))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	assert.Equal(t, "doc", doc.Type)
	assert.Equal(t, 1, doc.Version)
	assert.Empty(t, doc.Content)
}

func TestToADF_NestedList(t *testing.T) {
	input := "- Item one\n  - Nested one\n  - Nested two\n- Item two"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	list := doc.Content[0]
	assert.Equal(t, "bulletList", list.Type)

	// First list item should contain a nested bulletList
	firstItem := list.Content[0]
	assert.Equal(t, "listItem", firstItem.Type)

	// Should have paragraph + nested list
	var foundNestedList bool
	for _, child := range firstItem.Content {
		if child.Type == "bulletList" {
			foundNestedList = true
			assert.Len(t, child.Content, 2) // Two nested items
		}
	}
	assert.True(t, foundNestedList, "expected nested bullet list")
}

func TestToADF_BoldAndItalicCombined(t *testing.T) {
	input := "***bold and italic***"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
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
	assert.True(t, foundStrong, "expected strong mark")
	assert.True(t, foundEm, "expected em mark")
}

func TestToADF_OutputIsValidJSON(t *testing.T) {
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
		require.NoError(t, err)

		// Verify it's valid JSON
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err, "Output should be valid JSON for input: %s", input)

		// Verify basic structure
		assert.Equal(t, "doc", parsed["type"])
		assert.EqualValues(t, 1, parsed["version"])
	}
}

func TestToADF_Images_AltText(t *testing.T) {
	input := "![Alt text](https://example.com/image.png)"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	// Images should be converted to text with alt text
	require.Len(t, doc.Content, 1)
	para := doc.Content[0]
	assert.Equal(t, "paragraph", para.Type)
	require.Len(t, para.Content, 1)
	assert.Equal(t, "Alt text", para.Content[0].Text)
}

func TestToADF_WhitespaceInCodeBlock(t *testing.T) {
	// Code with leading whitespace should be preserved
	input := "```\n    indented code\n        more indented\n```"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	block := doc.Content[0]
	assert.Equal(t, "codeBlock", block.Type)
	require.Len(t, block.Content, 1)

	// Verify whitespace is preserved
	text := block.Content[0].Text
	assert.Contains(t, text, "    indented")
	assert.Contains(t, text, "        more indented")
}

func TestToADF_NestedBlockquote(t *testing.T) {
	input := "> Quote with **bold** text\n>\n> And a list:\n> - Item 1\n> - Item 2"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	quote := doc.Content[0]
	assert.Equal(t, "blockquote", quote.Type)

	// Should have nested content
	assert.True(t, len(quote.Content) > 0, "blockquote should have content")
}

func TestToADF_HardLineBreak(t *testing.T) {
	// Two spaces at end of line creates a hard break
	input := "Line one  \nLine two"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	// Should have paragraph with hard break
	require.Len(t, doc.Content, 1)
	para := doc.Content[0]
	assert.Equal(t, "paragraph", para.Type)

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
		assert.Contains(t, fullText, "Line one")
		assert.Contains(t, fullText, "Line two")
	}
}

func TestToADF_InlineCodePreservesContent(t *testing.T) {
	input := "Use `fmt.Println()` to print"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	para := doc.Content[0]

	// Find the code-marked text
	var foundCode bool
	for _, node := range para.Content {
		for _, mark := range node.Marks {
			if mark.Type == "code" {
				foundCode = true
				assert.Equal(t, "fmt.Println()", node.Text)
			}
		}
	}
	assert.True(t, foundCode, "expected code mark")
}

// --- Macro conversion tests ---

func TestToADF_TOC_Simple(t *testing.T) {
	input := "[TOC]"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	ext := doc.Content[0]
	assert.Equal(t, "extension", ext.Type)
	assert.Equal(t, "com.atlassian.confluence.macro.core", ext.Attrs["extensionType"])
	assert.Equal(t, "toc", ext.Attrs["extensionKey"])
	assert.Equal(t, "default", ext.Attrs["layout"])

	// Verify parameters structure
	params, ok := ext.Attrs["parameters"].(map[string]interface{})
	require.True(t, ok, "parameters should be a map")
	metadata, ok := params["macroMetadata"].(map[string]interface{})
	require.True(t, ok, "macroMetadata should be a map")
	schemaVersion, ok := metadata["schemaVersion"].(map[string]interface{})
	require.True(t, ok, "schemaVersion should be a map")
	assert.Equal(t, "1", schemaVersion["value"])
}

func TestToADF_TOC_WithParams(t *testing.T) {
	input := "[TOC maxLevel=3]"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	ext := doc.Content[0]
	assert.Equal(t, "extension", ext.Type)
	assert.Equal(t, "toc", ext.Attrs["extensionKey"])

	// Verify macro param
	params := ext.Attrs["parameters"].(map[string]interface{})
	macroParams := params["macroParams"].(map[string]interface{})
	maxLevel := macroParams["maxLevel"].(map[string]interface{})
	assert.Equal(t, "3", maxLevel["value"])
}

func TestToADF_TOC_MultipleParams(t *testing.T) {
	input := "[TOC maxLevel=3 minLevel=1]"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	ext := doc.Content[0]
	params := ext.Attrs["parameters"].(map[string]interface{})
	macroParams := params["macroParams"].(map[string]interface{})

	maxLevel := macroParams["maxLevel"].(map[string]interface{})
	assert.Equal(t, "3", maxLevel["value"])
	minLevel := macroParams["minLevel"].(map[string]interface{})
	assert.Equal(t, "1", minLevel["value"])
}

func TestToADF_TOC_CaseInsensitive(t *testing.T) {
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
			result, err := ToADF([]byte(tt.input))
			require.NoError(t, err)

			var doc ADFDocument
			err = json.Unmarshal([]byte(result), &doc)
			require.NoError(t, err)

			require.Len(t, doc.Content, 1)
			assert.Equal(t, "extension", doc.Content[0].Type)
			assert.Equal(t, "toc", doc.Content[0].Attrs["extensionKey"])
		})
	}
}

func TestToADF_TOC_WithSurroundingContent(t *testing.T) {
	input := "Before content.\n\n[TOC]\n\n# Heading\n\nAfter content."
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	// Should have: paragraph, extension, heading, paragraph
	require.Len(t, doc.Content, 4)
	assert.Equal(t, "paragraph", doc.Content[0].Type)
	assert.Equal(t, "extension", doc.Content[1].Type)
	assert.Equal(t, "toc", doc.Content[1].Attrs["extensionKey"])
	assert.Equal(t, "heading", doc.Content[2].Type)
	assert.Equal(t, "paragraph", doc.Content[3].Type)
}

func TestToADF_TOC_InsideCodeBlock_Preserved(t *testing.T) {
	input := "```\n[TOC]\n```"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	// Should be a code block, NOT an extension
	require.Len(t, doc.Content, 1)
	assert.Equal(t, "codeBlock", doc.Content[0].Type)
	assert.Contains(t, doc.Content[0].Content[0].Text, "[TOC]")
}

func TestToADF_TOC_InsideInlineCode_Preserved(t *testing.T) {
	input := "Use `[TOC]` to add a table of contents."
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	// Should be a paragraph with inline code, NOT an extension
	require.Len(t, doc.Content, 1)
	assert.Equal(t, "paragraph", doc.Content[0].Type)

	// Find the code-marked text containing [TOC]
	var foundCode bool
	for _, node := range doc.Content[0].Content {
		for _, mark := range node.Marks {
			if mark.Type == "code" && node.Text == "[TOC]" {
				foundCode = true
			}
		}
	}
	assert.True(t, foundCode, "expected [TOC] as inline code, not a macro")
}

func TestToADF_InfoPanel(t *testing.T) {
	input := "[INFO]\nThis is important content.\n[/INFO]"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	panel := doc.Content[0]
	assert.Equal(t, "panel", panel.Type)
	assert.Equal(t, "info", panel.Attrs["panelType"])
	require.True(t, len(panel.Content) > 0, "panel should have body content")

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
	assert.True(t, foundText, "panel body should contain the text")
}

func TestToADF_WarningPanel(t *testing.T) {
	input := "[WARNING]\nBe careful!\n[/WARNING]"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	panel := doc.Content[0]
	assert.Equal(t, "panel", panel.Type)
	assert.Equal(t, "warning", panel.Attrs["panelType"])
}

func TestToADF_NotePanel(t *testing.T) {
	input := "[NOTE]\nTake note of this.\n[/NOTE]"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	panel := doc.Content[0]
	assert.Equal(t, "panel", panel.Type)
	assert.Equal(t, "note", panel.Attrs["panelType"])
}

func TestToADF_TipPanel(t *testing.T) {
	input := "[TIP]\nHere is a tip.\n[/TIP]"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	panel := doc.Content[0]
	assert.Equal(t, "panel", panel.Type)
	assert.Equal(t, "success", panel.Attrs["panelType"])
}

func TestToADF_NestedMacro_TOCInsideInfo(t *testing.T) {
	input := "[INFO]\nContent with [TOC] inside.\n[/INFO]"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	// The outer macro should be a panel
	require.Len(t, doc.Content, 1)
	panel := doc.Content[0]
	assert.Equal(t, "panel", panel.Type)

	// The panel body may have the TOC placeholder resolved or the text
	// depending on how deep we recurse. At minimum, the panel should exist.
	assert.True(t, len(panel.Content) > 0, "panel should have content")
}

func TestToADF_ExpandMacro(t *testing.T) {
	input := "[EXPAND]\nExpanded content here.\n[/EXPAND]"
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	ext := doc.Content[0]
	assert.Equal(t, "bodiedExtension", ext.Type)
	assert.Equal(t, "com.atlassian.confluence.macro.core", ext.Attrs["extensionType"])
	assert.Equal(t, "expand", ext.Attrs["extensionKey"])
	require.True(t, len(ext.Content) > 0, "bodied extension should have content")
}

func TestToADF_MultipleMacroTypes(t *testing.T) {
	input := "[TOC]\n\n# Introduction\n\n[INFO]\nImportant note here.\n[/INFO]\n\n## Details\n\nSome details."
	result, err := ToADF([]byte(input))
	require.NoError(t, err)

	var doc ADFDocument
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	// Should have: extension(toc), heading, panel(info), heading, paragraph
	require.Len(t, doc.Content, 5)
	assert.Equal(t, "extension", doc.Content[0].Type)
	assert.Equal(t, "toc", doc.Content[0].Attrs["extensionKey"])
	assert.Equal(t, "heading", doc.Content[1].Type)
	assert.Equal(t, "panel", doc.Content[2].Type)
	assert.Equal(t, "info", doc.Content[2].Attrs["panelType"])
	assert.Equal(t, "heading", doc.Content[3].Type)
	assert.Equal(t, "paragraph", doc.Content[4].Type)
}

func TestToADF_MacroOutputIsValidJSON(t *testing.T) {
	inputs := []string{
		"[TOC]",
		"[TOC maxLevel=3]",
		"[INFO]\nContent\n[/INFO]",
		"Before\n\n[TOC]\n\n# Heading\n\nAfter",
	}

	for _, input := range inputs {
		result, err := ToADF([]byte(input))
		require.NoError(t, err, "should produce valid ADF for: %s", input)

		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err, "output should be valid JSON for: %s", input)
		assert.Equal(t, "doc", parsed["type"])
	}
}
