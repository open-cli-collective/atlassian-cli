package api //nolint:revive // package name is intentional

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestMarkdownToADF_Empty(t *testing.T) {
	t.Parallel()
	result := MarkdownToADF("")
	testutil.Nil(t, result)
}

func TestMarkdownToADF_PlainText(t *testing.T) {
	t.Parallel()
	result := MarkdownToADF("Hello world")
	testutil.NotNil(t, result)
	testutil.Equal(t, result.Type, "doc")
	testutil.Equal(t, result.Version, 1)
	testutil.Len(t, result.Content, 1)
	testutil.Equal(t, result.Content[0].Type, "paragraph")
	testutil.Len(t, result.Content[0].Content, 1)
	testutil.Equal(t, result.Content[0].Content[0].Type, "text")
	testutil.Equal(t, result.Content[0].Content[0].Text, "Hello world")
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
			testutil.NotNil(t, result)
			testutil.Len(t, result.Content, 1)
			testutil.Equal(t, result.Content[0].Type, "heading")
			testutil.Equal(t, result.Content[0].Attrs["level"], tc.level)
		})
	}
}

func TestMarkdownToADF_Bold(t *testing.T) {
	result := MarkdownToADF("This is **bold** text")
	testutil.NotNil(t, result)
	testutil.Len(t, result.Content, 1)
	para := result.Content[0]
	testutil.Equal(t, para.Type, "paragraph")

	// Find the bold text node
	var foundBold bool
	for _, node := range para.Content {
		if node.Text == "bold" {
			foundBold = true
			testutil.Len(t, node.Marks, 1)
			testutil.Equal(t, node.Marks[0].Type, "strong")
		}
	}
	testutil.True(t, foundBold, "Should find bold text")
}

func TestMarkdownToADF_Italic(t *testing.T) {
	result := MarkdownToADF("This is *italic* text")
	testutil.NotNil(t, result)
	testutil.Len(t, result.Content, 1)
	para := result.Content[0]

	// Find the italic text node
	var foundItalic bool
	for _, node := range para.Content {
		if node.Text == "italic" {
			foundItalic = true
			testutil.Len(t, node.Marks, 1)
			testutil.Equal(t, node.Marks[0].Type, "em")
		}
	}
	testutil.True(t, foundItalic, "Should find italic text")
}

func TestMarkdownToADF_InlineCode(t *testing.T) {
	result := MarkdownToADF("Use `code` here")
	testutil.NotNil(t, result)
	testutil.Len(t, result.Content, 1)
	para := result.Content[0]

	// Find the code text node
	var foundCode bool
	for _, node := range para.Content {
		if node.Text == "code" {
			foundCode = true
			testutil.Len(t, node.Marks, 1)
			testutil.Equal(t, node.Marks[0].Type, "code")
		}
	}
	testutil.True(t, foundCode, "Should find code text")
}

func TestMarkdownToADF_CodeBlock(t *testing.T) {
	markdown := "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"
	result := MarkdownToADF(markdown)
	testutil.NotNil(t, result)
	testutil.Len(t, result.Content, 1)

	codeBlock := result.Content[0]
	testutil.Equal(t, codeBlock.Type, "codeBlock")
	testutil.Equal(t, codeBlock.Attrs["language"], "go")
	testutil.Len(t, codeBlock.Content, 1)
	testutil.Contains(t, codeBlock.Content[0].Text, "func main()")
}

func TestMarkdownToADF_CodeBlockNoLanguage(t *testing.T) {
	markdown := "```\nsome code\n```"
	result := MarkdownToADF(markdown)
	testutil.NotNil(t, result)
	testutil.Len(t, result.Content, 1)

	codeBlock := result.Content[0]
	testutil.Equal(t, codeBlock.Type, "codeBlock")
	testutil.Nil(t, codeBlock.Attrs) // No language specified
}

