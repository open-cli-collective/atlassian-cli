package md

import (
	"fmt"
	"regexp"
	"strconv"
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
	placeholderOpenPrefix  = "CFMACROOPEN"
	placeholderClosePrefix = "CFMACROCLOSE"
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

// macroPlaceholder stores the bracket syntax for a macro placeholder
type macroPlaceholder struct {
	openTag  string // e.g., "[INFO title=Title]"
	closeTag string // e.g., "[/INFO]" (empty for simple macros)
}

// replaceMacroPlaceholders replaces placeholder markers with actual bracket syntax
func replaceMacroPlaceholders(markdown string, macroMap map[int]macroPlaceholder) string {
	for id, macro := range macroMap {
		openPlaceholder := fmt.Sprintf("%s%d", placeholderOpenPrefix, id)
		closePlaceholder := fmt.Sprintf("%s%d", placeholderClosePrefix, id)

		markdown = strings.Replace(markdown, openPlaceholder, macro.openTag, 1)
		if macro.closeTag != "" {
			markdown = strings.Replace(markdown, closePlaceholder, macro.closeTag, 1)
		}
	}
	return markdown
}

// processConfluenceMacrosWithPlaceholders processes Confluence macros in HTML.
// When showMacros is true, macros are converted to bracket syntax.
// When showMacros is false, macros are stripped from output.
func processConfluenceMacrosWithPlaceholders(html string, showMacros bool) (string, map[int]macroPlaceholder) {
	// Convert code blocks first (special handling)
	html = convertCodeBlockMacros(html)

	macroMap := make(map[int]macroPlaceholder)

	if !showMacros {
		// Strip all macros without placeholders
		return stripConfluenceMacros(html), macroMap
	}

	// Parse the XML to extract macros
	result, err := ParseConfluenceXML(html)
	if err != nil {
		return html, macroMap
	}

	// Build output with placeholders
	var output strings.Builder
	var macroIDCounter macroIDTracker

	for _, seg := range result.Segments {
		switch seg.Type {
		case SegmentText:
			output.WriteString(seg.Text)
		case SegmentMacro:
			// Recursively process macro and nested macros
			macroIDCounter.addMacrosWithPlaceholders(seg.Macro, &output, macroMap)
		}
	}

	return output.String(), macroMap
}

// macroIDTracker tracks IDs for macro placeholders during recursive processing
type macroIDTracker struct {
	nextID int
}

// addMacrosWithPlaceholders recursively adds a macro and its nested macros to the output as placeholders.
// The body may contain CFXMLCHILD placeholders indicating where nested macros appeared in the original XML.
func (t *macroIDTracker) addMacrosWithPlaceholders(node *MacroNode, output *strings.Builder, macroMap map[int]macroPlaceholder) {
	currentID := t.nextID
	t.nextID++

	// Create placeholder for this macro
	placeholder := renderMacroToPlaceholders(node, currentID)
	macroMap[currentID] = placeholder

	// Add opening placeholder
	output.WriteString(placeholderOpenPrefix + strconv.Itoa(currentID))

	// If macro has body, process it
	macroType, _ := LookupMacro(node.Name)
	if macroType.HasBody {
		if len(node.Children) > 0 {
			// Body contains CFXMLCHILD markers where nested macros should appear.
			// Process each child and replace its marker with the actual macro placeholder.
			bodyWithPlaceholders := node.Body
			for i, child := range node.Children {
				// Recursively process the child (this registers the child in macroMap)
				childOutput := &strings.Builder{}
				t.addMacrosWithPlaceholders(child, childOutput, macroMap)

				// Replace the child marker in the body with the child's content
				childMarker := fmt.Sprintf("%s%d", xmlChildPlaceholderPrefix, i)
				bodyWithPlaceholders = strings.Replace(bodyWithPlaceholders, childMarker, childOutput.String(), 1)
			}
			output.WriteString(bodyWithPlaceholders)
		} else {
			// No nested macros, just write body
			output.WriteString(node.Body)
		}
		// Add closing placeholder
		output.WriteString(placeholderClosePrefix + strconv.Itoa(currentID))
	}
}

// renderMacroToPlaceholders creates a macroPlaceholder from a MacroNode.
func renderMacroToPlaceholders(node *MacroNode, id int) macroPlaceholder {
	macroType, _ := LookupMacro(node.Name)

	openTag := RenderMacroToBracketOpen(node)

	var closeTag string
	if macroType.HasBody {
		closeTag = "[/" + strings.ToUpper(node.Name) + "]"
	}

	return macroPlaceholder{
		openTag:  openTag,
		closeTag: closeTag,
	}
}

// stripConfluenceMacros removes all Confluence structured macros from HTML.
func stripConfluenceMacros(html string) string {
	result, err := ParseConfluenceXML(html)
	if err != nil {
		return html
	}

	var output strings.Builder
	for _, seg := range result.Segments {
		if seg.Type == SegmentText {
			output.WriteString(seg.Text)
		}
		// Macros are silently dropped
	}
	return output.String()
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
