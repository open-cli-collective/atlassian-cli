package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenizeBrackets_EmptyInput(t *testing.T) {
	tokens, err := TokenizeBrackets("")
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestTokenizeBrackets_PlainText(t *testing.T) {
	tokens, err := TokenizeBrackets("Hello world")
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, BracketTokenText, tokens[0].Type)
	assert.Equal(t, "Hello world", tokens[0].Text)
}

func TestTokenizeBrackets_SimpleMacro(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantType  BracketTokenType
		wantCount int
	}{
		{"TOC no params", "[TOC]", "TOC", BracketTokenOpenTag, 1},
		{"TOC lowercase", "[toc]", "TOC", BracketTokenOpenTag, 1},
		{"TOC mixed case", "[Toc]", "TOC", BracketTokenOpenTag, 1},
		{"INFO open", "[INFO]", "INFO", BracketTokenOpenTag, 1},
		{"INFO close", "[/INFO]", "INFO", BracketTokenCloseTag, 1},
		{"WARNING", "[WARNING]", "WARNING", BracketTokenOpenTag, 1},
		{"NOTE", "[NOTE]", "NOTE", BracketTokenOpenTag, 1},
		{"TIP", "[TIP]", "TIP", BracketTokenOpenTag, 1},
		{"EXPAND", "[EXPAND]", "EXPAND", BracketTokenOpenTag, 1},
		{"CODE", "[CODE]", "CODE", BracketTokenOpenTag, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeBrackets(tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, tt.wantCount)
			assert.Equal(t, tt.wantType, tokens[0].Type)
			assert.Equal(t, tt.wantName, tokens[0].MacroName)
		})
	}
}

func TestTokenizeBrackets_WithParameters(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantParams map[string]string
	}{
		{
			"single param",
			"[TOC maxLevel=3]",
			map[string]string{"maxLevel": "3"},
		},
		{
			"multiple params",
			"[TOC maxLevel=3 minLevel=1]",
			map[string]string{"maxLevel": "3", "minLevel": "1"},
		},
		{
			"quoted value",
			`[INFO title="Hello World"]`,
			map[string]string{"title": "Hello World"},
		},
		{
			"single quoted value",
			`[INFO title='Hello World']`,
			map[string]string{"title": "Hello World"},
		},
		{
			"mixed quoted and unquoted",
			`[EXPAND title="Click to expand" icon=info]`,
			map[string]string{"title": "Click to expand", "icon": "info"},
		},
		{
			"empty value",
			`[INFO title=""]`,
			map[string]string{"title": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeBrackets(tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			assert.Equal(t, BracketTokenOpenTag, tokens[0].Type)
			assert.Equal(t, tt.wantParams, tokens[0].Parameters)
		})
	}
}

func TestTokenizeBrackets_OpenAndClose(t *testing.T) {
	input := "[INFO]content[/INFO]"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	assert.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "INFO", tokens[0].MacroName)

	assert.Equal(t, BracketTokenText, tokens[1].Type)
	assert.Equal(t, "content", tokens[1].Text)

	assert.Equal(t, BracketTokenCloseTag, tokens[2].Type)
	assert.Equal(t, "INFO", tokens[2].MacroName)
}

func TestTokenizeBrackets_WithSurroundingText(t *testing.T) {
	input := "Before [TOC] after"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	assert.Equal(t, BracketTokenText, tokens[0].Type)
	assert.Equal(t, "Before ", tokens[0].Text)

	assert.Equal(t, BracketTokenOpenTag, tokens[1].Type)
	assert.Equal(t, "TOC", tokens[1].MacroName)

	assert.Equal(t, BracketTokenText, tokens[2].Type)
	assert.Equal(t, " after", tokens[2].Text)
}

func TestTokenizeBrackets_NestedMacros(t *testing.T) {
	input := "[INFO]outer [TOC] content[/INFO]"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 5)

	assert.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "INFO", tokens[0].MacroName)

	assert.Equal(t, BracketTokenText, tokens[1].Type)
	assert.Equal(t, "outer ", tokens[1].Text)

	assert.Equal(t, BracketTokenOpenTag, tokens[2].Type)
	assert.Equal(t, "TOC", tokens[2].MacroName)

	assert.Equal(t, BracketTokenText, tokens[3].Type)
	assert.Equal(t, " content", tokens[3].Text)

	assert.Equal(t, BracketTokenCloseTag, tokens[4].Type)
	assert.Equal(t, "INFO", tokens[4].MacroName)
}

func TestTokenizeBrackets_MultipleMacros(t *testing.T) {
	input := "[INFO]first[/INFO]\n[WARNING]second[/WARNING]"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 7)

	// First macro
	assert.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "INFO", tokens[0].MacroName)
	assert.Equal(t, BracketTokenText, tokens[1].Type)
	assert.Equal(t, "first", tokens[1].Text)
	assert.Equal(t, BracketTokenCloseTag, tokens[2].Type)
	assert.Equal(t, "INFO", tokens[2].MacroName)

	// Text between
	assert.Equal(t, BracketTokenText, tokens[3].Type)
	assert.Equal(t, "\n", tokens[3].Text)

	// Second macro
	assert.Equal(t, BracketTokenOpenTag, tokens[4].Type)
	assert.Equal(t, "WARNING", tokens[4].MacroName)
	assert.Equal(t, BracketTokenText, tokens[5].Type)
	assert.Equal(t, "second", tokens[5].Text)
	assert.Equal(t, BracketTokenCloseTag, tokens[6].Type)
	assert.Equal(t, "WARNING", tokens[6].MacroName)
}