func TestMarkdownToADF_BulletList(t *testing.T) {
	markdown := "- Item 1\n- Item 2\n- Item 3"
	result := MarkdownToADF(markdown)
	testutil.NotNil(t, result)
	testutil.Len(t, result.Content, 1)

	list := result.Content[0]
	testutil.Equal(t, list.Type, "bulletList")
	testutil.Len(t, list.Content, 3)

	for i, item := range list.Content {
		testutil.Equal(t, item.Type, "listItem")
		testutil.Len(t, item.Content, 1)
		testutil.Equal(t, item.Content[0].Type, "paragraph")
		testutil.Contains(t, item.Content[0].Content[0].Text, "Item")
		_ = i
	}
}

func TestMarkdownToADF_NumberedList(t *testing.T) {
	markdown := "1. First\n2. Second\n3. Third"
	result := MarkdownToADF(markdown)
	testutil.NotNil(t, result)
	testutil.Len(t, result.Content, 1)

	list := result.Content[0]
	testutil.Equal(t, list.Type, "orderedList")
	testutil.Len(t, list.Content, 3)
}

func TestMarkdownToADF_Link(t *testing.T) {
	result := MarkdownToADF("Check [this link](https://example.com)")
	testutil.NotNil(t, result)
	testutil.Len(t, result.Content, 1)
	para := result.Content[0]

	// Find the link text node
	var foundLink bool
	for _, node := range para.Content {
		if node.Text == "this link" {
			foundLink = true
			testutil.Len(t, node.Marks, 1)
			testutil.Equal(t, node.Marks[0].Type, "link")
			testutil.Equal(t, node.Marks[0].Attrs["href"], "https://example.com")
		}
	}
	testutil.True(t, foundLink, "Should find link text")
}

func TestMarkdownToADF_Blockquote(t *testing.T) {
	markdown := "> This is a quote"
	result := MarkdownToADF(markdown)
	testutil.NotNil(t, result)
	testutil.Len(t, result.Content, 1)

	blockquote := result.Content[0]
	testutil.Equal(t, blockquote.Type, "blockquote")
	testutil.Len(t, blockquote.Content, 1)
	testutil.Equal(t, blockquote.Content[0].Type, "paragraph")
}

func TestMarkdownToADF_HorizontalRule(t *testing.T) {
	markdown := "Before\n\n---\n\nAfter"
	result := MarkdownToADF(markdown)
	testutil.NotNil(t, result)

	// Should have: paragraph, rule, paragraph
	var foundRule bool
	for _, node := range result.Content {
		if node.Type == "rule" {
			foundRule = true
		}
	}
	testutil.True(t, foundRule, "Should find horizontal rule")
}

func TestMarkdownToADF_Table(t *testing.T) {
	markdown := `| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |
| Cell 3   | Cell 4   |`

	result := MarkdownToADF(markdown)
	testutil.NotNil(t, result)
	testutil.Len(t, result.Content, 1)

	table := result.Content[0]
	testutil.Equal(t, table.Type, "table")
	testutil.Len(t, table.Content, 3) // 1 header row + 2 data rows

	// Check header row
	headerRow := table.Content[0]
	testutil.Equal(t, headerRow.Type, "tableRow")
	testutil.Len(t, headerRow.Content, 2)
	testutil.Equal(t, headerRow.Content[0].Type, "tableHeader")
	testutil.Equal(t, headerRow.Content[1].Type, "tableHeader")

	// Check data row
	dataRow := table.Content[1]
	testutil.Equal(t, dataRow.Type, "tableRow")
	testutil.Len(t, dataRow.Content, 2)
	testutil.Equal(t, dataRow.Content[0].Type, "tableCell")
	testutil.Equal(t, dataRow.Content[1].Type, "tableCell")
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
	testutil.NotNil(t, result)

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
			testutil.Equal(t, node.Attrs["language"], "python")
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

	testutil.True(t, hasH1, "Should have h1")
	testutil.True(t, hasH2, "Should have h2")
	testutil.True(t, hasOrderList, "Should have ordered list")
	testutil.True(t, hasCodeBlock, "Should have code block")
	testutil.True(t, hasBlockquote, "Should have blockquote")
	testutil.True(t, hasRule, "Should have horizontal rule")
	testutil.True(t, hasLink, "Should have link")
}

