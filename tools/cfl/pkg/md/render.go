package md

import (
	"fmt"
	"sort"
	"strings"

	"github.com/open-cli-collective/atlassian-go/adf"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
		case BodyTypeNone:
			// no body wrapper needed
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

// adfPlaceholderPrefix is used for ADF macro placeholders (distinct from storage).
const adfPlaceholderPrefix = "CFADFM"
const adfPlaceholderSuffix = "ENDA"

// FormatADFPlaceholder creates an ADF macro placeholder string.
func FormatADFPlaceholder(id int) string {
	return fmt.Sprintf("%s%d%s", adfPlaceholderPrefix, id, adfPlaceholderSuffix)
}

// panelMacros maps macro names to their ADF panel type.
var panelMacros = map[string]string{
	"info":    "info",
	"warning": "warning",
	"note":    "note",
	"tip":     "success",
}

// RenderMacroToADFNode converts a MacroNode to an ADF node.
//
// Bodyless macros (e.g., TOC) become "extension" nodes.
// Panel macros (info, warning, note, tip) become "panel" nodes.
// Other body macros (expand, code) become "bodiedExtension" nodes.
func RenderMacroToADFNode(node *MacroNode) *adf.Node {
	macroType, _ := LookupMacro(node.Name)

	// Build macro parameters in the ADF format
	params := buildADFMacroParams(node)

	if macroType.HasBody {
		// Convert body markdown to ADF content nodes
		var content []*adf.Node
		if node.Body != "" {
			bodyDoc := adf.ToDocument(node.Body)
			if bodyDoc != nil {
				content = bodyDoc.Content
			}
		}
		if content == nil {
			content = []*adf.Node{}
		}

		// Panel macros get native panel nodes
		if panelType, ok := panelMacros[node.Name]; ok {
			return &adf.Node{
				Type:    "panel",
				Attrs:   map[string]any{"panelType": panelType},
				Content: content,
			}
		}

		// Other body macros get bodiedExtension nodes
		return &adf.Node{
			Type: "bodiedExtension",
			Attrs: map[string]any{
				"extensionType": "com.atlassian.confluence.macro.core",
				"extensionKey":  node.Name,
				"parameters":    params,
				"layout":        "default",
			},
			Content: content,
		}
	}

	// Bodyless macros get extension nodes
	return &adf.Node{
		Type: "extension",
		Attrs: map[string]any{
			"extensionType": "com.atlassian.confluence.macro.core",
			"extensionKey":  node.Name,
			"parameters":    params,
			"layout":        "default",
		},
	}
}

// buildADFMacroParams builds the ADF parameters structure for a macro.
func buildADFMacroParams(node *MacroNode) map[string]any {
	macroParams := make(map[string]any)
	for k, v := range node.Parameters {
		macroParams[k] = map[string]any{"value": v}
	}

	macroTitle := macroDisplayName(node.Name)
	return map[string]any{
		"macroParams": macroParams,
		"macroMetadata": map[string]any{
			"schemaVersion": map[string]any{"value": "1"},
			"title":         macroTitle,
		},
	}
}

// macroDisplayName returns the human-readable display name for a macro.
func macroDisplayName(name string) string {
	displayNames := map[string]string{
		"toc":     "Table of Contents",
		"info":    "Info",
		"warning": "Warning",
		"note":    "Note",
		"tip":     "Tip",
		"expand":  "Expand",
		"code":    "Code Block",
	}
	if dn, ok := displayNames[name]; ok {
		return dn
	}
	return cases.Title(language.English).String(name)
}
