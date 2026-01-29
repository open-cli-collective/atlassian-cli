// render.go provides functions to render MacroNodes to Confluence storage format.
package md

import (
	"fmt"
	"sort"
	"strings"
)

// RenderMacroToXML converts a MacroNode to Confluence XML storage format.
func RenderMacroToXML(node *MacroNode) string {
	var sb strings.Builder

	// Opening tag
	sb.WriteString(`<ac:structured-macro ac:name="`)
	sb.WriteString(node.Name)
	sb.WriteString(`" ac:schema-version="1">`)

	// Parameters (sorted for consistent output)
	keys := make([]string, 0, len(node.Parameters))
	for k := range node.Parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := node.Parameters[key]
		sb.WriteString(`<ac:parameter ac:name="`)
		sb.WriteString(key)
		sb.WriteString(`">`)
		sb.WriteString(escapeXML(value))
		sb.WriteString(`</ac:parameter>`)
	}

	// Body content
	macroType, _ := LookupMacro(node.Name)
	if macroType.HasBody && node.Body != "" {
		switch macroType.BodyType {
		case BodyTypeRichText:
			sb.WriteString(`<ac:rich-text-body>`)
			sb.WriteString(node.Body)
			sb.WriteString(`</ac:rich-text-body>`)
		case BodyTypePlainText:
			sb.WriteString(`<ac:plain-text-body><![CDATA[`)
			sb.WriteString(node.Body)
			sb.WriteString(`]]></ac:plain-text-body>`)
		}
	}

	// Closing tag
	sb.WriteString(`</ac:structured-macro>`)

	return sb.String()
}

// escapeXML escapes special XML characters in a string.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// RenderMacroToBracket converts a MacroNode back to bracket syntax.
func RenderMacroToBracket(node *MacroNode) string {
	var sb strings.Builder

	// Render opening bracket with parameters
	sb.WriteString(RenderMacroToBracketOpen(node))

	// Body and close tag for macros with body
	macroType, _ := LookupMacro(node.Name)
	if macroType.HasBody {
		sb.WriteString(node.Body)
		sb.WriteString("[/")
		sb.WriteString(strings.ToUpper(node.Name))
		sb.WriteString("]")
	}

	return sb.String()
}

// RenderMacroToBracketOpen renders just the opening bracket tag (without body or close).
func RenderMacroToBracketOpen(node *MacroNode) string {
	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(strings.ToUpper(node.Name))

	// Parameters (sorted for consistent output)
	keys := make([]string, 0, len(node.Parameters))
	for k := range node.Parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := node.Parameters[key]
		sb.WriteString(" ")
		sb.WriteString(key)
		sb.WriteString("=")
		if strings.ContainsAny(value, " \t\n\"") {
			sb.WriteString(`"`)
			sb.WriteString(strings.ReplaceAll(value, `"`, `\"`))
			sb.WriteString(`"`)
		} else {
			sb.WriteString(value)
		}
	}
	sb.WriteString("]")
	return sb.String()
}

// FormatPlaceholder creates a macro placeholder string.
func FormatPlaceholder(id int) string {
	return fmt.Sprintf("%s%d%s", macroPlaceholderPrefix, id, macroPlaceholderSuffix)
}