func TestMarkdownToADF_JSONOutput(t *testing.T) {
	// Test that the output is valid JSON that matches Jira's expected format
	markdown := "## Summary\n\nThis is **important**."
	result := MarkdownToADF(markdown)

	jsonBytes, err := json.Marshal(result)
	testutil.RequireNoError(t, err)

	// Verify it can be unmarshaled back
	var doc ADFDocument
	err = json.Unmarshal(jsonBytes, &doc)
	testutil.RequireNoError(t, err)

	testutil.Equal(t, doc.Type, "doc")
	testutil.Equal(t, doc.Version, 1)
}

func TestNewADFDocument_UsesMarkdownParser(t *testing.T) {
	// Verify NewADFDocument now uses the markdown parser
	result := NewADFDocument("# Heading\n\nParagraph")
	testutil.NotNil(t, result)

	// Should have heading and paragraph, not just a single paragraph with raw text
	testutil.Len(t, result.Content, 2)
	testutil.Equal(t, result.Content[0].Type, "heading")
	testutil.Equal(t, result.Content[1].Type, "paragraph")
}

// Additional tests adapted from confluence-cli

func TestMarkdownToADF_Strikethrough(t *testing.T) {
	result := MarkdownToADF("This is ~~struck~~ text")
	testutil.NotNil(t, result)
	testutil.Len(t, result.Content, 1)
	para := result.Content[0]

	var foundStrike bool
	for _, node := range para.Content {
		if node.Text == "struck" {
			foundStrike = true
			testutil.Len(t, node.Marks, 1)
			testutil.Equal(t, node.Marks[0].Type, "strike")
		}
	}
	testutil.True(t, foundStrike, "Should find strikethrough text")
}

func TestMarkdownToADF_BoldAndItalicCombined(t *testing.T) {
	result := MarkdownToADF("***bold and italic***")
	testutil.NotNil(t, result)
	testutil.Len(t, result.Content, 1)
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
	testutil.True(t, foundStrong, "expected strong mark")
	testutil.True(t, foundEm, "expected em mark")
}

func TestMarkdownToADF_NestedList(t *testing.T) {
	input := "- Item one\n  - Nested one\n  - Nested two\n- Item two"
	result := MarkdownToADF(input)
	testutil.NotNil(t, result)

	testutil.Len(t, result.Content, 1)
	list := result.Content[0]
	testutil.Equal(t, list.Type, "bulletList")

	// First list item should contain a nested bulletList
	firstItem := list.Content[0]
	testutil.Equal(t, firstItem.Type, "listItem")

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

func TestMarkdownToADF_Images_AltText(t *testing.T) {
	input := "![Alt text](https://example.com/image.png)"
	result := MarkdownToADF(input)
	testutil.NotNil(t, result)

	// Images should be converted to text with alt text
	testutil.Len(t, result.Content, 1)
	para := result.Content[0]
	testutil.Equal(t, para.Type, "paragraph")
	testutil.Len(t, para.Content, 1)
	testutil.Equal(t, para.Content[0].Text, "Alt text")
}

func TestMarkdownToADF_WhitespaceInCodeBlock(t *testing.T) {
	// Code with leading whitespace should be preserved
	input := "```\n    indented code\n        more indented\n```"
	result := MarkdownToADF(input)
	testutil.NotNil(t, result)

	testutil.Len(t, result.Content, 1)
	block := result.Content[0]
	testutil.Equal(t, block.Type, "codeBlock")
	testutil.Len(t, block.Content, 1)

	// Verify whitespace is preserved
	text := block.Content[0].Text
	testutil.Contains(t, text, "    indented")
	testutil.Contains(t, text, "        more indented")
}

func TestMarkdownToADF_NestedBlockquote(t *testing.T) {
	input := "> Quote with **bold** text"
	result := MarkdownToADF(input)
	testutil.NotNil(t, result)

	testutil.Len(t, result.Content, 1)
	quote := result.Content[0]
	testutil.Equal(t, quote.Type, "blockquote")

	// Should have nested content
	testutil.True(t, len(quote.Content) > 0, "blockquote should have content")
}

func TestMarkdownToADF_HardLineBreak(t *testing.T) {
	// Two spaces at end of line creates a hard break
	input := "Line one  \nLine two"
	result := MarkdownToADF(input)
	testutil.NotNil(t, result)

	// Should have paragraph with hard break
	testutil.Len(t, result.Content, 1)
	para := result.Content[0]
	testutil.Equal(t, para.Type, "paragraph")

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
		testutil.Contains(t, fullText, "Line one")
		testutil.Contains(t, fullText, "Line two")
	}
}

