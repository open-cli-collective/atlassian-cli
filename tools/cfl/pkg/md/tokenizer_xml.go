// tokenizer_xml.go implements tokenization for Confluence storage format XML.
package md

import (
	"regexp"
	"strings"
)

// Regex patterns for Confluence XML elements
var (
	// Matches self-closing <ac:structured-macro ac:name="NAME" ... />
	// Must be checked BEFORE macroOpenPattern to avoid incorrect matches
	// Uses non-greedy [^>]*? to avoid over-matching across attributes
	macroSelfClosingPattern = regexp.MustCompile(`<ac:structured-macro\s+[^>]*?ac:name="([^"]*)"[^>]*?/>`)
	// Matches <ac:structured-macro ac:name="NAME" ...>
	macroOpenPattern = regexp.MustCompile(`<ac:structured-macro[^>]*ac:name="([^"]*)"[^>]*>`)
	// Matches </ac:structured-macro>
	macroClosePattern = regexp.MustCompile(`</ac:structured-macro>`)
	// Matches <ac:parameter ac:name="NAME">VALUE</ac:parameter>
	paramPattern = regexp.MustCompile(`<ac:parameter[^>]*ac:name="([^"]*)"[^>]*>([^<]*)</ac:parameter>`)
	// Matches <ac:rich-text-body> opening
	richTextBodyOpen = regexp.MustCompile(`<ac:rich-text-body>`)
	// Matches </ac:rich-text-body> closing
	richTextBodyClose = regexp.MustCompile(`</ac:rich-text-body>`)
	// Matches <ac:plain-text-body> opening
	plainTextBodyOpen = regexp.MustCompile(`<ac:plain-text-body>`)
	// Matches </ac:plain-text-body> closing
	plainTextBodyClose = regexp.MustCompile(`</ac:plain-text-body>`)
	// Matches CDATA content: <![CDATA[...]]>
	cdataPattern = regexp.MustCompile(`(?s)<!\[CDATA\[(.*?)\]\]>`)
)

// TokenizeConfluenceXML scans input for Confluence storage format macros and returns a token stream.
// This tokenizer produces a flat stream of tokens that the parser will assemble into a tree.
func TokenizeConfluenceXML(input string) ([]XMLToken, error) {
	var tokens []XMLToken
	pos := 0

	for pos < len(input) {
		// Try to find the next macro or body tag
		remaining := input[pos:]

		// Check for self-closing macro tag (must check before regular open tag)
		// Self-closing tags like <ac:structured-macro ac:name="toc" /> need both open and close tokens
		if loc := macroSelfClosingPattern.FindStringSubmatchIndex(remaining); loc != nil && loc[0] == 0 {
			macroName := remaining[loc[2]:loc[3]]
			// Emit open tag
			tokens = append(tokens, XMLToken{
				Type:      XMLTokenOpenTag,
				MacroName: strings.ToLower(macroName),
				Position:  pos,
			})
			// Immediately emit close tag for self-closing
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenCloseTag,
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Check for macro open tag
		if loc := macroOpenPattern.FindStringSubmatchIndex(remaining); loc != nil && loc[0] == 0 {
			macroName := remaining[loc[2]:loc[3]]
			tokens = append(tokens, XMLToken{
				Type:      XMLTokenOpenTag,
				MacroName: strings.ToLower(macroName),
				Position:  pos,
			})
			pos += loc[1]
			continue
		}

		// Check for macro close tag
		if loc := macroClosePattern.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenCloseTag,
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Check for parameter
		if loc := paramPattern.FindStringSubmatchIndex(remaining); loc != nil && loc[0] == 0 {
			paramName := remaining[loc[2]:loc[3]]
			paramValue := remaining[loc[4]:loc[5]]
			tokens = append(tokens, XMLToken{
				Type:      XMLTokenParameter,
				ParamName: paramName,
				Value:     paramValue,
				Position:  pos,
			})
			pos += loc[1]
			continue
		}

		// Check for rich-text-body open
		if loc := richTextBodyOpen.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenBody,
				Value:    "rich-text",
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Check for rich-text-body close
		if loc := richTextBodyClose.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenBodyEnd,
				Value:    "rich-text",
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Check for plain-text-body open
		if loc := plainTextBodyOpen.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenBody,
				Value:    "plain-text",
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Check for plain-text-body close
		if loc := plainTextBodyClose.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenBodyEnd,
				Value:    "plain-text",
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Check for CDATA (inside plain-text-body)
		if loc := cdataPattern.FindStringSubmatchIndex(remaining); loc != nil && loc[0] == 0 {
			cdataContent := remaining[loc[2]:loc[3]]
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenText,
				Text:     cdataContent,
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Find the next macro-related tag
		nextMacroSelfClosing := macroSelfClosingPattern.FindStringIndex(remaining)
		nextMacroOpen := macroOpenPattern.FindStringIndex(remaining)
		nextMacroClose := macroClosePattern.FindStringIndex(remaining)
		nextParam := paramPattern.FindStringIndex(remaining)
		nextRichOpen := richTextBodyOpen.FindStringIndex(remaining)
		nextRichClose := richTextBodyClose.FindStringIndex(remaining)
		nextPlainOpen := plainTextBodyOpen.FindStringIndex(remaining)
		nextPlainClose := plainTextBodyClose.FindStringIndex(remaining)

		// Find minimum positive start position
		nextTagPos := len(remaining)
		for _, loc := range [][]int{nextMacroSelfClosing, nextMacroOpen, nextMacroClose, nextParam, nextRichOpen, nextRichClose, nextPlainOpen, nextPlainClose} {
			if loc != nil && loc[0] > 0 && loc[0] < nextTagPos {
				nextTagPos = loc[0]
			}
		}

		if nextTagPos > 0 {
			// Emit text up to next tag
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenText,
				Text:     remaining[:nextTagPos],
				Position: pos,
			})
			pos += nextTagPos
		} else {
			// Single character fallback (shouldn't normally happen)
			pos++
		}
	}

	return tokens, nil
}

// ExtractCDATAContent extracts content from a CDATA section.
// Input: "<![CDATA[content]]>" Output: "content"
func ExtractCDATAContent(s string) string {
	if match := cdataPattern.FindStringSubmatch(s); match != nil {
		return match[1]
	}
	return s
}
