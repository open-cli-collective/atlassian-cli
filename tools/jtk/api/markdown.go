package api //nolint:revive // package name is intentional

import (
	"encoding/json"

	"github.com/open-cli-collective/atlassian-go/adf"
)

// MarkdownToADF converts text to an Atlassian Document Format document.
// Supports: headings (h1-h6), paragraphs, bold, italic, strikethrough, code,
// code blocks, bullet lists, numbered lists, links, blockquotes, and tables.
//
// Auto-detection is conservative by design: it prioritizes not corrupting plain
// markdown over detecting every wiki edge case. Inline-only wiki formatting
// (e.g., ~subscript~ without block-level markers like h1.) will NOT be detected.
// This bias is intentional for mixed content from LLM agents and user input.
// Callers that know the input is wiki markup should call WikiToADFMarkdown +
// adf.ToDocumentWiki directly to bypass heuristics.
//
// Before any markdown handling, the input is checked for raw ADF: if it
// parses as JSON shaped like an ADF document ({"type":"doc","version":1,...}
// with at least one content node), it is unmarshaled into the internal ADF
// types and re-marshaled from them rather than returned as the original
// bytes. This lets callers pass through documents built by another tool
// (e.g. containing inlineCard nodes) without the markdown converter
// mangling them, but it is not a byte-for-byte passthrough: unknown node
// keys outside type/attrs/content/text/marks are dropped, and numeric attrs
// round-trip through float64. Spec-conformant ADF is unaffected. Anything
// else — invalid JSON, valid JSON that isn't a doc, a doc with version != 1
// or no content, or markdown that merely starts with "{" — falls through to
// markdown conversion unchanged.
func MarkdownToADF(markdown string) *ADFDocument {
	if markdown == "" {
		return nil
	}

	if doc, ok := parseRawADFDocument(markdown); ok {
		return doc
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

// IsRawADFDocument reports whether text would be treated as raw ADF
// passthrough by MarkdownToADF. Callers that apply CLI-only text
// conveniences (such as escape-sequence interpretation for \n and \t)
// before handing text to NewADFDocument / MarkdownToADF should skip those
// conveniences when this returns true: escape interpretation is a markdown
// convenience, and applying it to raw ADF JSON first can corrupt it (e.g. by
// turning an escaped "\n" inside a JSON string into a literal newline byte)
// before the ADF parser ever sees it.
func IsRawADFDocument(text string) bool {
	_, ok := parseRawADFDocument(text)
	return ok
}

// parseRawADFDocument reports whether text is a pre-built ADF document
// passed through as raw JSON, returning the parsed document when it is.
// It requires the JSON to unmarshal cleanly into an ADF document shape with
// type "doc", version exactly 1 (the only version the ADF spec and issue
// #484 define), and at least one content node; anything else (invalid
// JSON, valid JSON of a different shape, an unrecognized version, or a
// content-less doc) is rejected so it falls through to markdown conversion.
// Content-less docs are rejected rather than passed through because
// marshaling a nil Content slice produces "content": null, which Jira's
// API rejects with a 400.
func parseRawADFDocument(text string) (*ADFDocument, bool) {
	var doc ADFDocument
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		return nil, false
	}
	if doc.Type != "doc" || doc.Version != 1 || len(doc.Content) == 0 {
		return nil, false
	}
	return &doc, true
}
