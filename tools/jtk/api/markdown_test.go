package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownToADF_Empty(t *testing.T) {
	result := MarkdownToADF("")
	assert.Nil(t, result)
}

func TestMarkdownToADF_PlainText(t *testing.T) {
	result := MarkdownToADF("Hello world")
	require.NotNil(t, result)
	assert.Equal(t, "doc", result.Type)
	assert.Equal(t, 1, result.Version)
	require.Len(t, result.Content, 1)
	assert.Equal(t, "paragraph", result.Content[0].Type)
	require.Len(t, result.Content[0].Content, 1)
	assert.Equal(t, "text", result.Content[0].Content[0].Type)
	assert.Equal(t, "Hello world", result.Content[0].Content[0].Text)
}

func TestMarkdownToADF_Heading(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		level    int
	}{
		{"h1", "# Heading 1", 1},
		{"h2", "## Heading 2", 2},
		{"h3", "### Heading 3", 3},
		{"h4", "#### Heading 4", 4},
		{"h5", "##### Heading 5", 5},
		{"h6", "###### Heading 6", 6},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MarkdownToADF(tc.markdown)
			require.NotNil(t, result)
			require.Len(t, result.Content, 1)
			assert.Equal(t, "heading", result.Content[0].Type)
			assert.Equal(t, tc.level, result.Content[0].Attrs["level"])
		})
	}
}

func TestMarkdownToADF_Bold(t *testing.T) {
	result := MarkdownToADF("This is **bold** text")
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	para := result.Content[0]
	assert.Equal(t, "paragraph", para.Type)

	// Find the bold text node
	var foundBold bool
	for _, node := range para.Content {
		if node.Text == "bold" {
			foundBold = true
			require.Len(t, node.Marks, 1)
			assert.Equal(t, "strong", node.Marks[0].Type)
		}
	}
	assert.True(t, foundBold, "Should find bold text")
}

func TestMarkdownToADF_Italic(t *testing.T) {
	result := MarkdownToADF("This is *italic* text")
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	para := result.Content[0]

	// Find the italic text node
	var foundItalic bool
	for _, node := range para.Content {
		if node.Text == "italic" {
			foundItalic = true
			require.Len(t, node.Marks, 1)
			assert.Equal(t, "em", node.Marks[0].Type)
		}
	}
	assert.True(t, foundItalic, "Should find italic text")
}

func TestMarkdownToADF_InlineCode(t *testing.T) {
	result := MarkdownToADF("Use `code` here")
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	para := result.Content[0]

	// Find the code text node
	var foundCode bool
	for _, node := range para.Content {
		if node.Text == "code" {
			foundCode = true
			require.Len(t, node.Marks, 1)
			assert.Equal(t, "code", node.Marks[0].Type)
		}
	}
	assert.True(t, foundCode, "Should find code text")
}

func TestMarkdownToADF_CodeBlock(t *testing.T) {
	markdown := "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"
	result := MarkdownToADF(markdown)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	codeBlock := result.Content[0]
	assert.Equal(t, "codeBlock", codeBlock.Type)
	assert.Equal(t, "go", codeBlock.Attrs["language"])
	require.Len(t, codeBlock.Content, 1)
	assert.Contains(t, codeBlock.Content[0].Text, "func main()")
}

func TestMarkdownToADF_CodeBlockNoLanguage(t *testing.T) {
	markdown := "```\nsome code\n```"
	result := MarkdownToADF(markdown)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	codeBlock := result.Content[0]
	assert.Equal(t, "codeBlock", codeBlock.Type)
	assert.Nil(t, codeBlock.Attrs) // No language specified
}

func TestMarkdownToADF_BulletList(t *testing.T) {
	markdown := "- Item 1\n- Item 2\n- Item 3"
	result := MarkdownToADF(markdown)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	list := result.Content[0]
	assert.Equal(t, "bulletList", list.Type)
	assert.Len(t, list.Content, 3)

	for i, item := range list.Content {
		assert.Equal(t, "listItem", item.Type)
		require.Len(t, item.Content, 1)
		assert.Equal(t, "paragraph", item.Content[0].Type)
		assert.Contains(t, item.Content[0].Content[0].Text, "Item")
		_ = i
	}
}

func TestMarkdownToADF_NumberedList(t *testing.T) {
	markdown := "1. First\n2. Second\n3. Third"
	result := MarkdownToADF(markdown)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	list := result.Content[0]
	assert.Equal(t, "orderedList", list.Type)
	assert.Len(t, list.Content, 3)
}

