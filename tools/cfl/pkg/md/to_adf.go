package md

import (
	"github.com/open-cli-collective/atlassian-go/adf"
)

// Type aliases for backward compatibility with the shared adf package.
type ADFDocument = adf.Document
type ADFNode = adf.Node
type ADFMark = adf.Mark

// ToADF converts markdown content to Atlassian Document Format (ADF) JSON.
// The returned string is a JSON-encoded ADF document.
//
// Wiki-links like [[Page Title]] are converted to standard markdown links
// before ADF conversion, producing text nodes with link marks.
// Code regions (fenced blocks, inline code) are excluded from conversion.
func ToADF(markdown []byte) (string, error) {
	// Preprocess wiki-links into standard markdown links before ADF conversion
	processed := preprocessWikiLinksForADF(markdown)
	return adf.ToJSON(processed)
}
