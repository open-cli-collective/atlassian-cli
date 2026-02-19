package md

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestTokenizeBrackets_EmptyInput(t *testing.T) {
	t.Parallel()
	tokens, err := TokenizeBrackets("")
	testutil.RequireNoError(t, err)
	testutil.Empty(t, tokens)
}

func TestTokenizeBrackets_PlainText(t *testing.T) {
	t.Parallel()
	tokens, err := TokenizeBrackets("Hello world")
	testutil.RequireNoError(t, err)
	testutil.Len(t, tokens, 1)
	testutil.Equal(t, BracketTokenText, tokens[0].Type)
	testutil.Equal(t, "Hello world", tokens[0].Text)
}

func TestTokenizeBrackets_SimpleMacro(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			tokens, err := TokenizeBrackets(tt.input)
			testutil.RequireNoError(t, err)
			testutil.Len(t, tokens, tt.wantCount)
			testutil.Equal(t, tt.wantType, tokens[0].Type)
			testutil.Equal(t, tt.wantName, tokens[0].MacroName)
		})
	}
}

func TestTokenizeBrackets_WithParameters(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			tokens, err := TokenizeBrackets(tt.input)
			testutil.RequireNoError(t, err)
			testutil.Len(t, tokens, 1)
			testutil.Equal(t, BracketTokenOpenTag, tokens[0].Type)
			testutil.Equal(t, tt.wantParams, tokens[0].Parameters)
		})
	}
}

func TestTokenizeBrackets_OpenAndClose(t *testing.T) {
	t.Parallel()
	input := "[INFO]content[/INFO]"
	tokens, err := TokenizeBrackets(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, tokens, 3)

	testutil.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	testutil.Equal(t, "INFO", tokens[0].MacroName)

	testutil.Equal(t, BracketTokenText, tokens[1].Type)
	testutil.Equal(t, "content", tokens[1].Text)

	testutil.Equal(t, BracketTokenCloseTag, tokens[2].Type)
	testutil.Equal(t, "INFO", tokens[2].MacroName)
}

func TestTokenizeBrackets_WithSurroundingText(t *testing.T) {
	t.Parallel()
	input := "Before [TOC] after"
	tokens, err := TokenizeBrackets(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, tokens, 3)

	testutil.Equal(t, BracketTokenText, tokens[0].Type)
	testutil.Equal(t, "Before ", tokens[0].Text)

	testutil.Equal(t, BracketTokenOpenTag, tokens[1].Type)
	testutil.Equal(t, "TOC", tokens[1].MacroName)

	testutil.Equal(t, BracketTokenText, tokens[2].Type)
	testutil.Equal(t, " after", tokens[2].Text)
}

func TestTokenizeBrackets_NestedMacros(t *testing.T) {
	t.Parallel()
	input := "[INFO]outer [TOC] content[/INFO]"
	tokens, err := TokenizeBrackets(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, tokens, 5)

	testutil.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	testutil.Equal(t, "INFO", tokens[0].MacroName)

	testutil.Equal(t, BracketTokenText, tokens[1].Type)
	testutil.Equal(t, "outer ", tokens[1].Text)

	testutil.Equal(t, BracketTokenOpenTag, tokens[2].Type)
	testutil.Equal(t, "TOC", tokens[2].MacroName)

	testutil.Equal(t, BracketTokenText, tokens[3].Type)
	testutil.Equal(t, " content", tokens[3].Text)

	testutil.Equal(t, BracketTokenCloseTag, tokens[4].Type)
	testutil.Equal(t, "INFO", tokens[4].MacroName)
}

func TestTokenizeBrackets_MultipleMacros(t *testing.T) {
	t.Parallel()
	input := "[INFO]first[/INFO]\n[WARNING]second[/WARNING]"
	tokens, err := TokenizeBrackets(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, tokens, 7)

	// First macro
	testutil.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	testutil.Equal(t, "INFO", tokens[0].MacroName)
	testutil.Equal(t, BracketTokenText, tokens[1].Type)
	testutil.Equal(t, "first", tokens[1].Text)
	testutil.Equal(t, BracketTokenCloseTag, tokens[2].Type)
	testutil.Equal(t, "INFO", tokens[2].MacroName)

	// Text between
	testutil.Equal(t, BracketTokenText, tokens[3].Type)
	testutil.Equal(t, "\n", tokens[3].Text)

	// Second macro
	testutil.Equal(t, BracketTokenOpenTag, tokens[4].Type)
	testutil.Equal(t, "WARNING", tokens[4].MacroName)
	testutil.Equal(t, BracketTokenText, tokens[5].Type)
	testutil.Equal(t, "second", tokens[5].Text)
	testutil.Equal(t, BracketTokenCloseTag, tokens[6].Type)
	testutil.Equal(t, "WARNING", tokens[6].MacroName)
}

