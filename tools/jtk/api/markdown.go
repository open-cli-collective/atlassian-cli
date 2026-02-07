package api

import (
	"github.com/open-cli-collective/atlassian-go/adf"
)

// MarkdownToADF converts markdown text to an Atlassian Document Format document.
// Supports: headings (h1-h6), paragraphs, bold, italic, strikethrough, code,
// code blocks, bullet lists, numbered lists, links, blockquotes, and tables.
//
// If the input contains Jira wiki markup (h1., {{code}}, [text|url], etc.),
// it will be automatically converted to markdown first.
func MarkdownToADF(markdown string) *ADFDocument {
	if markdown == "" {
		return nil
	}

	// Auto-detect and convert wiki markup to markdown
	if IsWikiMarkup(markdown) {
		markdown = WikiToMarkdown(markdown)
	}

	return adf.ToDocument(markdown)
}
