package md

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenizeConfluenceXML_EmptyInput(t *testing.T) {
	tokens, err := TokenizeConfluenceXML("")
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestTokenizeConfluenceXML_PlainHTML(t *testing.T) {
	input := "<p>Hello world</p>"
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, XMLTokenText, tokens[0].Type)
	assert.Equal(t, input, tokens[0].Text)
}

func TestTokenizeConfluenceXML_SimpleMacro(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, tokens, 2)

	assert.Equal(t, XMLTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "toc", tokens[0].MacroName)

	assert.Equal(t, XMLTokenCloseTag, tokens[1].Type)
}

func TestTokenizeConfluenceXML_MacroWithParameter(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"><ac:parameter ac:name="maxLevel">3</ac:parameter></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	assert.Equal(t, XMLTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "toc", tokens[0].MacroName)

	assert.Equal(t, XMLTokenParameter, tokens[1].Type)
	assert.Equal(t, "maxLevel", tokens[1].ParamName)
	assert.Equal(t, "3", tokens[1].Value)

	assert.Equal(t, XMLTokenCloseTag, tokens[2].Type)
}

func TestTokenizeConfluenceXML_MacroWithMultipleParameters(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"><ac:parameter ac:name="maxLevel">3</ac:parameter><ac:parameter ac:name="minLevel">1</ac:parameter><ac:parameter ac:name="type">flat</ac:parameter></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Open, 3 params, close = 5 tokens
	require.Len(t, tokens, 5)

	assert.Equal(t, XMLTokenParameter, tokens[1].Type)
	assert.Equal(t, "maxLevel", tokens[1].ParamName)
	assert.Equal(t, "3", tokens[1].Value)

	assert.Equal(t, XMLTokenParameter, tokens[2].Type)
	assert.Equal(t, "minLevel", tokens[2].ParamName)
	assert.Equal(t, "1", tokens[2].Value)

	assert.Equal(t, XMLTokenParameter, tokens[3].Type)
	assert.Equal(t, "type", tokens[3].ParamName)
	assert.Equal(t, "flat", tokens[3].Value)
}

func TestTokenizeConfluenceXML_PanelWithRichTextBody(t *testing.T) {
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><p>Content</p></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Open, body open, text, body close, close = 5 tokens
	require.Len(t, tokens, 5)

	assert.Equal(t, XMLTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "info", tokens[0].MacroName)

	assert.Equal(t, XMLTokenBody, tokens[1].Type)
	assert.Equal(t, "rich-text", tokens[1].Value)

	assert.Equal(t, XMLTokenText, tokens[2].Type)
	assert.Equal(t, "<p>Content</p>", tokens[2].Text)

	assert.Equal(t, XMLTokenBodyEnd, tokens[3].Type)
	assert.Equal(t, "rich-text", tokens[3].Value)

	assert.Equal(t, XMLTokenCloseTag, tokens[4].Type)
}

