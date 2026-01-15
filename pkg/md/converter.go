// Package md provides markdown conversion utilities for Confluence.
package md

import (
	"bytes"
	"fmt"
	"regexp"
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
func preprocessMacros(markdown []byte) ([]byte, map[int]string) {
	result := string(markdown)
	macros := make(map[int]string)
	counter := 0

	// Convert [TOC] or [TOC param=value ...] to placeholder
	// Case-insensitive matching for the macro name
	tocPattern := regexp.MustCompile(`(?i)\[TOC(?:\s+([^\]]*))?\]`)
	result = tocPattern.ReplaceAllStringFunc(result, func(match string) string {
		macroXML := convertTOCMacro(match)
		macros[counter] = macroXML
		placeholder := macroPlaceholderPrefix + fmt.Sprintf("%d", counter) + macroPlaceholderSuffix
		counter++
		return placeholder
	})

	// Convert panel macros: [INFO]...[/INFO], [WARNING]...[/WARNING], etc.
	// Go's regexp doesn't support backreferences, so we match each type separately
	panelTypes := []string{"INFO", "WARNING", "NOTE", "TIP", "EXPAND"}
	for _, panelType := range panelTypes {
		// Case-insensitive matching, supports parameters like [INFO title="Title"]
		pattern := regexp.MustCompile(`(?is)\[` + panelType + `([^\]]*)\](.*?)\[/` + panelType + `\]`)
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			macroXML := convertPanelMacro(match, panelType)
			macros[counter] = macroXML
			placeholder := macroPlaceholderPrefix + fmt.Sprintf("%d", counter) + macroPlaceholderSuffix
			counter++
			return placeholder
		})
	}

	return []byte(result), macros
}

// postprocessMacros replaces placeholder markers with actual macro XML.
func postprocessMacros(html string, macros map[int]string) string {
	// First pass: resolve any placeholders that exist within other macro values.
	// This handles nested macros (e.g., [TOC] inside [INFO]...[/INFO]).
	// The inner macro placeholder ends up embedded in the outer macro's XML.
	for id, macroXML := range macros {
		for innerId, innerXML := range macros {
			if innerId == id {
				continue
			}
			placeholder := macroPlaceholderPrefix + fmt.Sprintf("%d", innerId) + macroPlaceholderSuffix
			if strings.Contains(macroXML, placeholder) {
				macros[id] = strings.Replace(macroXML, placeholder, innerXML, 1)
				macroXML = macros[id]
			}
		}
	}

	// Second pass: replace placeholders in the main HTML
	for id, macroXML := range macros {
		placeholder := macroPlaceholderPrefix + fmt.Sprintf("%d", id) + macroPlaceholderSuffix
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

// convertPanelMacro converts a [INFO]...[/INFO] style placeholder to Confluence structured macro XML.
func convertPanelMacro(match string, panelType string) string {
	// Extract parameters and body content
	pattern := regexp.MustCompile(`(?is)\[` + panelType + `([^\]]*)\](.*?)\[/` + panelType + `\]`)
	groups := pattern.FindStringSubmatch(match)

	if len(groups) < 3 {
		return match // Return unchanged if pattern doesn't match
	}

	macroName := strings.ToLower(panelType)
	paramStr := strings.TrimSpace(groups[1])
	bodyContent := strings.TrimSpace(groups[2])

	// Parse parameters
	var params []string
	if paramStr != "" {
		params = parseKeyValueParams(paramStr)
	}

	// Convert body content from markdown to HTML
	var bodyHTML string
	if bodyContent != "" {
		// Use goldmark to convert the body content
		var buf bytes.Buffer
		if err := mdParser.Convert([]byte(bodyContent), &buf); err == nil {
			bodyHTML = buf.String()
		} else {
			// Fallback: wrap in paragraph
			bodyHTML = "<p>" + bodyContent + "</p>"
		}
	}

	// Build the Confluence macro XML
	var sb strings.Builder
	sb.WriteString(`<ac:structured-macro ac:name="`)
	sb.WriteString(macroName)
	sb.WriteString(`" ac:schema-version="1">`)

	// Add parameters
	for _, param := range params {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			sb.WriteString(`<ac:parameter ac:name="`)
			sb.WriteString(name)
			sb.WriteString(`">`)
			sb.WriteString(value)
			sb.WriteString(`</ac:parameter>`)
		}
	}

	// Add body content
	if bodyHTML != "" {
		sb.WriteString(`<ac:rich-text-body>`)
		sb.WriteString(strings.TrimSpace(bodyHTML))
		sb.WriteString(`</ac:rich-text-body>`)
	}

	sb.WriteString(`</ac:structured-macro>`)

	return sb.String()
}

// convertTOCMacro converts a [TOC ...] placeholder to Confluence structured macro XML.
func convertTOCMacro(match string) string {
	// Extract parameters from [TOC param1=value1 param2=value2]
	tocPattern := regexp.MustCompile(`(?i)\[TOC(?:\s+([^\]]*))?\]`)
	groups := tocPattern.FindStringSubmatch(match)

	var params []string
	if len(groups) > 1 && groups[1] != "" {
		// Parse key=value pairs
		paramStr := strings.TrimSpace(groups[1])
		params = parseKeyValueParams(paramStr)
	}

	// Build the Confluence macro XML
	var sb strings.Builder
	sb.WriteString(`<ac:structured-macro ac:name="toc" ac:schema-version="1">`)
	for _, param := range params {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			sb.WriteString(`<ac:parameter ac:name="`)
			sb.WriteString(name)
			sb.WriteString(`">`)
			sb.WriteString(value)
			sb.WriteString(`</ac:parameter>`)
		}
	}
	sb.WriteString(`</ac:structured-macro>`)

	return sb.String()
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
