package md

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/open-cli-collective/atlassian-go/adf"
)

// Type aliases for backward compatibility with the shared adf package.
type ADFDocument = adf.Document
type ADFNode = adf.Node
type ADFMark = adf.Mark

// ToADF converts markdown content to Atlassian Document Format (ADF) JSON.
// The returned string is a JSON-encoded ADF document.
//
// Wiki-links like [[Page Title]] are converted to standard markdown links
// before ADF conversion, producing text nodes with link marks.
// Bracket macros like [TOC] are converted to ADF extension nodes.
// Code regions (fenced blocks, inline code) are excluded from conversion.
func ToADF(markdown []byte) (string, error) {
	if len(markdown) == 0 {
		doc := &adf.Document{Type: "doc", Version: 1, Content: []*adf.Node{}}
		result, err := json.Marshal(doc)
		if err != nil {
			return "", err
		}
		return string(result), nil
	}

	// Preprocess wiki-links into standard markdown links before ADF conversion
	processed := preprocessWikiLinksForADF(markdown)

	// Preprocess bracket macros (e.g., [TOC], [INFO]...[/INFO])
	processed, macros := preprocessMacrosForADF(processed)

	// Convert to ADF document struct
	doc := adf.ToDocument(string(processed))
	if doc == nil {
		doc = &adf.Document{Type: "doc", Version: 1, Content: []*adf.Node{}}
	}

	// Replace macro placeholders with ADF extension/panel nodes
	if len(macros) > 0 {
		doc.Content = replaceMacroPlaceholdersADF(doc.Content, macros)
	}

	result, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// preprocessMacrosForADF replaces bracket macros with alphanumeric placeholders,
// storing the parsed MacroNode for each. Unlike the storage path, body content
// is kept as raw markdown (not converted to HTML) so it can be converted to ADF
// nodes during postprocessing.
func preprocessMacrosForADF(markdown []byte) ([]byte, map[int]*MacroNode) {
	// Protect code regions
	protected, codeRegions := protectCodeRegions(markdown)

	result, err := ParseBracketMacros(string(protected))
	if err != nil {
		return markdown, nil
	}

	macros := make(map[int]*MacroNode)
	var output strings.Builder
	counter := 0

	for _, seg := range result.Segments {
		switch seg.Type {
		case SegmentText:
			output.WriteString(seg.Text)
		case SegmentMacro:
			processMacroNodeForADF(seg.Macro, &output, macros, &counter)
		}
	}

	// Restore code regions
	processed := restoreCodeRegions([]byte(output.String()), codeRegions)
	return processed, macros
}

// processMacroNodeForADF recursively processes a macro and its nested children,
// storing MacroNode references (with raw markdown body) for ADF postprocessing.
func processMacroNodeForADF(node *MacroNode, output *strings.Builder, macros map[int]*MacroNode, counter *int) {
	macroType, _ := LookupMacro(node.Name)

	if macroType.HasBody && node.Body != "" {
		bodyWithPlaceholders := node.Body

		// Process each child macro recursively
		for i, child := range node.Children {
			childOutput := &strings.Builder{}
			processMacroNodeForADF(child, childOutput, macros, counter)

			childMarker := childPlaceholderPrefix + strconv.Itoa(i)
			bodyWithPlaceholders = strings.Replace(bodyWithPlaceholders, childMarker, childOutput.String(), 1)
		}

		node.Body = bodyWithPlaceholders
	}

	currentID := *counter
	macros[currentID] = node
	*counter++

	output.WriteString(FormatADFPlaceholder(currentID))
}

// replaceMacroPlaceholdersADF walks the ADF node tree and replaces paragraph
// nodes containing a macro placeholder with the corresponding ADF extension node.
func replaceMacroPlaceholdersADF(nodes []*adf.Node, macros map[int]*MacroNode) []*adf.Node {
	var result []*adf.Node
	for _, node := range nodes {
		if id, ok := extractADFPlaceholder(node); ok {
			macroNode := macros[id]
			result = append(result, RenderMacroToADFNode(macroNode))
			continue
		}

		// Recurse into children (for blockquotes, list items, etc.)
		if len(node.Content) > 0 {
			node.Content = replaceMacroPlaceholdersADF(node.Content, macros)
		}
		result = append(result, node)
	}
	return result
}

// extractADFPlaceholder checks if an ADF node is a paragraph containing only
// a macro placeholder. Returns the placeholder ID and true if found.
func extractADFPlaceholder(node *adf.Node) (int, bool) {
	if node.Type != "paragraph" || len(node.Content) != 1 {
		return 0, false
	}
	textNode := node.Content[0]
	if textNode.Type != "text" || len(textNode.Marks) > 0 {
		return 0, false
	}
	text := strings.TrimSpace(textNode.Text)
	if !strings.HasPrefix(text, adfPlaceholderPrefix) || !strings.HasSuffix(text, adfPlaceholderSuffix) {
		return 0, false
	}
	idStr := text[len(adfPlaceholderPrefix) : len(text)-len(adfPlaceholderSuffix)]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, false
	}
	return id, true
}