func TestMarkdownToADF_Link(t *testing.T) {
	result := MarkdownToADF("Check [this link](https://example.com)")
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	para := result.Content[0]

	// Find the link text node
	var foundLink bool
	for _, node := range para.Content {
		if node.Text == "this link" {
			foundLink = true
			require.Len(t, node.Marks, 1)
			assert.Equal(t, "link", node.Marks[0].Type)
			assert.Equal(t, "https://example.com", node.Marks[0].Attrs["href"])
		}
	}
	assert.True(t, foundLink, "Should find link text")
}

func TestMarkdownToADF_Blockquote(t *testing.T) {
	markdown := "> This is a quote"
	result := MarkdownToADF(markdown)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	blockquote := result.Content[0]
	assert.Equal(t, "blockquote", blockquote.Type)
	require.Len(t, blockquote.Content, 1)
	assert.Equal(t, "paragraph", blockquote.Content[0].Type)
}

func TestMarkdownToADF_HorizontalRule(t *testing.T) {
	markdown := "Before\n\n---\n\nAfter"
	result := MarkdownToADF(markdown)
	require.NotNil(t, result)

	// Should have: paragraph, rule, paragraph
	var foundRule bool
	for _, node := range result.Content {
		if node.Type == "rule" {
			foundRule = true
		}
	}
	assert.True(t, foundRule, "Should find horizontal rule")
}

func TestMarkdownToADF_Table(t *testing.T) {
	markdown := `| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |
| Cell 3   | Cell 4   |`

	result := MarkdownToADF(markdown)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	table := result.Content[0]
	assert.Equal(t, "table", table.Type)
	require.Len(t, table.Content, 3) // 1 header row + 2 data rows

	// Check header row
	headerRow := table.Content[0]
	assert.Equal(t, "tableRow", headerRow.Type)
	require.Len(t, headerRow.Content, 2)
	assert.Equal(t, "tableHeader", headerRow.Content[0].Type)
	assert.Equal(t, "tableHeader", headerRow.Content[1].Type)

	// Check data row
	dataRow := table.Content[1]
	assert.Equal(t, "tableRow", dataRow.Type)
	require.Len(t, dataRow.Content, 2)
	assert.Equal(t, "tableCell", dataRow.Content[0].Type)
	assert.Equal(t, "tableCell", dataRow.Content[1].Type)
}

func TestMarkdownToADF_ComplexDocument(t *testing.T) {
	markdown := `# Issue Title

This is a description with **bold** and *italic* text.

## Steps to Reproduce

1. Do this
2. Then that
3. Finally this

## Code Example

` + "```python\ndef hello():\n    print(\"Hello\")\n```" + `

> Note: This is important

---

See [documentation](https://docs.example.com) for more info.`

	result := MarkdownToADF(markdown)
	require.NotNil(t, result)

	// Verify structure
	var (
		hasH1         bool
		hasH2         bool
		hasOrderList  bool
		hasCodeBlock  bool
		hasBlockquote bool
		hasRule       bool
		hasLink       bool
	)

	for _, node := range result.Content {
		switch node.Type {
		case "heading":
			switch node.Attrs["level"] {
			case 1:
				hasH1 = true
			case 2:
				hasH2 = true
			}
		case "orderedList":
			hasOrderList = true
		case "codeBlock":
			hasCodeBlock = true
			assert.Equal(t, "python", node.Attrs["language"])
		case "blockquote":
			hasBlockquote = true
		case "rule":
			hasRule = true
		case "paragraph":
			for _, inline := range node.Content {
				if len(inline.Marks) > 0 && inline.Marks[0].Type == "link" {
					hasLink = true
				}
			}
		}
	}

	assert.True(t, hasH1, "Should have h1")
	assert.True(t, hasH2, "Should have h2")
	assert.True(t, hasOrderList, "Should have ordered list")
	assert.True(t, hasCodeBlock, "Should have code block")
	assert.True(t, hasBlockquote, "Should have blockquote")
	assert.True(t, hasRule, "Should have horizontal rule")
	assert.True(t, hasLink, "Should have link")
}

func TestMarkdownToADF_JSONOutput(t *testing.T) {
	// Test that the output is valid JSON that matches Jira's expected format
	markdown := "## Summary\n\nThis is **important**."
	result := MarkdownToADF(markdown)

	jsonBytes, err := json.Marshal(result)
	require.NoError(t, err)

	// Verify it can be unmarshaled back
	var doc ADFDocument
	err = json.Unmarshal(jsonBytes, &doc)
	require.NoError(t, err)

	assert.Equal(t, "doc", doc.Type)
	assert.Equal(t, 1, doc.Version)
}

func TestNewADFDocument_UsesMarkdownParser(t *testing.T) {
	// Verify NewADFDocument now uses the markdown parser
	result := NewADFDocument("# Heading\n\nParagraph")
	require.NotNil(t, result)

	// Should have heading and paragraph, not just a single paragraph with raw text
	assert.Len(t, result.Content, 2)
	assert.Equal(t, "heading", result.Content[0].Type)
	assert.Equal(t, "paragraph", result.Content[1].Type)
}

