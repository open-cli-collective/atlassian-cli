package md

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
)

// ConvertOptions configures the HTML to markdown conversion.
type ConvertOptions struct {
	// ShowMacros shows placeholder text for Confluence macros instead of stripping them.
	ShowMacros bool
}

// Placeholder markers for macro brackets (avoid html-to-markdown escaping)
const (
	macroOpenPrefix  = "CFMACROOPEN"
	macroCloseSuffix = "CFMACROCLOSE"
)

// FromConfluenceStorage converts Confluence storage format (XHTML) to markdown.
func FromConfluenceStorage(html string) (string, error) {
	return FromConfluenceStorageWithOptions(html, ConvertOptions{})
}

// FromConfluenceStorageWithOptions converts Confluence storage format (XHTML) to markdown
// with configurable options.
func FromConfluenceStorageWithOptions(html string, opts ConvertOptions) (string, error) {
	if html == "" {
		return "", nil
	}

	// Process Confluence macros before conversion, get placeholders map
	html, macroMap := processConfluenceMacrosWithPlaceholders(html, opts.ShowMacros)

	// Create converter with table support
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
		),
	)

	markdown, err := conv.ConvertString(html)
	if err != nil {
		return "", err
	}

	// Replace placeholders with actual bracket syntax
	markdown = replaceMacroPlaceholders(markdown, macroMap)

	// Clean up the output - trim whitespace
	return strings.TrimSpace(markdown), nil
}

// panelMacroNames contains macro names that have body content (ac:rich-text-body)
var panelMacroNames = map[string]bool{
	"info":    true,
	"warning": true,
	"note":    true,
	"tip":     true,
	"expand":  true,
}

// macroPlaceholder stores the bracket syntax for a macro placeholder
type macroPlaceholder struct {
	openTag  string // e.g., "[INFO title=Title]"
	closeTag string // e.g., "[/INFO]" (empty for simple macros)
}

// replaceMacroPlaceholders replaces placeholder markers with actual bracket syntax
func replaceMacroPlaceholders(markdown string, macroMap map[int]macroPlaceholder) string {
	for id, macro := range macroMap {
		openPlaceholder := fmt.Sprintf("%s%d", macroOpenPrefix, id)
		closePlaceholder := fmt.Sprintf("%s%d", macroCloseSuffix, id)

		markdown = strings.Replace(markdown, openPlaceholder, macro.openTag, 1)
		if macro.closeTag != "" {
			markdown = strings.Replace(markdown, closePlaceholder, macro.closeTag, 1)
		}
	}
	return markdown
}

// processConfluenceMacrosWithPlaceholders handles Confluence-specific macro elements.
// If showMacros is false, macros are stripped entirely.
// If showMacros is true, macros are replaced with placeholders that will be converted
// to bracket syntax after markdown conversion (to avoid escaping issues).
// Returns the processed HTML and a map of placeholder IDs to bracket syntax.
func processConfluenceMacrosWithPlaceholders(html string, showMacros bool) (string, map[int]macroPlaceholder) {
	macroMap := make(map[int]macroPlaceholder)
	counter := 0

	// First, convert code block macros to HTML pre/code elements
	html = convertCodeBlockMacros(html)

	const openTag = "<ac:structured-macro"
	const closeTag = "</ac:structured-macro>"

	// Process macros from innermost to outermost.
	// We must use string-based parsing because regex cannot properly handle
	// nested XML structures (it would match from outer open to inner close).
	for {
		startIdx, macroStr, endIdx := findInnermostMacro(html, openTag, closeTag)
		if startIdx == -1 {
			break
		}

		var replacement string
		if showMacros {
			placeholder, macro := convertMacroToPlaceholders(macroStr, counter)
			macroMap[counter] = macro
			counter++
			replacement = placeholder
		}

		html = html[:startIdx] + replacement + html[endIdx:]
	}

	// Strip other Confluence-specific elements
	html = stripConfluenceElements(html)

	return html, macroMap
}

