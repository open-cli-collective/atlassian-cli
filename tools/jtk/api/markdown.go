package api //nolint:revive // package name is intentional

import (
	"github.com/open-cli-collective/atlassian-go/adf"
)

// MarkdownToADF converts markdown text to an Atlassian Document Format document.
// Supports: headings (h1-h6), paragraphs, bold, italic, strikethrough, code,
// code blocks, bullet lists, numbered lists, links, blockquotes, and tables.
//
// Auto-detection is conservative by design: it prioritizes not corrupting plain
// markdown over detecting every wiki edge case. Inline-only wiki formatting
// (e.g., ~subscript~ without block-level markers like h1.) will NOT be detected.
// This bias is intentional for mixed content from LLM agents and user input.
// Callers that know the input is wiki markup should call WikiToADFMarkdown +
// adf.ToDocumentWiki directly to bypass heuristics.
func MarkdownToADF(markdown string) *ADFDocument {
	if markdown == "" {
		return nil
	}

	// Auto-detect and convert wiki markup to markdown.
	// Wiki-converted text uses the extended parser (subscript, superscript,
	// insert) since ~text~ and ^text^ are intentional wiki formatting.
	// Plain markdown uses the standard parser to avoid mangling tildes
	// and carets in compound words (e.g., "signal~webapp~frontend").
	if IsWikiMarkup(markdown) {
		markdown = WikiToADFMarkdown(markdown)
		return adf.ToDocumentWiki(markdown)
	}

	return adf.ToDocument(markdown)
}