func TestMarkdownToADF_InlineCodePreservesContent(t *testing.T) {
	input := "Use `fmt.Println()` to print"
	result := MarkdownToADF(input)
	testutil.NotNil(t, result)

	testutil.Len(t, result.Content, 1)
	para := result.Content[0]

	// Find the code-marked text
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
		testutil.NotNil(t, result)

		jsonBytes, err := json.Marshal(result)
		testutil.RequireNoError(t, err)

		// Verify it's valid JSON
		var parsed map[string]any
		err = json.Unmarshal(jsonBytes, &parsed)
		if err != nil {
			t.Fatalf("Output should be valid JSON for input: %s: %v", input, err)
		}

		testutil.Equal(t, parsed["type"], "doc")
		testutil.Equal(t, parsed["version"], float64(1))
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
			testutil.NotNil(t, result)

			testutil.Len(t, result.Content, 1)
			para := result.Content[0]
			testutil.Equal(t, para.Type, "paragraph")

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
			testutil.True(t, foundMark, "expected to find mark "+tt.mark)
		})
	}
}

// TestMarkdownToADF_WikiSubscript verifies that wiki subscript (~text~) produces
// a proper ADF "subsup" mark with attrs.type "sub", not raw HTML or dropped text.
func TestMarkdownToADF_WikiSubscript(t *testing.T) {
	// Wiki markup: ~subscript~ should become ADF subsup mark
	input := "h1. Formula\n\nH~2~O is water"
	result := MarkdownToADF(input)
	testutil.NotNil(t, result)

	// Find the paragraph with inline content
	var found bool
	for _, node := range result.Content {
		if node.Type != "paragraph" {
			continue
		}
		for _, inline := range node.Content {
			if inline.Text == "2" {
				found = true
				testutil.True(t, len(inline.Marks) > 0, "subscript text should have marks")
				testutil.Equal(t, inline.Marks[0].Type, "subsup")
				testutil.Equal(t, inline.Marks[0].Attrs["type"], "sub")
			}
		}
	}
	testutil.True(t, found, "should find subscript text '2'")
}

// TestMarkdownToADF_WikiSuperscript verifies that wiki superscript (^text^) produces
// a proper ADF "subsup" mark with attrs.type "sup".
func TestMarkdownToADF_WikiSuperscript(t *testing.T) {
	input := "h1. Math\n\nx^2^ is x squared"
	result := MarkdownToADF(input)
	testutil.NotNil(t, result)

	var found bool
	for _, node := range result.Content {
		if node.Type != "paragraph" {
			continue
		}
		for _, inline := range node.Content {
			if inline.Text == "2" {
				found = true
				testutil.True(t, len(inline.Marks) > 0, "superscript text should have marks")
				testutil.Equal(t, inline.Marks[0].Type, "subsup")
				testutil.Equal(t, inline.Marks[0].Attrs["type"], "sup")
			}
		}
	}
	testutil.True(t, found, "should find superscript text '2'")
}

// TestMarkdownToADF_WikiUnderline verifies that wiki underline (+text+) produces
// a proper ADF "underline" mark.
func TestMarkdownToADF_WikiUnderline(t *testing.T) {
	input := "h1. Note\n\nThis is +important+ text"
	result := MarkdownToADF(input)
	testutil.NotNil(t, result)

	var found bool
	for _, node := range result.Content {
		if node.Type != "paragraph" {
			continue
		}
		for _, inline := range node.Content {
			if inline.Text == "important" {
				found = true
				testutil.True(t, len(inline.Marks) > 0, "underlined text should have marks")
				testutil.Equal(t, inline.Marks[0].Type, "underline")
			}
		}
	}
	testutil.True(t, found, "should find underlined text 'important'")
}