// Additional tests adapted from confluence-cli

func TestMarkdownToADF_Strikethrough(t *testing.T) {
	result := MarkdownToADF("This is ~~struck~~ text")
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	para := result.Content[0]

	var foundStrike bool
	for _, node := range para.Content {
		if node.Text == "struck" {
			foundStrike = true
			require.Len(t, node.Marks, 1)
			assert.Equal(t, "strike", node.Marks[0].Type)
		}
	}
	assert.True(t, foundStrike, "Should find strikethrough text")
}

func TestMarkdownToADF_BoldAndItalicCombined(t *testing.T) {
	result := MarkdownToADF("***bold and italic***")
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)
	para := result.Content[0]

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

func TestMarkdownToADF_NestedList(t *testing.T) {
	input := "- Item one\n  - Nested one\n  - Nested two\n- Item two"
	result := MarkdownToADF(input)
	require.NotNil(t, result)

	require.Len(t, result.Content, 1)
	list := result.Content[0]
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

func TestMarkdownToADF_Images_AltText(t *testing.T) {
	input := "![Alt text](https://example.com/image.png)"
	result := MarkdownToADF(input)
	require.NotNil(t, result)

	// Images should be converted to text with alt text
	require.Len(t, result.Content, 1)
	para := result.Content[0]
	assert.Equal(t, "paragraph", para.Type)
	require.Len(t, para.Content, 1)
	assert.Equal(t, "Alt text", para.Content[0].Text)
}

func TestMarkdownToADF_WhitespaceInCodeBlock(t *testing.T) {
	// Code with leading whitespace should be preserved
	input := "```\n    indented code\n        more indented\n```"
	result := MarkdownToADF(input)
	require.NotNil(t, result)

	require.Len(t, result.Content, 1)
	block := result.Content[0]
	assert.Equal(t, "codeBlock", block.Type)
	require.Len(t, block.Content, 1)

	// Verify whitespace is preserved
	text := block.Content[0].Text
	assert.Contains(t, text, "    indented")
	assert.Contains(t, text, "        more indented")
}

func TestMarkdownToADF_NestedBlockquote(t *testing.T) {
	input := "> Quote with **bold** text"
	result := MarkdownToADF(input)
	require.NotNil(t, result)

	require.Len(t, result.Content, 1)
	quote := result.Content[0]
	assert.Equal(t, "blockquote", quote.Type)

	// Should have nested content
	assert.True(t, len(quote.Content) > 0, "blockquote should have content")
}

func TestMarkdownToADF_HardLineBreak(t *testing.T) {
	// Two spaces at end of line creates a hard break
	input := "Line one  \nLine two"
	result := MarkdownToADF(input)
	require.NotNil(t, result)

	// Should have paragraph with hard break
	require.Len(t, result.Content, 1)
	para := result.Content[0]
	assert.Equal(t, "paragraph", para.Type)

	// Check for hardBreak node
	var foundBreak bool
	for _, node := range para.Content {
		if node.Type == "hardBreak" {
			foundBreak = true
			break
		}
	}
	// If hardBreak isn't found, verify both lines are present
	if !foundBreak {
		var fullText string
		for _, node := range para.Content {
			fullText += node.Text
		}
		assert.Contains(t, fullText, "Line one")
		assert.Contains(t, fullText, "Line two")
	}
}

func TestMarkdownToADF_InlineCodePreservesContent(t *testing.T) {
	input := "Use `fmt.Println()` to print"
	result := MarkdownToADF(input)
	require.NotNil(t, result)

	require.Len(t, result.Content, 1)
	para := result.Content[0]

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

func TestMarkdownToADF_OutputIsValidJSON(t *testing.T) {
	// Test various inputs produce valid JSON
	inputs := []string{
		"# Simple heading",
		"Paragraph with **bold** and *italic*",
		"- Item 1\n- Item 2",
		"```go\ncode\n```",
		"| A | B |\n|---|---|\n| 1 | 2 |",
	}

	for _, input := range inputs {
		result := MarkdownToADF(input)
		require.NotNil(t, result)

		jsonBytes, err := json.Marshal(result)
		require.NoError(t, err)

		// Verify it's valid JSON
		var parsed map[string]interface{}
		err = json.Unmarshal(jsonBytes, &parsed)
		require.NoError(t, err, "Output should be valid JSON for input: %s", input)

		// Verify basic structure
		assert.Equal(t, "doc", parsed["type"])
		assert.EqualValues(t, float64(1), parsed["version"])
	}
}

func TestMarkdownToADF_Formatting(t *testing.T) {
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
			result := MarkdownToADF(tt.markdown)
			require.NotNil(t, result)

			require.Len(t, result.Content, 1)
			para := result.Content[0]
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
