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
func ToADF(markdown []byte) (string, error) {
	return adf.ToJSON(markdown)
}
