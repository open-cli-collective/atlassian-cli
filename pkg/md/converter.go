// Package md provides markdown conversion utilities for Confluence.
package md

import (
	"bytes"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// mdParser is a pre-configured goldmark instance with GFM table extension.
var mdParser = goldmark.New(
	goldmark.WithExtensions(extension.Table),
)

// macroPlaceholder is used to mark where macros should be inserted after goldmark processing.
// Using a format that won't be interpreted as markdown formatting (no underscores, asterisks, etc).
const macroPlaceholderPrefix = "CFMACRO"
const macroPlaceholderSuffix = "END"

// ToConfluenceStorage converts markdown content to Confluence storage format (XHTML).
func ToConfluenceStorage(markdown []byte) (string, error) {
	if len(markdown) == 0 {
		return "", nil
	}

	// Preprocess: replace macro placeholders with unique markers
	processed, macros := preprocessMacros(markdown)

	var buf bytes.Buffer
	if err := mdParser.Convert(processed, &buf); err != nil {
		return "", err
	}

	// Postprocess: replace markers with actual macro XML
	result := postprocessMacros(buf.String(), macros)

	return result, nil
}

// preprocessMacros replaces macro placeholders like [TOC] with unique markers.
// Returns the processed markdown and a map of marker IDs to macro XML.
//
// Uses ParseBracketMacros to correctly handle nested macros and close tags
// via its stack-based parser. This ensures:
// - Close tags like [/INFO] are properly consumed (not left as text)
// - Nested macros like [TOC] inside [INFO]...[/INFO] are correctly associated
func preprocessMacros(markdown []byte) ([]byte, map[int]string) {
	input := string(markdown)
	macros := make(map[int]string)

	// Parse using the stack-based parser which correctly handles nesting
	result, err := ParseBracketMacros(input)
	if err != nil {
		return markdown, macros
	}

	var outputBuf strings.Builder
	counter := 0

	// Process each segment from the parse result
	for _, seg := range result.Segments {
		switch seg.Type {
		case SegmentText:
			outputBuf.WriteString(seg.Text)
		case SegmentMacro:
			// Process macro node (handles nested children recursively)
			processMacroNode(seg.Macro, &outputBuf, macros, &counter)
		}
	}

	return []byte(outputBuf.String()), macros
}

// processMacroNode recursively processes a macro and its nested children.
// It converts markdown body content to HTML and renders the macro to XML.
//
// The body may contain child placeholders (CFCHILD0, CFCHILD1, etc.) that
// were inserted by ParseBracketMacros to mark where nested macros appear.
// These are replaced with the actual macro placeholders (CFMACRO0END, etc.)
// before converting the body markdown to HTML.
func processMacroNode(node *MacroNode, output *strings.Builder, macros map[int]string, counter *int) {
	macroType, _ := LookupMacro(node.Name)

	// If macro has body with nested children, we need to:
	// 1. Process each child to get its XML and placeholder ID
	// 2. Replace CFCHILD markers with macro placeholders
	// 3. Convert the body (with placeholders) to HTML
	// 4. Placeholders survive and get resolved in postprocessMacros
	if macroType.HasBody && node.Body != "" {
		bodyWithPlaceholders := node.Body

		// Process each child macro and replace CFCHILD markers with macro placeholders
		for i, child := range node.Children {
			// Recursively process the child (this increments counter, possibly multiple times)
			childOutput := &strings.Builder{}
			processMacroNode(child, childOutput, macros, counter)

			// Replace the child placeholder marker with the actual placeholder written by the child.
			// We use childOutput.String() because deeply nested macros increment the counter
			// multiple times, so we can't rely on a pre-captured counter value.
			childMarker := childPlaceholderPrefix + strconv.Itoa(i)
			bodyWithPlaceholders = strings.Replace(bodyWithPlaceholders, childMarker, childOutput.String(), 1)
		}

		// Convert body markdown to HTML (placeholders survive conversion)
		var bodyBuf bytes.Buffer
		if err := mdParser.Convert([]byte(bodyWithPlaceholders), &bodyBuf); err == nil {
			node.Body = bodyBuf.String()
		} else {
			node.Body = "<p>" + bodyWithPlaceholders + "</p>"
		}
	}

	// Render this macro to XML
	macroXML := RenderMacroToXML(node)
	currentID := *counter
	macros[currentID] = macroXML
	*counter++

	// Insert placeholder for this macro
	output.WriteString(FormatPlaceholder(currentID))
}

// postprocessMacros replaces placeholder markers with actual macro XML.
func postprocessMacros(html string, macros map[int]string) string {
	// Get sorted IDs - innermost macros have lowest IDs, so we must process
	// them in order to ensure nested placeholders are resolved before being
	// embedded in outer macros. Map iteration order is non-deterministic in Go.
	ids := make([]int, 0, len(macros))
	for id := range macros {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	// First pass: resolve any placeholders that exist within other macro values.
	// This handles nested macros (e.g., [TOC] inside [INFO]...[/INFO]).
	// The inner macro placeholder ends up embedded in the outer macro's XML.
	for _, id := range ids {
		macroXML := macros[id]
		for _, innerId := range ids {
			if innerId >= id {
				continue // Only replace placeholders from earlier (inner) macros
			}
			placeholder := FormatPlaceholder(innerId)
			if strings.Contains(macroXML, placeholder) {
				macros[id] = strings.Replace(macroXML, placeholder, macros[innerId], 1)
				macroXML = macros[id]
			}
		}
	}

	// Second pass: replace placeholders in the main HTML
	for _, id := range ids {
		macroXML := macros[id]
		placeholder := FormatPlaceholder(id)
		// The placeholder might be wrapped in <p> tags, so handle that
		wrappedPlaceholder := "<p>" + placeholder + "</p>"
		if strings.Contains(html, wrappedPlaceholder) {
			html = strings.Replace(html, wrappedPlaceholder, macroXML, 1)
		} else {
			html = strings.Replace(html, placeholder, macroXML, 1)
		}
	}
	return html
}

// parseKeyValueParams parses a string like "key1=value1 key2=value2" into ["key1=value1", "key2=value2"].
// Handles values with quotes: key="value with spaces"
func parseKeyValueParams(s string) []string {
	var params []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for i, r := range s {
		switch {
		case (r == '"' || r == '\'') && !inQuotes:
			inQuotes = true
			quoteChar = r
			// Don't include the opening quote in the value
		case r == quoteChar && inQuotes:
			inQuotes = false
			quoteChar = 0
			// Don't include the closing quote in the value
		case r == ' ' && !inQuotes:
			if current.Len() > 0 {
				params = append(params, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}

		// Handle end of string
		if i == len(s)-1 && current.Len() > 0 {
			params = append(params, current.String())
		}
	}

	return params
}
