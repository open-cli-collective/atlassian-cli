package adf

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToJSON_Paragraph(t *testing.T) {
	input := "Hello world"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
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

func TestToJSON_Headings(t *testing.T) {
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
			require.NoError(t, err)

			var doc Document
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

func TestToJSON_Formatting(t *testing.T) {
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
			result, err := ToJSON([]byte(tt.markdown))
			require.NoError(t, err)

			var doc Document
			err = json.Unmarshal([]byte(result), &doc)
			require.NoError(t, err)

			require.Len(t, doc.Content, 1)
			para := doc.Content[0]
			assert.Equal(t, "paragraph", para.Type)

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

func TestToJSON_Links(t *testing.T) {
	input := "[Example](https://example.com)"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	para := doc.Content[0]

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

func TestToJSON_BulletList(t *testing.T) {
	input := "- Item one\n- Item two\n- Item three"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
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

func TestToJSON_OrderedList(t *testing.T) {
	input := "1. First\n2. Second\n3. Third"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	list := doc.Content[0]
	assert.Equal(t, "orderedList", list.Type)
	assert.EqualValues(t, 1, list.Attrs["order"])
	assert.Len(t, list.Content, 3)
}

func TestToJSON_CodeBlock(t *testing.T) {
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
			result, err := ToJSON([]byte(tt.markdown))
			require.NoError(t, err)

			var doc Document
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

func TestToJSON_Blockquote(t *testing.T) {
	input := "> This is a quote"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	quote := doc.Content[0]
	assert.Equal(t, "blockquote", quote.Type)
	require.Len(t, quote.Content, 1)
	assert.Equal(t, "paragraph", quote.Content[0].Type)
}

func TestToJSON_HorizontalRule(t *testing.T) {
	input := "Above\n\n---\n\nBelow"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	assert.Len(t, doc.Content, 3)
	assert.Equal(t, "paragraph", doc.Content[0].Type)
	assert.Equal(t, "rule", doc.Content[1].Type)
	assert.Equal(t, "paragraph", doc.Content[2].Type)
}

func TestToJSON_Table(t *testing.T) {
	input := "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	table := doc.Content[0]
	assert.Equal(t, "table", table.Type)
	assert.Len(t, table.Content, 2)

	headerRow := table.Content[0]
	assert.Equal(t, "tableRow", headerRow.Type)
	assert.Len(t, headerRow.Content, 2)
	assert.Equal(t, "tableHeader", headerRow.Content[0].Type)

	dataRow := table.Content[1]
	assert.Equal(t, "tableRow", dataRow.Type)
	assert.Len(t, dataRow.Content, 2)
	assert.Equal(t, "tableCell", dataRow.Content[0].Type)
}

func TestToJSON_EmptyInput(t *testing.T) {
	result, err := ToJSON([]byte(""))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	assert.Equal(t, "doc", doc.Type)
	assert.Equal(t, 1, doc.Version)
	assert.Empty(t, doc.Content)
}

func TestToJSON_NestedList(t *testing.T) {
	input := "- Item one\n  - Nested one\n  - Nested two\n- Item two"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	list := doc.Content[0]
	assert.Equal(t, "bulletList", list.Type)

	firstItem := list.Content[0]
	assert.Equal(t, "listItem", firstItem.Type)

	var foundNestedList bool
	for _, child := range firstItem.Content {
		if child.Type == "bulletList" {
			foundNestedList = true
			assert.Len(t, child.Content, 2)
		}
	}
	assert.True(t, foundNestedList, "expected nested bullet list")
}

func TestToJSON_BoldAndItalicCombined(t *testing.T) {
	input := "***bold and italic***"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
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
	assert.True(t, foundStrong, "expected strong mark")
	assert.True(t, foundEm, "expected em mark")
}

func TestToJSON_OutputIsValidJSON(t *testing.T) {
	inputs := []string{
		"# Simple heading",
		"Paragraph with **bold** and *italic*",
		"- Item 1\n- Item 2",
		"```go\ncode\n```",
		"| A | B |\n|---|---|\n| 1 | 2 |",
	}

	for _, input := range inputs {
		result, err := ToJSON([]byte(input))
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err, "Output should be valid JSON for input: %s", input)

		assert.Equal(t, "doc", parsed["type"])
		assert.EqualValues(t, 1, parsed["version"])
	}
}

func TestToJSON_Images_AltText(t *testing.T) {
	input := "![Alt text](https://example.com/image.png)"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	para := doc.Content[0]
	assert.Equal(t, "paragraph", para.Type)
	require.Len(t, para.Content, 1)
	assert.Equal(t, "Alt text", para.Content[0].Text)
}

func TestToJSON_WhitespaceInCodeBlock(t *testing.T) {
	input := "```\n    indented code\n        more indented\n```"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	block := doc.Content[0]
	assert.Equal(t, "codeBlock", block.Type)
	require.Len(t, block.Content, 1)

	text := block.Content[0].Text
	assert.Contains(t, text, "    indented")
	assert.Contains(t, text, "        more indented")
}

func TestToJSON_NestedBlockquote(t *testing.T) {
	input := "> Quote with **bold** text\n>\n> And a list:\n> - Item 1\n> - Item 2"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	quote := doc.Content[0]
	assert.Equal(t, "blockquote", quote.Type)
	assert.True(t, len(quote.Content) > 0, "blockquote should have content")
}

func TestToJSON_HardLineBreak(t *testing.T) {
	input := "Line one  \nLine two"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	para := doc.Content[0]
	assert.Equal(t, "paragraph", para.Type)

	var foundBreak bool
	for _, node := range para.Content {
		if node.Type == "hardBreak" {
			foundBreak = true
			break
		}
	}
	assert.True(t, foundBreak, "expected hardBreak node")
}

func TestToJSON_InlineCodePreservesContent(t *testing.T) {
	input := "Use `fmt.Println()` to print"
	result, err := ToJSON([]byte(input))
	require.NoError(t, err)

	var doc Document
	err = json.Unmarshal([]byte(result), &doc)
	require.NoError(t, err)

	require.Len(t, doc.Content, 1)
	para := doc.Content[0]

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

func TestToDocument_Empty(t *testing.T) {
	doc := ToDocument("")
	assert.Nil(t, doc)
}

func TestToDocument_PlainText(t *testing.T) {
	doc := ToDocument("Hello world")
	require.NotNil(t, doc)
	assert.Equal(t, "doc", doc.Type)
	assert.Equal(t, 1, doc.Version)
	require.Len(t, doc.Content, 1)
	assert.Equal(t, "paragraph", doc.Content[0].Type)
	require.Len(t, doc.Content[0].Content, 1)
	assert.Equal(t, "Hello world", doc.Content[0].Content[0].Text)
}

func TestToDocument_ToPlainText(t *testing.T) {
	doc := ToDocument("# Title\n\nSome text\n\n- Item 1\n- Item 2")
	require.NotNil(t, doc)

	text := doc.ToPlainText()
	assert.Contains(t, text, "Title")
	assert.Contains(t, text, "Some text")
	assert.Contains(t, text, "Item 1")
	assert.Contains(t, text, "Item 2")
}

func TestToPlainText_Nil(t *testing.T) {
	var doc *Document
	assert.Equal(t, "", doc.ToPlainText())
}