func TestTokenizeConfluenceXML_PanelWithTitleAndBody(t *testing.T) {
	input := `<ac:structured-macro ac:name="warning" ac:schema-version="1"><ac:parameter ac:name="title">Watch Out</ac:parameter><ac:rich-text-body><p>Warning content</p></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Open, param, body open, text, body close, close = 6 tokens
	require.Len(t, tokens, 6)

	assert.Equal(t, "warning", tokens[0].MacroName)
	assert.Equal(t, "title", tokens[1].ParamName)
	assert.Equal(t, "Watch Out", tokens[1].Value)
	assert.Equal(t, XMLTokenBody, tokens[2].Type)
	assert.Contains(t, tokens[3].Text, "Warning content")
}

func TestTokenizeConfluenceXML_CodeMacroWithCDATA(t *testing.T) {
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:parameter ac:name="language">python</ac:parameter><ac:plain-text-body><![CDATA[print("Hello")]]></ac:plain-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Open, param, body open, text (CDATA content), body close, close = 6 tokens
	require.Len(t, tokens, 6)

	assert.Equal(t, "code", tokens[0].MacroName)
	assert.Equal(t, "language", tokens[1].ParamName)
	assert.Equal(t, "python", tokens[1].Value)

	assert.Equal(t, XMLTokenBody, tokens[2].Type)
	assert.Equal(t, "plain-text", tokens[2].Value)

	assert.Equal(t, XMLTokenText, tokens[3].Type)
	assert.Equal(t, `print("Hello")`, tokens[3].Text)

	assert.Equal(t, XMLTokenBodyEnd, tokens[4].Type)
	assert.Equal(t, "plain-text", tokens[4].Value)
}

func TestTokenizeConfluenceXML_NestedMacros(t *testing.T) {
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><p>Before</p><ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro><p>After</p></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

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

	assert.Equal(t, 2, openCount, "should have 2 macro opens (info and toc)")
	assert.Equal(t, 2, closeCount, "should have 2 macro closes")
}

func TestTokenizeConfluenceXML_WithSurroundingHTML(t *testing.T) {
	input := `<h1>Title</h1><ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro><p>Content</p>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// text, open, close, text = 4 tokens
	require.Len(t, tokens, 4)

	assert.Equal(t, XMLTokenText, tokens[0].Type)
	assert.Equal(t, "<h1>Title</h1>", tokens[0].Text)

	assert.Equal(t, XMLTokenOpenTag, tokens[1].Type)
	assert.Equal(t, "toc", tokens[1].MacroName)

	assert.Equal(t, XMLTokenCloseTag, tokens[2].Type)

	assert.Equal(t, XMLTokenText, tokens[3].Type)
	assert.Equal(t, "<p>Content</p>", tokens[3].Text)
}

func TestTokenizeConfluenceXML_AllPanelTypes(t *testing.T) {
	panelTypes := []string{"info", "warning", "note", "tip", "expand"}

	for _, pt := range panelTypes {
		t.Run(pt, func(t *testing.T) {
			input := `<ac:structured-macro ac:name="` + pt + `" ac:schema-version="1"><ac:rich-text-body><p>Content</p></ac:rich-text-body></ac:structured-macro>`
			tokens, err := TokenizeConfluenceXML(input)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(tokens), 2)
			assert.Equal(t, pt, tokens[0].MacroName)
		})
	}
}

func TestTokenizeConfluenceXML_Positions(t *testing.T) {
	input := `abc<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>def`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, tokens, 4)

	assert.Equal(t, 0, tokens[0].Position) // "abc"
	assert.Equal(t, 3, tokens[1].Position) // macro open
	// Close and "def" positions will follow
}

func TestTokenizeConfluenceXML_CDATAWithSpecialChars(t *testing.T) {
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:plain-text-body><![CDATA[if x < 10 && y > 5 {
    fmt.Println("test")
}]]></ac:plain-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Find the CDATA content token
	var cdataToken *XMLToken
	for i := range tokens {
		if tokens[i].Type == XMLTokenText && strings.Contains(tokens[i].Text, "x < 10") {
			cdataToken = &tokens[i]
			break
		}
	}

	require.NotNil(t, cdataToken, "should find CDATA content")
	assert.Contains(t, cdataToken.Text, "x < 10")
	assert.Contains(t, cdataToken.Text, "&&")
	assert.Contains(t, cdataToken.Text, "y > 5")
}

func TestTokenizeConfluenceXML_MultilineCDATA(t *testing.T) {
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:plain-text-body><![CDATA[
line1
line2
line3
]]></ac:plain-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Find CDATA content
	var found bool
	for _, tok := range tokens {
		if tok.Type == XMLTokenText && strings.Contains(tok.Text, "line1") {
			found = true
			assert.Contains(t, tok.Text, "line2")
			assert.Contains(t, tok.Text, "line3")
			assert.Contains(t, tok.Text, "\n")
		}
	}
	assert.True(t, found, "should find multiline CDATA content")
}