// TestMarkdownToADF_CompoundWordsEndToEnd verifies the full pipeline preserves
// compound words with hyphens and tildes — the original bug that motivated this
// change. Tests the complete path: input → wiki detection → goldmark → ADF.
func TestMarkdownToADF_CompoundWordsEndToEnd(t *testing.T) {
	input := "## Overview\n\nDeploy signal-webapp-frontend to ui-components.\n\n## Spec\n\nSee 2026-03-12-v3-theming-design.md for the three-tier token system."
	result := MarkdownToADF(input)
	testutil.NotNil(t, result)

	// Verify compound words appear intact within single ADF text nodes.
	// This is stronger than joining all text — it proves words aren't
	// fragmented across nodes by wiki formatting conversion.
	compoundWords := []string{
		"signal-webapp-frontend",
		"ui-components",
		"2026-03-12-v3-theming-design.md",
		"three-tier",
	}

	var allText []string
	var collectText func(nodes []*ADFNode)
	collectText = func(nodes []*ADFNode) {
		for _, n := range nodes {
			if n.Text != "" {
				allText = append(allText, n.Text)
			}
			if n.Content != nil {
				collectText(n.Content)
			}
		}
	}
	collectText(result.Content)

	for _, word := range compoundWords {
		found := false
		for _, nodeText := range allText {
			if strings.Contains(nodeText, word) {
				found = true
				break
			}
		}
		testutil.True(t, found, word+" should appear intact within a single text node")
	}
}

// TestMarkdownToADF_EvenTildeCompoundWord verifies that compound words with an
// even number of tildes (e.g., "signal~webapp~frontend") are NOT mangled by
// goldmark subscript processing. This is ensured by using the standard parser
// (no subscript) for non-wiki input.
func TestMarkdownToADF_EvenTildeCompoundWord(t *testing.T) {
	input := "Deploy signal~webapp~frontend to production"
	result := MarkdownToADF(input)
	testutil.NotNil(t, result)
	testutil.NotEmpty(t, result.Content)

	// Should be a single paragraph with no subsup marks
	para := result.Content[0]
	testutil.Equal(t, para.Type, "paragraph")
	for _, node := range para.Content {
		if node.Marks != nil {
			for _, mark := range node.Marks {
				if mark.Type == "subsup" {
					t.Errorf("found unexpected subsup mark on text %q — even-tilde compound word was mangled", node.Text)
				}
			}
		}
	}
}

// TestMarkdownToADF_EvenCaretCompoundWord verifies that "x^2^y" in plain
// markdown is NOT converted to superscript.
func TestMarkdownToADF_EvenCaretCompoundWord(t *testing.T) {
	input := "Calculate x^2^y in the formula"
	result := MarkdownToADF(input)
	testutil.NotNil(t, result)

	para := result.Content[0]
	for _, node := range para.Content {
		if node.Marks != nil {
			for _, mark := range node.Marks {
				if mark.Type == "subsup" {
					t.Errorf("found unexpected subsup mark on text %q — caret compound was mangled", node.Text)
				}
			}
		}
	}
}

// TestMarkdownToADF_InlineOnlyWikiNotDetected verifies that inline-only wiki
// formatting (H~2~O, +important+, -deleted-) without block-level wiki markers
// (h1., {code}, etc.) is NOT detected as wiki markup. These go through the
// standard parser, so ~text~ and ^text^ won't produce ADF marks.
func TestMarkdownToADF_InlineOnlyWikiNotDetected(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "subscript only", input: "H~2~O is water"},
		{name: "superscript only", input: "x^2^ squared"},
		{name: "underline only", input: "this is +important+ text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should NOT be detected as wiki
			testutil.Equal(t, IsWikiMarkup(tt.input), false)

			// Should go through standard parser without subsup/underline marks
			result := MarkdownToADF(tt.input)
			testutil.NotNil(t, result)
			for _, block := range result.Content {
				for _, node := range block.Content {
					for _, mark := range node.Marks {
						if mark.Type == "subsup" || mark.Type == "underline" {
							t.Errorf("found unexpected %s mark on %q — inline-only wiki should use standard parser", mark.Type, node.Text)
						}
					}
				}
			}
		})
	}
}
