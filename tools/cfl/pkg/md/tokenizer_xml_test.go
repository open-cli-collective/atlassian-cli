package md

import (
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestTokenizeConfluenceXML_EmptyInput(t *testing.T) {
	t.Parallel()
	tokens, err := TokenizeConfluenceXML("")
	testutil.RequireNoError(t, err)
	testutil.Empty(t, tokens)
}

func TestTokenizeConfluenceXML_PlainHTML(t *testing.T) {
	t.Parallel()
	input := "<p>Hello world</p>"
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, tokens, 1)
	testutil.Equal(t, XMLTokenText, tokens[0].Type)
	testutil.Equal(t, input, tokens[0].Text)
}

func TestTokenizeConfluenceXML_SimpleMacro(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, tokens, 2)

	testutil.Equal(t, XMLTokenOpenTag, tokens[0].Type)
	testutil.Equal(t, "toc", tokens[0].MacroName)

	testutil.Equal(t, XMLTokenCloseTag, tokens[1].Type)
}

func TestTokenizeConfluenceXML_MacroWithParameter(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"><ac:parameter ac:name="maxLevel">3</ac:parameter></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, tokens, 3)

	testutil.Equal(t, XMLTokenOpenTag, tokens[0].Type)
	testutil.Equal(t, "toc", tokens[0].MacroName)

	testutil.Equal(t, XMLTokenParameter, tokens[1].Type)
	testutil.Equal(t, "maxLevel", tokens[1].ParamName)
	testutil.Equal(t, "3", tokens[1].Value)

	testutil.Equal(t, XMLTokenCloseTag, tokens[2].Type)
}

func TestTokenizeConfluenceXML_MacroWithMultipleParameters(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"><ac:parameter ac:name="maxLevel">3</ac:parameter><ac:parameter ac:name="minLevel">1</ac:parameter><ac:parameter ac:name="type">flat</ac:parameter></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	// Open, 3 params, close = 5 tokens
	testutil.Len(t, tokens, 5)

	testutil.Equal(t, XMLTokenParameter, tokens[1].Type)
	testutil.Equal(t, "maxLevel", tokens[1].ParamName)
	testutil.Equal(t, "3", tokens[1].Value)

	testutil.Equal(t, XMLTokenParameter, tokens[2].Type)
	testutil.Equal(t, "minLevel", tokens[2].ParamName)
	testutil.Equal(t, "1", tokens[2].Value)

	testutil.Equal(t, XMLTokenParameter, tokens[3].Type)
	testutil.Equal(t, "type", tokens[3].ParamName)
	testutil.Equal(t, "flat", tokens[3].Value)
}

func TestTokenizeConfluenceXML_PanelWithRichTextBody(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><p>Content</p></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	// Open, body open, text, body close, close = 5 tokens
	testutil.Len(t, tokens, 5)

	testutil.Equal(t, XMLTokenOpenTag, tokens[0].Type)
	testutil.Equal(t, "info", tokens[0].MacroName)

	testutil.Equal(t, XMLTokenBody, tokens[1].Type)
	testutil.Equal(t, "rich-text", tokens[1].Value)

	testutil.Equal(t, XMLTokenText, tokens[2].Type)
	testutil.Equal(t, "<p>Content</p>", tokens[2].Text)

	testutil.Equal(t, XMLTokenBodyEnd, tokens[3].Type)
	testutil.Equal(t, "rich-text", tokens[3].Value)

	testutil.Equal(t, XMLTokenCloseTag, tokens[4].Type)
}

func TestTokenizeConfluenceXML_PanelWithTitleAndBody(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="warning" ac:schema-version="1"><ac:parameter ac:name="title">Watch Out</ac:parameter><ac:rich-text-body><p>Warning content</p></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	// Open, param, body open, text, body close, close = 6 tokens
	testutil.Len(t, tokens, 6)

	testutil.Equal(t, "warning", tokens[0].MacroName)
	testutil.Equal(t, "title", tokens[1].ParamName)
	testutil.Equal(t, "Watch Out", tokens[1].Value)
	testutil.Equal(t, XMLTokenBody, tokens[2].Type)
	testutil.Contains(t, tokens[3].Text, "Warning content")
}