// findInnermostMacro finds a macro that contains no nested macros.
// Returns the start index, the full macro string, and the end index.
// Returns -1, "", -1 if no macro is found.
func findInnermostMacro(html, openTag, closeTag string) (int, string, int) {
	searchStart := 0

	for searchStart < len(html) {
		// Find next opening tag
		openIdx := strings.Index(html[searchStart:], openTag)
		if openIdx == -1 {
			return -1, "", -1
		}
		openIdx += searchStart

		// Find matching closing tag (handling nesting with balanced counting)
		closeIdx := findMatchingCloseTag(html, openIdx, openTag, closeTag)
		if closeIdx == -1 {
			return -1, "", -1
		}

		macroStr := html[openIdx:closeIdx]

		// Check if this macro contains nested macros
		// Look for openTag after the initial tag's closing >
		tagEndIdx := strings.Index(macroStr, ">")
		if tagEndIdx == -1 {
			searchStart = openIdx + 1
			continue
		}

		innerContent := macroStr[tagEndIdx+1:]
		if !strings.Contains(innerContent, openTag) {
			// No nested macros - this is innermost
			return openIdx, macroStr, closeIdx
		}

		// Has nested macros, continue searching from inside this macro
		// to find the innermost one
		searchStart = openIdx + tagEndIdx + 1
	}

	return -1, "", -1
}

// findMatchingCloseTag finds the matching close tag for an open tag at openIdx,
// properly handling nested tags using balanced parentheses counting.
func findMatchingCloseTag(html string, openIdx int, openTag, closeTag string) int {
	depth := 1
	pos := openIdx + len(openTag)

	for pos < len(html) && depth > 0 {
		remaining := html[pos:]

		nextOpen := strings.Index(remaining, openTag)
		nextClose := strings.Index(remaining, closeTag)

		if nextClose == -1 {
			// No closing tag found - malformed
			return -1
		}

		if nextOpen != -1 && nextOpen < nextClose {
			// Found another opening tag first - increase depth
			depth++
			pos += nextOpen + len(openTag)
		} else {
			// Found closing tag first - decrease depth
			depth--
			if depth == 0 {
				return pos + nextClose + len(closeTag)
			}
			pos += nextClose + len(closeTag)
		}
	}

	return -1
}

// convertMacroToPlaceholders converts a Confluence macro to placeholder markers.
// Returns the placeholder HTML and the bracket syntax to substitute later.
func convertMacroToPlaceholders(match string, id int) (string, macroPlaceholder) {
	// Extract macro name
	nameMatch := regexp.MustCompile(`ac:name="([^"]*)"`).FindStringSubmatch(match)
	if len(nameMatch) < 2 {
		return fmt.Sprintf("%s%d", macroOpenPrefix, id), macroPlaceholder{openTag: "[MACRO]"}
	}
	macroName := strings.ToUpper(nameMatch[1])
	macroNameLower := strings.ToLower(nameMatch[1])

	// Extract parameters from the macro, but only those that are direct children
	// (not nested inside <ac:rich-text-body> which may contain other macros)
	paramSearchArea := match
	if bodyStart := strings.Index(match, "<ac:rich-text-body>"); bodyStart != -1 {
		paramSearchArea = match[:bodyStart]
	}
	paramPattern := regexp.MustCompile(`<ac:parameter[^>]*ac:name="([^"]*)"[^>]*>([^<]*)</ac:parameter>`)
	paramMatches := paramPattern.FindAllStringSubmatch(paramSearchArea, -1)

	// Build parameter string
	var params []string
	for _, p := range paramMatches {
		if len(p) >= 3 {
			paramName := strings.TrimSpace(p[1])
			paramValue := strings.TrimSpace(p[2])
			if paramName != "" && paramValue != "" {
				// Quote values that contain spaces
				if strings.Contains(paramValue, " ") {
					paramValue = `"` + paramValue + `"`
				}
				params = append(params, paramName+"="+paramValue)
			}
		}
	}

	// Build opening tag with parameters
	openTag := "[" + macroName
	if len(params) > 0 {
		openTag += " " + strings.Join(params, " ")
	}
	openTag += "]"

	openPlaceholder := fmt.Sprintf("%s%d", macroOpenPrefix, id)
	closePlaceholder := fmt.Sprintf("%s%d", macroCloseSuffix, id)

	// Check if this is a panel macro with body content
	if panelMacroNames[macroNameLower] {
		// Extract body content from <ac:rich-text-body>
		bodyPattern := regexp.MustCompile(`(?s)<ac:rich-text-body>(.*?)</ac:rich-text-body>`)
		bodyMatch := bodyPattern.FindStringSubmatch(match)

		bodyContent := ""
		if len(bodyMatch) > 1 {
			bodyContent = strings.TrimSpace(bodyMatch[1])
		}

		closeTag := "[/" + macroName + "]"

		if bodyContent == "" {
			// Empty body - put open and close placeholders together
			return openPlaceholder + closePlaceholder, macroPlaceholder{openTag: openTag, closeTag: closeTag}
		}

		// Body content will be converted by html-to-markdown
		return openPlaceholder + "\n" + bodyContent + "\n" + closePlaceholder,
			macroPlaceholder{openTag: openTag, closeTag: closeTag}
	}

	// Simple macro without body (like TOC)
	return openPlaceholder, macroPlaceholder{openTag: openTag}
}

