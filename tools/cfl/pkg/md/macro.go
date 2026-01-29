// macro.go defines the core data structures for macro parsing.
package md

import "strings"

// MacroNode represents a parsed macro in either direction (MD↔XHTML).
type MacroNode struct {
	Name       string            // "toc", "info", "warning", etc.
	Parameters map[string]string // key-value pairs from macro attributes
	Body       string            // raw content for body macros
	Children   []*MacroNode      // nested macros within body
}

// BodyType indicates how a macro's body content should be handled.
type BodyType string

const (
	BodyTypeNone      BodyType = ""           // no body (e.g., TOC)
	BodyTypeRichText  BodyType = "rich-text"  // HTML content (e.g., panels)
	BodyTypePlainText BodyType = "plain-text" // CDATA content (e.g., code)
)

// MacroType defines the behavior for a specific macro.
type MacroType struct {
	Name     string   // canonical lowercase name
	HasBody  bool     // true for panels/expand/code, false for TOC
	BodyType BodyType // how to handle body content
}

// MacroRegistry maps macro names to their type definitions.
// Adding a new macro = adding one entry here.
var MacroRegistry = map[string]MacroType{
	"toc": {
		Name:    "toc",
		HasBody: false,
	},
	"info": {
		Name:     "info",
		HasBody:  true,
		BodyType: BodyTypeRichText,
	},
	"warning": {
		Name:     "warning",
		HasBody:  true,
		BodyType: BodyTypeRichText,
	},
	"note": {
		Name:     "note",
		HasBody:  true,
		BodyType: BodyTypeRichText,
	},
	"tip": {
		Name:     "tip",
		HasBody:  true,
		BodyType: BodyTypeRichText,
	},
	"expand": {
		Name:     "expand",
		HasBody:  true,
		BodyType: BodyTypeRichText,
	},
	"code": {
		Name:     "code",
		HasBody:  true,
		BodyType: BodyTypePlainText,
	},
}

// LookupMacro returns the MacroType for a given name, normalizing to lowercase.
// Returns ok=false if macro is not registered.
func LookupMacro(name string) (MacroType, bool) {
	mt, ok := MacroRegistry[strings.ToLower(name)]
	return mt, ok
}