func TestTokenizeConfluenceXML_CodeMacroWithCDATA(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:parameter ac:name="language">python</ac:parameter><ac:plain-text-body><![CDATA[print("Hello")]]></ac:plain-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	// Open, param, body open, text (CDATA content), body close, close = 6 tokens
	testutil.Len(t, tokens, 6)

	testutil.Equal(t, "code", tokens[0].MacroName)
	testutil.Equal(t, "language", tokens[1].ParamName)
	testutil.Equal(t, "python", tokens[1].Value)

	testutil.Equal(t, XMLTokenBody, tokens[2].Type)
	testutil.Equal(t, "plain-text", tokens[2].Value)

	testutil.Equal(t, XMLTokenText, tokens[3].Type)
	testutil.Equal(t, `print("Hello")`, tokens[3].Text)

	testutil.Equal(t, XMLTokenBodyEnd, tokens[4].Type)
	testutil.Equal(t, "plain-text", tokens[4].Value)
}

func TestTokenizeConfluenceXML_NestedMacros(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><p>Before</p><ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro><p>After</p></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	// Count token types
	openCount := 0
	closeCount := 0
	for _, tok := range tokens {
		if tok.Type == XMLTokenOpenTag {
			openCount++
		}
		if tok.Type == XMLTokenCloseTag {
			closeCount++
		}
	}

	testutil.Equal(t, 2, openCount)
	testutil.Equal(t, 2, closeCount)
}

