// tokens.go defines token types for macro parsing in both directions.
package md

// BracketTokenType represents token types for bracket syntax [MACRO]...[/MACRO]
type BracketTokenType int

const (
	BracketTokenText      BracketTokenType = iota // plain text between macros
	BracketTokenOpenTag                           // [MACRO] or [MACRO params]
	BracketTokenCloseTag                          // [/MACRO]
	BracketTokenSelfClose                         // [MACRO/] (no body)
)

// BracketToken represents a single token from bracket syntax parsing.
type BracketToken struct {
	Type            BracketTokenType
	MacroName       string            // set for OpenTag, CloseTag, SelfClose (uppercase for matching)
	OriginalName    string            // set for OpenTag, CloseTag, SelfClose (original case for reconstruction)
	Parameters      map[string]string // set for OpenTag, SelfClose
	Text            string            // set for Text tokens
	Position        int               // byte offset in original input
	OriginalTagText string            // the full original bracket text for unknown macro reconstruction
}

// XMLTokenType represents token types for Confluence XML parsing.
type XMLTokenType int

const (
	XMLTokenText      XMLTokenType = iota // text/HTML between macros
	XMLTokenOpenTag                       // <ac:structured-macro ac:name="...">
	XMLTokenCloseTag                      // </ac:structured-macro>
	XMLTokenParameter                     // <ac:parameter ac:name="...">value</ac:parameter>
	XMLTokenBody                          // <ac:rich-text-body> or <ac:plain-text-body>
	XMLTokenBodyEnd                       // </ac:rich-text-body> or </ac:plain-text-body>
)

// XMLToken represents a single token from Confluence XML parsing.
type XMLToken struct {
	Type      XMLTokenType
	MacroName string // set for OpenTag
	ParamName string // set for Parameter
	Value     string // parameter value or body type ("rich-text" or "plain-text")
	Text      string // set for Text tokens
	Position  int    // byte offset in original input
}
