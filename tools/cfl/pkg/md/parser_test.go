package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Bracket Parser Tests ====================

func TestParseBracketMacros_EmptyInput(t *testing.T) {
	result, err := ParseBracketMacros("")
	require.NoError(t, err)
	assert.Empty(t, result.Segments)
}

func TestParseBracketMacros_PlainText(t *testing.T) {
	result, err := ParseBracketMacros("Hello world")
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	assert.Equal(t, SegmentText, result.Segments[0].Type)
	assert.Equal(t, "Hello world", result.Segments[0].Text)
}

func TestParseBracketMacros_SimpleTOC(t *testing.T) {
	result, err := ParseBracketMacros("[TOC]")
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	assert.Equal(t, SegmentMacro, result.Segments[0].Type)
	assert.Equal(t, "toc", result.Segments[0].Macro.Name)
}

func TestParseBracketMacros_TOCWithParams(t *testing.T) {
	result, err := ParseBracketMacros("[TOC maxLevel=3 minLevel=1]")
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "toc", macro.Name)
	assert.Equal(t, "3", macro.Parameters["maxLevel"])
	assert.Equal(t, "1", macro.Parameters["minLevel"])
}

func TestParseBracketMacros_PanelWithBody(t *testing.T) {
	result, err := ParseBracketMacros("[INFO]Content here[/INFO]")
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "info", macro.Name)
	assert.Equal(t, "Content here", macro.Body)
}

func TestParseBracketMacros_PanelWithTitleAndBody(t *testing.T) {
	result, err := ParseBracketMacros(`[WARNING title="Watch Out"]Be careful![/WARNING]`)
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "warning", macro.Name)
	assert.Equal(t, "Watch Out", macro.Parameters["title"])
	assert.Equal(t, "Be careful!", macro.Body)
}

func TestParseBracketMacros_NestedMacros(t *testing.T) {
	result, err := ParseBracketMacros("[INFO]Before [TOC] after[/INFO]")
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "info", macro.Name)
	// TOC should be a child
	require.Len(t, macro.Children, 1)
	assert.Equal(t, "toc", macro.Children[0].Name)
}

func TestParseBracketMacros_MultipleMacros(t *testing.T) {
	result, err := ParseBracketMacros("Before [TOC] middle [INFO]content[/INFO] after")
	require.NoError(t, err)

	// Should have: text, macro, text, macro, text
	textCount := 0
	macroCount := 0
	for _, seg := range result.Segments {
		if seg.Type == SegmentText {
			textCount++
		} else {
			macroCount++
		}
	}
	assert.Equal(t, 2, macroCount, "should have 2 macros")
	assert.GreaterOrEqual(t, textCount, 2, "should have at least 2 text segments")
}

func TestParseBracketMacros_UnknownMacro(t *testing.T) {
	result, err := ParseBracketMacros("[UNKNOWN]content[/UNKNOWN]")
	require.NoError(t, err)
	// Unknown macro should be treated as text
	assert.GreaterOrEqual(t, len(result.Warnings), 1, "should have warning")
	// Content should be in text segments
	hasText := false
	for _, seg := range result.Segments {
		if seg.Type == SegmentText {
			hasText = true
		}
	}
	assert.True(t, hasText, "unknown macro should be preserved as text")
}

func TestParseBracketMacros_MismatchedClose(t *testing.T) {
	result, err := ParseBracketMacros("[INFO]content[/WARNING]more[/INFO]")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Warnings), 1, "should have warning for mismatch")
}

func TestParseBracketMacros_UnclosedMacro(t *testing.T) {
	result, err := ParseBracketMacros("[INFO]content without close")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Warnings), 1, "should have warning for unclosed")
	// Should be treated as text
	hasText := false
	for _, seg := range result.Segments {
		if seg.Type == SegmentText {
			hasText = true
		}
	}
	assert.True(t, hasText, "unclosed macro should be preserved as text")
}

// ==================== XML Parser Tests ====================

func TestParseConfluenceXML_EmptyInput(t *testing.T) {
	result, err := ParseConfluenceXML("")
	require.NoError(t, err)
	assert.Empty(t, result.Segments)
}

func TestParseConfluenceXML_PlainHTML(t *testing.T) {
	result, err := ParseConfluenceXML("<p>Hello world</p>")
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	assert.Equal(t, SegmentText, result.Segments[0].Type)
	assert.Equal(t, "<p>Hello world</p>", result.Segments[0].Text)
}

func TestParseConfluenceXML_SimpleTOC(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	assert.Equal(t, SegmentMacro, result.Segments[0].Type)
	assert.Equal(t, "toc", result.Segments[0].Macro.Name)
}

func TestParseConfluenceXML_TOCWithParams(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"><ac:parameter ac:name="maxLevel">3</ac:parameter><ac:parameter ac:name="minLevel">1</ac:parameter></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "toc", macro.Name)
	assert.Equal(t, "3", macro.Parameters["maxLevel"])
	assert.Equal(t, "1", macro.Parameters["minLevel"])
}

func TestParseConfluenceXML_PanelWithBody(t *testing.T) {
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><p>Content</p></ac:rich-text-body></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "info", macro.Name)
	assert.Contains(t, macro.Body, "Content")
}

func TestParseConfluenceXML_CodeWithCDATA(t *testing.T) {
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:parameter ac:name="language">python</ac:parameter><ac:plain-text-body><![CDATA[print("Hello")]]></ac:plain-text-body></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "code", macro.Name)
	assert.Equal(t, "python", macro.Parameters["language"])
	assert.Contains(t, macro.Body, `print("Hello")`)
}

func TestParseConfluenceXML_NestedMacros(t *testing.T) {
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro></ac:rich-text-body></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "info", macro.Name)
	require.Len(t, macro.Children, 1)
	assert.Equal(t, "toc", macro.Children[0].Name)
}

func TestParseConfluenceXML_WithSurroundingHTML(t *testing.T) {
	input := `<h1>Title</h1><ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro><p>Content</p>`
	result, err := ParseConfluenceXML(input)
	require.NoError(t, err)

	// Should have text, macro, text
	textCount := 0
	macroCount := 0
	for _, seg := range result.Segments {
		if seg.Type == SegmentText {
			textCount++
		} else {
			macroCount++
		}
	}
	assert.Equal(t, 1, macroCount)
	assert.Equal(t, 2, textCount)
}

// ==================== Segment Tests ====================

func TestParseResult_GetMacros(t *testing.T) {
	result := &ParseResult{}
	result.AddTextSegment("text1")
	result.AddMacroSegment(&MacroNode{Name: "toc"})
	result.AddTextSegment("text2")
	result.AddMacroSegment(&MacroNode{Name: "info"})

	macros := result.GetMacros()
	require.Len(t, macros, 2)
	assert.Equal(t, "toc", macros[0].Name)
	assert.Equal(t, "info", macros[1].Name)
}

func TestParseResult_MergeAdjacentText(t *testing.T) {
	result := &ParseResult{}
	result.AddTextSegment("hello ")
	result.AddTextSegment("world")

	// Should be merged into single segment
	require.Len(t, result.Segments, 1)
	assert.Equal(t, "hello world", result.Segments[0].Text)
}