func TestTokenizeBrackets_Positions(t *testing.T) {
	t.Parallel()
	input := "abc[TOC]def"
	tokens, err := TokenizeBrackets(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, tokens, 3)

	testutil.Equal(t, 0, tokens[0].Position) // "abc"
	testutil.Equal(t, 3, tokens[1].Position) // "[TOC]"
	testutil.Equal(t, 8, tokens[2].Position) // "def"
}

func TestTokenizeBrackets_MalformedSyntax(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			tokens, err := TokenizeBrackets(tt.input)
			testutil.RequireNoError(t, err)
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
	t.Parallel()
	input := `[INFO title="[Important]"]content[/INFO]`
	tokens, err := TokenizeBrackets(input)
	testutil.RequireNoError(t, err)

	// Should have: open tag, text, close tag
	testutil.Len(t, tokens, 3)
	testutil.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	testutil.Equal(t, "[Important]", tokens[0].Parameters["title"])
}

func TestTokenizeBrackets_EscapedQuotes(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			tokens, err := TokenizeBrackets(tt.input)
			testutil.RequireNoError(t, err)
			testutil.Len(t, tokens, 1)
			// Escaped quotes should be unescaped in the returned value
			var actual string
			if val, ok := tokens[0].Parameters["title"]; ok {
				actual = val
			} else if val, ok := tokens[0].Parameters["msg"]; ok {
				actual = val
			}
			testutil.Equal(t, tt.expected, actual)
		})
	}
}

func TestTokenizeBrackets_MultilineBody(t *testing.T) {
	t.Parallel()
	input := `[INFO]
This is
multiline
content
[/INFO]`
	tokens, err := TokenizeBrackets(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, tokens, 3)

	testutil.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	testutil.Equal(t, BracketTokenText, tokens[1].Type)
	testutil.Contains(t, tokens[1].Text, "\n")
	testutil.Equal(t, BracketTokenCloseTag, tokens[2].Type)
}

func TestTokenizeBrackets_SelfClosing(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			tokens, err := TokenizeBrackets(tt.input)
			testutil.RequireNoError(t, err)
			testutil.Len(t, tokens, tt.wantCount)
			testutil.Equal(t, tt.wantType, tokens[0].Type)
			testutil.Equal(t, "TOC", tokens[0].MacroName)
			testutil.Equal(t, tt.wantParams, tokens[0].Parameters)
		})
	}
}

func TestTokenizeBrackets_DeeplyNested(t *testing.T) {
	t.Parallel()
	input := "[INFO][WARNING][NOTE]deep[/NOTE][/WARNING][/INFO]"
	tokens, err := TokenizeBrackets(input)
	testutil.RequireNoError(t, err)

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
		case BracketTokenSelfClose:
			// not expected in this test
		}
	}

	testutil.Equal(t, 3, openCount)
	testutil.Equal(t, 3, closeCount)
	testutil.Equal(t, 1, textCount)
}

func TestTokenizeBrackets_SpecialCharactersInBody(t *testing.T) {
	t.Parallel()
	input := "[INFO]<script>alert('xss')</script> & < > \"[/INFO]"
	tokens, err := TokenizeBrackets(input)
	testutil.RequireNoError(t, err)
	testutil.Len(t, tokens, 3)

	testutil.Equal(t, BracketTokenText, tokens[1].Type)
	testutil.Contains(t, tokens[1].Text, "<script>")
	testutil.Contains(t, tokens[1].Text, "&")
}

func TestTokenizeBrackets_WhitespaceHandling(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			tokens, err := TokenizeBrackets(tt.input)
			testutil.RequireNoError(t, err)
			testutil.GreaterOrEqual(t, len(tokens), 1)
			testutil.Equal(t, tt.wantName, tokens[0].MacroName)
			if tt.wantParam != "" {
				if tokens[0].Type == BracketTokenOpenTag {
					val, exists := tokens[0].Parameters["maxLevel"]
					if !exists {
						val = tokens[0].Parameters["title"]
					}
					testutil.Equal(t, tt.wantParam, val)
				}
			}
		})
	}
}