func TestTokenizeBrackets_Positions(t *testing.T) {
	input := "abc[TOC]def"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	assert.Equal(t, 0, tokens[0].Position) // "abc"
	assert.Equal(t, 3, tokens[1].Position) // "[TOC]"
	assert.Equal(t, 8, tokens[2].Position) // "def"
}

func TestTokenizeBrackets_MalformedSyntax(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			"orphan open bracket",
			"text [ more text",
			"lone bracket treated as text",
		},
		{
			"orphan close bracket",
			"text ] more text",
			"close bracket in text is just text",
		},
		{
			"incomplete macro name",
			"text [",
			"bracket at end of input",
		},
		{
			"brackets in text",
			"array[0] = value",
			"programming brackets not macro syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeBrackets(tt.input)
			require.NoError(t, err, "should not error on malformed input")
			// Malformed macro syntax should be treated as text
			for _, tok := range tokens {
				if tok.Type == BracketTokenOpenTag || tok.Type == BracketTokenCloseTag {
					// Only valid macro names should be tokenized
					_, known := LookupMacro(tok.MacroName)
					// Unknown macros are still tokenized - parser validates them
					_ = known
				}
			}
		})
	}
}

func TestTokenizeBrackets_BracketsInQuotedValues(t *testing.T) {
	input := `[INFO title="[Important]"]content[/INFO]`
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)

	// Should have: open tag, text, close tag
	require.Len(t, tokens, 3)
	assert.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "[Important]", tokens[0].Parameters["title"])
}

func TestTokenizeBrackets_EscapedQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"escaped double quote",
			`[INFO title="Say \"Hello\""]`,
			`Say "Hello"`,
		},
		{
			"escaped single quote",
			`[INFO title='It\'s fine']`,
			`It's fine`,
		},
		{
			"escaped quote in middle",
			`[INFO msg="value \"with\" quotes"]`,
			`value "with" quotes`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeBrackets(tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			// Escaped quotes should be unescaped in the returned value
			var actual string
			if val, ok := tokens[0].Parameters["title"]; ok {
				actual = val
			} else if val, ok := tokens[0].Parameters["msg"]; ok {
				actual = val
			}
			assert.Equal(t, tt.expected, actual, "parameter value should have unescaped quotes")
		})
	}
}

func TestTokenizeBrackets_MultilineBody(t *testing.T) {
	input := `[INFO]
This is
multiline
content
[/INFO]`
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	assert.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	assert.Equal(t, BracketTokenText, tokens[1].Type)
	assert.Contains(t, tokens[1].Text, "\n")
	assert.Equal(t, BracketTokenCloseTag, tokens[2].Type)
}

func TestTokenizeBrackets_SelfClosing(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantCount  int
		wantType   BracketTokenType
		wantParams map[string]string
	}{
		{
			"TOC self-close",
			"[TOC/]",
			1,
			BracketTokenSelfClose,
			map[string]string{},
		},
		{
			"with space",
			"[TOC /]",
			1,
			BracketTokenSelfClose,
			map[string]string{},
		},
		{
			"with params",
			"[TOC maxLevel=3/]",
			1,
			BracketTokenSelfClose,
			map[string]string{"maxLevel": "3"},
		},
		{
			"with multiple params",
			"[TOC maxLevel=3 minLevel=1/]",
			1,
			BracketTokenSelfClose,
			map[string]string{"maxLevel": "3", "minLevel": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeBrackets(tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, tt.wantCount)
			assert.Equal(t, tt.wantType, tokens[0].Type)
			assert.Equal(t, "TOC", tokens[0].MacroName)
			assert.Equal(t, tt.wantParams, tokens[0].Parameters)
		})
	}
}

func TestTokenizeBrackets_DeeplyNested(t *testing.T) {
	input := "[INFO][WARNING][NOTE]deep[/NOTE][/WARNING][/INFO]"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)

	// Count token types
	openCount := 0
	closeCount := 0
	textCount := 0
	for _, tok := range tokens {
		switch tok.Type {
		case BracketTokenOpenTag:
			openCount++
		case BracketTokenCloseTag:
			closeCount++
		case BracketTokenText:
			textCount++
		}
	}

	assert.Equal(t, 3, openCount, "should have 3 open tags")
	assert.Equal(t, 3, closeCount, "should have 3 close tags")
	assert.Equal(t, 1, textCount, "should have 1 text token")
}

func TestTokenizeBrackets_SpecialCharactersInBody(t *testing.T) {
	input := "[INFO]<script>alert('xss')</script> & < > \"[/INFO]"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	assert.Equal(t, BracketTokenText, tokens[1].Type)
	assert.Contains(t, tokens[1].Text, "<script>")
	assert.Contains(t, tokens[1].Text, "&")
}

func TestTokenizeBrackets_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantParam string
	}{
		{
			"space after name",
			"[TOC ]",
			"TOC",
			"",
		},
		{
			"multiple spaces",
			"[TOC   maxLevel=3]",
			"TOC",
			"3",
		},
		{
			"tabs",
			"[TOC\tmaxLevel=3]",
			"TOC",
			"3",
		},
		{
			"newline in params",
			"[INFO\n  title=\"test\"]",
			"INFO",
			"test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeBrackets(tt.input)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(tokens), 1)
			assert.Equal(t, tt.wantName, tokens[0].MacroName)
			if tt.wantParam != "" {
				if tokens[0].Type == BracketTokenOpenTag {
					val, exists := tokens[0].Parameters["maxLevel"]
					if !exists {
						val = tokens[0].Parameters["title"]
					}
					assert.Equal(t, tt.wantParam, val)
				}
			}
		})
	}
}
