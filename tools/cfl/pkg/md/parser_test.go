package md

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

// ==================== Bracket Parser Tests ====================

func TestParseBracketMacros_EmptyInput(t *testing.T) {
	t.Parallel()
	result, err := ParseBracketMacros("")
	testutil.RequireNoError(t, err)
	testutil.Empty(t, result.Segments)
}

func TestParseBracketMacros_PlainText(t *testing.T) {
	t.Parallel()
	result, err := ParseBracketMacros("Hello world")
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Segments, 1)
	testutil.Equal(t, SegmentText, result.Segments[0].Type)
	testutil.Equal(t, "Hello world", result.Segments[0].Text)
}

func TestParseBracketMacros_SimpleTOC(t *testing.T) {
	t.Parallel()
	result, err := ParseBracketMacros("[TOC]")
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Segments, 1)
	testutil.Equal(t, SegmentMacro, result.Segments[0].Type)
	testutil.Equal(t, "toc", result.Segments[0].Macro.Name)
}

func TestParseBracketMacros_TOCWithParams(t *testing.T) {
	t.Parallel()
	result, err := ParseBracketMacros("[TOC maxLevel=3 minLevel=1]")
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	testutil.Equal(t, "toc", macro.Name)
	testutil.Equal(t, "3", macro.Parameters["maxLevel"])
	testutil.Equal(t, "1", macro.Parameters["minLevel"])
}

func TestParseBracketMacros_PanelWithBody(t *testing.T) {
	t.Parallel()
	result, err := ParseBracketMacros("[INFO]Content here[/INFO]")
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	testutil.Equal(t, "info", macro.Name)
	testutil.Equal(t, "Content here", macro.Body)
}

func TestParseBracketMacros_PanelWithTitleAndBody(t *testing.T) {
	t.Parallel()
	result, err := ParseBracketMacros(`[WARNING title="Watch Out"]Be careful![/WARNING]`)
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	testutil.Equal(t, "warning", macro.Name)
	testutil.Equal(t, "Watch Out", macro.Parameters["title"])
	testutil.Equal(t, "Be careful!", macro.Body)
}

func TestParseBracketMacros_NestedMacros(t *testing.T) {
	t.Parallel()
	result, err := ParseBracketMacros("[INFO]Before [TOC] after[/INFO]")
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	testutil.Equal(t, "info", macro.Name)
	// TOC should be a child
	testutil.Len(t, macro.Children, 1)
	testutil.Equal(t, "toc", macro.Children[0].Name)
}

func TestParseBracketMacros_MultipleMacros(t *testing.T) {
	t.Parallel()
	result, err := ParseBracketMacros("Before [TOC] middle [INFO]content[/INFO] after")
	testutil.RequireNoError(t, err)

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
	testutil.Equal(t, 2, macroCount)
	testutil.GreaterOrEqual(t, textCount, 2)
}

func TestParseBracketMacros_UnknownMacro(t *testing.T) {
	t.Parallel()
	result, err := ParseBracketMacros("[UNKNOWN]content[/UNKNOWN]")
	testutil.RequireNoError(t, err)
	// Unknown macro should be treated as text
	testutil.GreaterOrEqual(t, len(result.Warnings), 1)
	// Content should be in text segments
	hasText := false
	for _, seg := range result.Segments {
		if seg.Type == SegmentText {
			hasText = true
		}
	}
	testutil.True(t, hasText, "unknown macro should be preserved as text")
}

func TestParseBracketMacros_MismatchedClose(t *testing.T) {
	t.Parallel()
	result, err := ParseBracketMacros("[INFO]content[/WARNING]more[/INFO]")
	testutil.RequireNoError(t, err)
	testutil.GreaterOrEqual(t, len(result.Warnings), 1)
}

func TestParseBracketMacros_UnclosedMacro(t *testing.T) {
	t.Parallel()
	result, err := ParseBracketMacros("[INFO]content without close")
	testutil.RequireNoError(t, err)
	testutil.GreaterOrEqual(t, len(result.Warnings), 1)
	// Should be treated as text
	hasText := false
	for _, seg := range result.Segments {
		if seg.Type == SegmentText {
			hasText = true
		}
	}
	testutil.True(t, hasText, "unclosed macro should be preserved as text")
}

// ==================== XML Parser Tests ====================

func TestParseConfluenceXML_EmptyInput(t *testing.T) {
	t.Parallel()
	result, err := ParseConfluenceXML("")
	testutil.RequireNoError(t, err)
	testutil.Empty(t, result.Segments)
}

func TestParseConfluenceXML_PlainHTML(t *testing.T) {
	t.Parallel()
	result, err := ParseConfluenceXML("<p>Hello world</p>")
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Segments, 1)
	testutil.Equal(t, SegmentText, result.Segments[0].Type)
	testutil.Equal(t, "<p>Hello world</p>", result.Segments[0].Text)
}

func TestParseConfluenceXML_SimpleTOC(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Segments, 1)
	testutil.Equal(t, SegmentMacro, result.Segments[0].Type)
	testutil.Equal(t, "toc", result.Segments[0].Macro.Name)
}

func TestParseConfluenceXML_TOCWithParams(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"><ac:parameter ac:name="maxLevel">3</ac:parameter><ac:parameter ac:name="minLevel">1</ac:parameter></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	testutil.Equal(t, "toc", macro.Name)
	testutil.Equal(t, "3", macro.Parameters["maxLevel"])
	testutil.Equal(t, "1", macro.Parameters["minLevel"])
}

func TestParseConfluenceXML_PanelWithBody(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><p>Content</p></ac:rich-text-body></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	testutil.Equal(t, "info", macro.Name)
	testutil.Contains(t, macro.Body, "Content")
}

func TestParseConfluenceXML_CodeWithCDATA(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:parameter ac:name="language">python</ac:parameter><ac:plain-text-body><![CDATA[print("Hello")]]></ac:plain-text-body></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	testutil.Equal(t, "code", macro.Name)
	testutil.Equal(t, "python", macro.Parameters["language"])
	testutil.Contains(t, macro.Body, `print("Hello")`)
}

func TestParseConfluenceXML_NestedMacros(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro></ac:rich-text-body></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	testutil.Equal(t, "info", macro.Name)
	testutil.Len(t, macro.Children, 1)
	testutil.Equal(t, "toc", macro.Children[0].Name)
}

func TestParseConfluenceXML_WithSurroundingHTML(t *testing.T) {
	t.Parallel()
	input := `<h1>Title</h1><ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro><p>Content</p>`
	result, err := ParseConfluenceXML(input)
	testutil.RequireNoError(t, err)

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
	testutil.Equal(t, 1, macroCount)
	testutil.Equal(t, 2, textCount)
}

// ==================== Segment Tests ====================

func TestParseResult_GetMacros(t *testing.T) {
	t.Parallel()
	result := &ParseResult{}
	result.AddTextSegment("text1")
	result.AddMacroSegment(&MacroNode{Name: "toc"})
	result.AddTextSegment("text2")
	result.AddMacroSegment(&MacroNode{Name: "info"})

	macros := result.GetMacros()
	testutil.Len(t, macros, 2)
	testutil.Equal(t, "toc", macros[0].Name)
	testutil.Equal(t, "info", macros[1].Name)
}

func TestParseResult_MergeAdjacentText(t *testing.T) {
	t.Parallel()
	result := &ParseResult{}
	result.AddTextSegment("hello ")
	result.AddTextSegment("world")

	// Should be merged into single segment
	testutil.Len(t, result.Segments, 1)
	testutil.Equal(t, "hello world", result.Segments[0].Text)
}