func TestTokenizeConfluenceXML_DeeplyNestedMacros(t *testing.T) {
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><ac:structured-macro ac:name="warning" ac:schema-version="1"><ac:rich-text-body><ac:structured-macro ac:name="note" ac:schema-version="1"><ac:rich-text-body><p>Deep</p></ac:rich-text-body></ac:structured-macro></ac:rich-text-body></ac:structured-macro></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

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

	assert.Equal(t, 3, openCount, "should have 3 macro opens")
	assert.Equal(t, 3, closeCount, "should have 3 macro closes")
	assert.Contains(t, macroNames, "info")
	assert.Contains(t, macroNames, "warning")
	assert.Contains(t, macroNames, "note")
}

func TestTokenizeConfluenceXML_WhitespaceInMacro(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1">
    <ac:parameter ac:name="maxLevel">3</ac:parameter>
</ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Should still find open, param, close (whitespace becomes text tokens)
	var foundParam bool
	for _, tok := range tokens {
		if tok.Type == XMLTokenParameter && tok.ParamName == "maxLevel" {
			foundParam = true
			assert.Equal(t, "3", tok.Value)
		}
	}
	assert.True(t, foundParam, "should find maxLevel parameter")
}

func TestTokenizeConfluenceXML_EmptyParameter(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"><ac:parameter ac:name="title"></ac:parameter></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	var foundParam bool
	for _, tok := range tokens {
		if tok.Type == XMLTokenParameter && tok.ParamName == "title" {
			foundParam = true
			assert.Equal(t, "", tok.Value)
		}
	}
	assert.True(t, foundParam, "should find empty parameter")
}

func TestTokenizeConfluenceXML_EmptyRichTextBody(t *testing.T) {
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

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
	assert.Equal(t, 1, bodyOpenCount)
	assert.Equal(t, 1, bodyCloseCount)
}

func TestTokenizeConfluenceXML_MacroNameCaseInsensitive(t *testing.T) {
	inputs := []string{
		`<ac:structured-macro ac:name="TOC" ac:schema-version="1"></ac:structured-macro>`,
		`<ac:structured-macro ac:name="Toc" ac:schema-version="1"></ac:structured-macro>`,
		`<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>`,
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			tokens, err := TokenizeConfluenceXML(input)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(tokens), 1)
			// All should normalize to lowercase
			assert.Equal(t, "toc", tokens[0].MacroName)
		})
	}
}

func TestExtractCDATAContent(t *testing.T) {
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
			result := ExtractCDATAContent(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Tests for self-closing macros (issue #56)
func TestTokenizeConfluenceXML_SelfClosingMacro(t *testing.T) {
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
			tokens, err := TokenizeConfluenceXML(tt.input)
			require.NoError(t, err)

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

			assert.Equal(t, tt.expectedOpens, openCount, "open tag count mismatch")
			assert.Equal(t, tt.expectedCloses, closeCount, "close tag count mismatch")
			for _, expected := range tt.expectedMacros {
				assert.Contains(t, macroNames, expected, "expected macro %s not found", expected)
			}
		})
	}
}

func TestTokenizeConfluenceXML_SelfClosingNestedInBodyMacro(t *testing.T) {
	// This is the exact scenario from issue #56
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><p><ac:structured-macro ac:name="toc" ac:schema-version="1" /></p></ac:rich-text-body></ac:structured-macro>`

	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

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

	assert.Equal(t, 2, openCount, "should have 2 macro opens (info + toc)")
	assert.Equal(t, 2, closeCount, "should have 2 macro closes (info + toc)")
	assert.Contains(t, macroNames, "info")
	assert.Contains(t, macroNames, "toc")

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
	assert.Less(t, infoIdx, tocIdx, "info should open before toc")
}

func TestTokenizeConfluenceXML_SelfClosingVsRegularMacro(t *testing.T) {
	// Make sure regular macros still work and are distinguished from self-closing
	regular := `<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>`
	selfClosing := `<ac:structured-macro ac:name="toc" ac:schema-version="1" />`

	regularTokens, err := TokenizeConfluenceXML(regular)
	require.NoError(t, err)

	selfClosingTokens, err := TokenizeConfluenceXML(selfClosing)
	require.NoError(t, err)

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

	assert.Equal(t, 1, regularOpens)
	assert.Equal(t, 1, regularCloses)
	assert.Equal(t, 1, selfClosingOpens)
	assert.Equal(t, 1, selfClosingCloses)
}