// stripConfluenceElements removes other Confluence-specific elements from HTML
func stripConfluenceElements(html string) string {
	// ac:link elements (internal Confluence links)
	linkPattern := regexp.MustCompile(`<ac:link[^>]*>.*?</ac:link>`)
	html = linkPattern.ReplaceAllString(html, "")

	// ri:page references
	pageRefPattern := regexp.MustCompile(`<ri:page[^>]*/?>`)
	html = pageRefPattern.ReplaceAllString(html, "")

	// ac:plain-text-link-body (link text)
	linkBodyPattern := regexp.MustCompile(`<ac:plain-text-link-body><!\[CDATA\[(.*?)\]\]></ac:plain-text-link-body>`)
	html = linkBodyPattern.ReplaceAllString(html, "$1")

	// ac:parameter elements (macro parameters like minLevel, maxLevel)
	paramPattern := regexp.MustCompile(`<ac:parameter[^>]*>.*?</ac:parameter>`)
	html = paramPattern.ReplaceAllString(html, "")

	// Self-closing ac:parameter
	paramSelfClosingPattern := regexp.MustCompile(`<ac:parameter[^>]*/>`)
	html = paramSelfClosingPattern.ReplaceAllString(html, "")

	return html
}

// convertCodeBlockMacros converts Confluence code macro elements to HTML pre/code elements.
// This preserves code blocks when converting to markdown.
func convertCodeBlockMacros(html string) string {
	// Match code block macros - use (?s) flag for . to match newlines
	// Confluence code blocks: <ac:structured-macro ac:name="code" ...>...</ac:structured-macro>
	codeBlockPattern := regexp.MustCompile(`(?s)<ac:structured-macro[^>]*ac:name="code"[^>]*>(.*?)</ac:structured-macro>`)

	return codeBlockPattern.ReplaceAllStringFunc(html, func(match string) string {
		// Extract language parameter if present
		// <ac:parameter ac:name="language">python</ac:parameter>
		langPattern := regexp.MustCompile(`<ac:parameter[^>]*ac:name="language"[^>]*>([^<]*)</ac:parameter>`)
		langMatch := langPattern.FindStringSubmatch(match)
		language := ""
		if len(langMatch) > 1 {
			language = strings.TrimSpace(langMatch[1])
		}

		// Extract code content from CDATA
		// <ac:plain-text-body><![CDATA[code here]]></ac:plain-text-body>
		cdataPattern := regexp.MustCompile(`(?s)<ac:plain-text-body><!\[CDATA\[(.*?)\]\]></ac:plain-text-body>`)
		cdataMatch := cdataPattern.FindStringSubmatch(match)
		code := ""
		if len(cdataMatch) > 1 {
			code = cdataMatch[1]
		}

		// Convert to HTML pre/code which the markdown converter understands
		if language != "" {
			return "<pre><code class=\"language-" + language + "\">" + escapeHTMLInCode(code) + "</code></pre>"
		}
		return "<pre><code>" + escapeHTMLInCode(code) + "</code></pre>"
	})
}

// escapeHTMLInCode escapes HTML special characters in code content.
func escapeHTMLInCode(code string) string {
	code = strings.ReplaceAll(code, "&", "&amp;")
	code = strings.ReplaceAll(code, "<", "&lt;")
	code = strings.ReplaceAll(code, ">", "&gt;")
	return code
}
