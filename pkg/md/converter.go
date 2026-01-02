package md

import (
	"bytes"

	"github.com/yuin/goldmark"
)

// ToConfluenceStorage converts markdown content to Confluence storage format (XHTML).
func ToConfluenceStorage(markdown []byte) (string, error) {
	if len(markdown) == 0 {
		return "", nil
	}

	var buf bytes.Buffer
	if err := goldmark.Convert(markdown, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