func TestTokenizeConfluenceXML_WithSurroundingHTML(t *testing.T) {
	t.Parallel()
	input := `<h1>Title</h1><ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro><p>Content</p>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	// text, open, close, text = 4 tokens
	testutil.Len(t, tokens, 4)

	testutil.Equal(t, XMLTokenText, tokens[0].Type)
	testutil.Equal(t, "<h1>Title</h1>", tokens[0].Text)

	testutil.Equal(t, XMLTokenOpenTag, tokens[1].Type)
	testutil.Equal(t, "toc", tokens[1].MacroName)

	testutil.Equal(t, XMLTokenCloseTag, tokens[2].Type)

	testutil.Equal(t, XMLTokenText, tokens[3].Type)
	testutil.Equal(t, "<p>Content</p>", tokens[3].Text)
}

func TestTokenizeConfluenceXML_AllPanelTypes(t *testing.T) {
	t.Parallel()
	panelTypes := []string{"info", "warning", "note", "tip", "expand"}

	for _, pt := range panelTypes {
		t.Run(pt, func(t *testing.T) {
			t.Parallel()
			input := `<ac:structured-macro ac:name="` + pt + `" ac:schema-version="1"><ac:rich-text-body><p>Content</p></ac:rich-text-body></ac:structured-macro>`
			tokens, err := TokenizeConfluenceXML(input)
			testutil.RequireNoError(t, err)
			testutil.GreaterOrEqual(t, len(tokens), 2)
			testutil.Equal(t, pt, tokens[0].MacroName)
		})
	}
}

func TestTokenizeConfluenceXML_Positions(t *testing.T) {
	t.Parallel()
	input := `abc<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>def`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, tokens, 4)

	testutil.Equal(t, 0, tokens[0].Position) // "abc"
	testutil.Equal(t, 3, tokens[1].Position) // macro open
	// Close and "def" positions will follow
}

func TestTokenizeConfluenceXML_CDATAWithSpecialChars(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:plain-text-body><![CDATA[if x < 10 && y > 5 {
    fmt.Println("test")
}]]></ac:plain-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	// Find the CDATA content token
	var cdataToken *XMLToken
	for i := range tokens {
		if tokens[i].Type == XMLTokenText && strings.Contains(tokens[i].Text, "x < 10") {
			cdataToken = &tokens[i]
			break
		}
	}

	testutil.NotNil(t, cdataToken)
	testutil.Contains(t, cdataToken.Text, "x < 10")
	testutil.Contains(t, cdataToken.Text, "&&")
	testutil.Contains(t, cdataToken.Text, "y > 5")
}

func TestTokenizeConfluenceXML_MultilineCDATA(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:plain-text-body><![CDATA[
line1
line2
line3
]]></ac:plain-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	// Find CDATA content
	var found bool
	for _, tok := range tokens {
		if tok.Type == XMLTokenText && strings.Contains(tok.Text, "line1") {
			found = true
			testutil.Contains(t, tok.Text, "line2")
			testutil.Contains(t, tok.Text, "line3")
			testutil.Contains(t, tok.Text, "\n")
		}
	}
	testutil.True(t, found, "should find multiline CDATA content")
}

func TestTokenizeConfluenceXML_DeeplyNestedMacros(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><ac:structured-macro ac:name="warning" ac:schema-version="1"><ac:rich-text-body><ac:structured-macro ac:name="note" ac:schema-version="1"><ac:rich-text-body><p>Deep</p></ac:rich-text-body></ac:structured-macro></ac:rich-text-body></ac:structured-macro></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	// Count opens and closes
	openCount := 0
	closeCount := 0
	macroNames := []string{}
	for _, tok := range tokens {
		if tok.Type == XMLTokenOpenTag {
			openCount++
			macroNames = append(macroNames, tok.MacroName)
		}
		if tok.Type == XMLTokenCloseTag {
			closeCount++
		}
	}

	testutil.Equal(t, 3, openCount)
	testutil.Equal(t, 3, closeCount)
	testutil.Contains(t, strings.Join(macroNames, ","), "info")
	testutil.Contains(t, strings.Join(macroNames, ","), "warning")
	testutil.Contains(t, strings.Join(macroNames, ","), "note")
}

func TestTokenizeConfluenceXML_WhitespaceInMacro(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1">
    <ac:parameter ac:name="maxLevel">3</ac:parameter>
</ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	// Should still find open, param, close (whitespace becomes text tokens)
	var foundParam bool
	for _, tok := range tokens {
		if tok.Type == XMLTokenParameter && tok.ParamName == "maxLevel" {
			foundParam = true
			testutil.Equal(t, "3", tok.Value)
		}
	}
	testutil.True(t, foundParam, "should find maxLevel parameter")
}

func TestTokenizeConfluenceXML_EmptyParameter(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"><ac:parameter ac:name="title"></ac:parameter></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	var foundParam bool
	for _, tok := range tokens {
		if tok.Type == XMLTokenParameter && tok.ParamName == "title" {
			foundParam = true
			testutil.Equal(t, "", tok.Value)
		}
	}
	testutil.True(t, foundParam, "should find empty parameter")
}

func TestTokenizeConfluenceXML_EmptyRichTextBody(t *testing.T) {
	t.Parallel()
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	// Open, body open, body close, close = 4 tokens (no text between body tags)
	bodyOpenCount := 0
	bodyCloseCount := 0
	for _, tok := range tokens {
		if tok.Type == XMLTokenBody {
			bodyOpenCount++
		}
		if tok.Type == XMLTokenBodyEnd {
			bodyCloseCount++
		}
	}
	testutil.Equal(t, 1, bodyOpenCount)
	testutil.Equal(t, 1, bodyCloseCount)
}

func TestTokenizeConfluenceXML_MacroNameCaseInsensitive(t *testing.T) {
	t.Parallel()
	inputs := []string{
		`<ac:structured-macro ac:name="TOC" ac:schema-version="1"></ac:structured-macro>`,
		`<ac:structured-macro ac:name="Toc" ac:schema-version="1"></ac:structured-macro>`,
		`<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>`,
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			tokens, err := TokenizeConfluenceXML(input)
			testutil.RequireNoError(t, err)
			testutil.GreaterOrEqual(t, len(tokens), 1)
			// All should normalize to lowercase
			testutil.Equal(t, "toc", tokens[0].MacroName)
		})
	}
}

func TestExtractCDATAContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"<![CDATA[hello]]>", "hello"},
		{"<![CDATA[multi\nline]]>", "multi\nline"},
		{"<![CDATA[x < 10 && y > 5]]>", "x < 10 && y > 5"},
		{"<![CDATA[]]>", ""},
		{"not cdata", "not cdata"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := ExtractCDATAContent(tt.input)
			testutil.Equal(t, tt.expected, result)
		})
	}
}

// Tests for self-closing macros (issue #56)
func TestTokenizeConfluenceXML_SelfClosingMacro(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          string
		expectedOpens  int
		expectedCloses int
		expectedMacros []string
	}{
		{
			name:           "simple self-closing macro with space",
			input:          `<ac:structured-macro ac:name="toc" ac:schema-version="1" />`,
			expectedOpens:  1,
			expectedCloses: 1,
			expectedMacros: []string{"toc"},
		},
		{
			name:           "simple self-closing macro without space",
			input:          `<ac:structured-macro ac:name="toc" ac:schema-version="1"/>`,
			expectedOpens:  1,
			expectedCloses: 1,
			expectedMacros: []string{"toc"},
		},
		{
			name:           "self-closing macro in p tag",
			input:          `<p><ac:structured-macro ac:name="toc" ac:schema-version="1" /></p>`,
			expectedOpens:  1,
			expectedCloses: 1,
			expectedMacros: []string{"toc"},
		},
		{
			name:           "multiple self-closing macros",
			input:          `<ac:structured-macro ac:name="toc" /><ac:structured-macro ac:name="anchor" ac:schema-version="1" />`,
			expectedOpens:  2,
			expectedCloses: 2,
			expectedMacros: []string{"toc", "anchor"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tokens, err := TokenizeConfluenceXML(tt.input)
			testutil.RequireNoError(t, err)

			openCount := 0
			closeCount := 0
			macroNames := []string{}
			for _, tok := range tokens {
				if tok.Type == XMLTokenOpenTag {
					openCount++
					macroNames = append(macroNames, tok.MacroName)
				}
				if tok.Type == XMLTokenCloseTag {
					closeCount++
				}
			}

			testutil.Equal(t, tt.expectedOpens, openCount)
			testutil.Equal(t, tt.expectedCloses, closeCount)
			for _, expected := range tt.expectedMacros {
				testutil.Contains(t, strings.Join(macroNames, ","), expected)
			}
		})
	}
}

func TestTokenizeConfluenceXML_SelfClosingNestedInBodyMacro(t *testing.T) {
	t.Parallel()
	// This is the exact scenario from issue #56
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><p><ac:structured-macro ac:name="toc" ac:schema-version="1" /></p></ac:rich-text-body></ac:structured-macro>`

	tokens, err := TokenizeConfluenceXML(input)
	testutil.RequireNoError(t, err)

	// Expected token sequence:
	// 1. XMLTokenOpenTag (info)
	// 2. XMLTokenBody (rich-text)
	// 3. XMLTokenText (<p>)
	// 4. XMLTokenOpenTag (toc) - from self-closing
	// 5. XMLTokenCloseTag - from self-closing
	// 6. XMLTokenText (</p>)
	// 7. XMLTokenBodyEnd
	// 8. XMLTokenCloseTag (info)

	openCount := 0
	closeCount := 0
	macroNames := []string{}
	for _, tok := range tokens {
		if tok.Type == XMLTokenOpenTag {
			openCount++
			macroNames = append(macroNames, tok.MacroName)
		}
		if tok.Type == XMLTokenCloseTag {
			closeCount++
		}
	}

	testutil.Equal(t, 2, openCount)
	testutil.Equal(t, 2, closeCount)
	testutil.Contains(t, strings.Join(macroNames, ","), "info")
	testutil.Contains(t, strings.Join(macroNames, ","), "toc")

	// Verify token order: info open should come before toc open
	var infoIdx, tocIdx int
	for i, tok := range tokens {
		if tok.Type == XMLTokenOpenTag && tok.MacroName == "info" {
			infoIdx = i
		}
		if tok.Type == XMLTokenOpenTag && tok.MacroName == "toc" {
			tocIdx = i
		}
	}
	testutil.True(t, infoIdx < tocIdx, "info should open before toc")
}

func TestTokenizeConfluenceXML_SelfClosingVsRegularMacro(t *testing.T) {
	t.Parallel()
	// Make sure regular macros still work and are distinguished from self-closing
	regular := `<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>`
	selfClosing := `<ac:structured-macro ac:name="toc" ac:schema-version="1" />`

	regularTokens, err := TokenizeConfluenceXML(regular)
	testutil.RequireNoError(t, err)

	selfClosingTokens, err := TokenizeConfluenceXML(selfClosing)
	testutil.RequireNoError(t, err)

	// Both should have 1 open and 1 close
	countTokens := func(tokens []XMLToken) (opens, closes int) {
		for _, tok := range tokens {
			if tok.Type == XMLTokenOpenTag {
				opens++
			}
			if tok.Type == XMLTokenCloseTag {
				closes++
			}
		}
		return
	}

	regularOpens, regularCloses := countTokens(regularTokens)
	selfClosingOpens, selfClosingCloses := countTokens(selfClosingTokens)

	testutil.Equal(t, 1, regularOpens)
	testutil.Equal(t, 1, regularCloses)
	testutil.Equal(t, 1, selfClosingOpens)
	testutil.Equal(t, 1, selfClosingCloses)
}
