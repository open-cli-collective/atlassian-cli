// Package md provides markdown conversion utilities for Confluence.
package md

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// mdParser is a pre-configured goldmark instance with GFM table extension.
var mdParser = goldmark.New(
	goldmark.WithExtensions(extension.Table),
)

// ToConfluenceStorage converts markdown content to Confluence storage format (XHTML).
func ToConfluenceStorage(markdown []byte) (string, error) {
	if len(markdown) == 0 {
		return "", nil
	}

	var buf bytes.Buffer
	if err := mdParser.Convert(markdown, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
