package md

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-cli-collective/atlassian-go/adf"
)

// FromADF converts an ADF JSON string to markdown.
// This is the inverse of ToADF: it walks the ADF document tree
// and renders each node as the corresponding markdown syntax.
func FromADF(adfJSON string) (string, error) {
	if adfJSON == "" {
		return "", nil
	}

	var doc adf.Document
	if err := json.Unmarshal([]byte(adfJSON), &doc); err != nil {
		return "", fmt.Errorf("parsing ADF JSON: %w", err)
	}

	var sb strings.Builder
	renderADFBlockNodes(&sb, doc.Content, 0, false)

	result := strings.TrimRight(sb.String(), "\n")
	if result != "" {
		result += "\n"
	}
	return result, nil
}

// renderADFBlockNodes renders a slice of block-level ADF nodes as markdown.
func renderADFBlockNodes(sb *strings.Builder, nodes []*adf.Node, depth int, inList bool) {
	for i, node := range nodes {
		renderADFBlockNode(sb, node, depth, inList)

		// Add blank line between top-level block nodes (not inside lists).
		if !inList && i < len(nodes)-1 {
			sb.WriteString("\n")
		}
	}
}

// renderADFBlockNode renders a single block-level ADF node as markdown.
func renderADFBlockNode(sb *strings.Builder, node *adf.Node, depth int, inList bool) {
	switch node.Type {
	case "heading":
		renderHeading(sb, node)
	case "paragraph":
		renderParagraph(sb, node, depth)
	case "bulletList":
		renderBulletList(sb, node, depth)
	case "orderedList":
		renderOrderedList(sb, node, depth)
	case "codeBlock":
		renderCodeBlock(sb, node)
	case "blockquote":
		renderBlockquote(sb, node, depth)
	case "table":
		renderTable(sb, node)
	case "rule":
		sb.WriteString("---\n")
	case "panel":
		renderPanel(sb, node, depth)
	case "extension":
		renderExtension(sb, node)
	case "bodiedExtension":
		renderBodiedExtension(sb, node, depth)
	case "mediaSingle", "mediaGroup":
		// Media nodes can't be represented in markdown; skip silently.
	default:
		// Unknown block node: render children as best effort.
		if len(node.Content) > 0 {
			renderADFBlockNodes(sb, node.Content, depth, inList)
		} else if node.Text != "" {
			sb.WriteString(node.Text)
			sb.WriteString("\n")
		}
	}
}

func renderHeading(sb *strings.Builder, node *adf.Node) {
	level := 1
	if l, ok := node.Attrs["level"]; ok {
		switch v := l.(type) {
		case float64:
			level = int(v)
		case int:
			level = v
		}
	}
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}

	sb.WriteString(strings.Repeat("#", level))
	sb.WriteString(" ")
	renderInlineNodes(sb, node.Content)
	sb.WriteString("\n")
}

func renderParagraph(sb *strings.Builder, node *adf.Node, _ int) {
	renderInlineNodes(sb, node.Content)
	sb.WriteString("\n")
}

func renderBulletList(sb *strings.Builder, node *adf.Node, depth int) {
	for _, item := range node.Content {
		if item.Type == "listItem" {
			renderListItem(sb, item, depth, "- ")
		}
	}
}

func renderOrderedList(sb *strings.Builder, node *adf.Node, depth int) {
	start := 1
	if s, ok := node.Attrs["order"]; ok {
		switch v := s.(type) {
		case float64:
			start = int(v)
		case int:
			start = v
		}
	}

	for i, item := range node.Content {
		if item.Type == "listItem" {
			prefix := fmt.Sprintf("%d. ", start+i)
			renderListItem(sb, item, depth, prefix)
		}
	}
}

func renderListItem(sb *strings.Builder, node *adf.Node, depth int, prefix string) {
	indent := strings.Repeat("  ", depth)

	for i, child := range node.Content {
		switch child.Type {
		case "paragraph":
			if i == 0 {
				sb.WriteString(indent)
				sb.WriteString(prefix)
			} else {
				// Continuation paragraphs inside a list item get indent only.
				sb.WriteString(indent)
				sb.WriteString(strings.Repeat(" ", len(prefix)))
			}
			renderInlineNodes(sb, child.Content)
			sb.WriteString("\n")
		case "bulletList":
			renderBulletList(sb, child, depth+1)
		case "orderedList":
			renderOrderedList(sb, child, depth+1)
		case "codeBlock":
			renderCodeBlock(sb, child)
		default:
			// First child gets the bullet prefix; others are indented.
			if i == 0 {
				sb.WriteString(indent)
				sb.WriteString(prefix)
			}
			renderADFBlockNode(sb, child, depth, true)
		}
	}
}

func renderCodeBlock(sb *strings.Builder, node *adf.Node) {
	lang := ""
	if l, ok := node.Attrs["language"]; ok {
		if s, ok := l.(string); ok {
			lang = s
		}
	}

	sb.WriteString("```")
	sb.WriteString(lang)
	sb.WriteString("\n")

	for _, child := range node.Content {
		if child.Type == "text" {
			sb.WriteString(child.Text)
		}
	}
	sb.WriteString("\n```\n")
}

func renderBlockquote(sb *strings.Builder, node *adf.Node, depth int) {
	// Render children into a temporary buffer, then prefix each line with "> ".
	var inner strings.Builder
	renderADFBlockNodes(&inner, node.Content, depth, false)

	for _, line := range strings.Split(strings.TrimRight(inner.String(), "\n"), "\n") {
		sb.WriteString("> ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}
}

func renderTable(sb *strings.Builder, node *adf.Node) {
	if len(node.Content) == 0 {
		return
	}

	// Collect all rows.
	var rows [][]string
	for _, row := range node.Content {
		if row.Type != "tableRow" {
			continue
		}
		var cells []string
		for _, cell := range row.Content {
			if cell.Type != "tableCell" && cell.Type != "tableHeader" {
				continue
			}
			var cellBuf strings.Builder
			for _, child := range cell.Content {
				if child.Type == "paragraph" {
					renderInlineNodes(&cellBuf, child.Content)
				}
			}
			cells = append(cells, strings.TrimSpace(cellBuf.String()))
		}
		rows = append(rows, cells)
	}

	if len(rows) == 0 {
		return
	}

	// Determine column count from the widest row.
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Pad short rows.
	for i := range rows {
		for len(rows[i]) < maxCols {
			rows[i] = append(rows[i], "")
		}
	}

	// Determine column widths.
	widths := make([]int, maxCols)
	for _, row := range rows {
		for j, cell := range row {
			if len(cell) > widths[j] {
				widths[j] = len(cell)
			}
		}
	}
	// Ensure minimum width of 3 for the separator.
	for i := range widths {
		if widths[i] < 3 {
			widths[i] = 3
		}
	}

	// Render header row.
	sb.WriteString("|")
	for j, cell := range rows[0] {
		sb.WriteString(" ")
		sb.WriteString(cell)
		sb.WriteString(strings.Repeat(" ", widths[j]-len(cell)))
		sb.WriteString(" |")
	}
	sb.WriteString("\n")

	// Render separator.
	sb.WriteString("|")
	for _, w := range widths {
		sb.WriteString(strings.Repeat("-", w+2))
		sb.WriteString("|")
	}
	sb.WriteString("\n")

	// Render data rows.
	for _, row := range rows[1:] {
		sb.WriteString("|")
		for j, cell := range row {
			sb.WriteString(" ")
			sb.WriteString(cell)
			sb.WriteString(strings.Repeat(" ", widths[j]-len(cell)))
			sb.WriteString(" |")
		}
		sb.WriteString("\n")
	}
}

// renderPanel converts ADF panel nodes to bracket macro syntax.
// Panel types: info, note, warning, tip, error → [INFO], [NOTE], etc.
func renderPanel(sb *strings.Builder, node *adf.Node, depth int) {
	panelType := "info"
	if pt, ok := node.Attrs["panelType"]; ok {
		if s, ok := pt.(string); ok {
			panelType = s
		}
	}

	macroName := panelTypeToMacroName(panelType)

	var inner strings.Builder
	renderADFBlockNodes(&inner, node.Content, depth, false)
	body := strings.TrimRight(inner.String(), "\n")

	sb.WriteString("[")
	sb.WriteString(macroName)
	sb.WriteString("]\n")
	if body != "" {
		sb.WriteString(body)
		sb.WriteString("\n")
	}
	sb.WriteString("[/")
	sb.WriteString(macroName)
	sb.WriteString("]\n")
}

// panelTypeToMacroName maps ADF panelType to bracket macro name.
func panelTypeToMacroName(panelType string) string {
	switch panelType {
	case "info":
		return "INFO"
	case "note":
		return "NOTE"
	case "warning":
		return "WARNING"
	case "tip":
		return "TIP"
	case "error":
		return "WARNING"
	default:
		return "INFO"
	}
}

// renderExtension converts bodyless ADF extension nodes (e.g., TOC) to bracket syntax.
func renderExtension(sb *strings.Builder, node *adf.Node) {
	key := ""
	if k, ok := node.Attrs["extensionKey"]; ok {
		if s, ok := k.(string); ok {
			key = s
		}
	}

	if key == "" {
		return
	}

	macroName := strings.ToUpper(key)
	params := extractExtensionParams(node)

	sb.WriteString("[")
	sb.WriteString(macroName)
	sb.WriteString(params)
	sb.WriteString("]\n")
}

// renderBodiedExtension converts bodied ADF extension nodes (e.g., EXPAND) to bracket syntax.
func renderBodiedExtension(sb *strings.Builder, node *adf.Node, depth int) {
	key := ""
	if k, ok := node.Attrs["extensionKey"]; ok {
		if s, ok := k.(string); ok {
			key = s
		}
	}

	if key == "" {
		if len(node.Content) > 0 {
			renderADFBlockNodes(sb, node.Content, depth, false)
		}
		return
	}

	macroName := strings.ToUpper(key)
	params := extractExtensionParams(node)

	var inner strings.Builder
	renderADFBlockNodes(&inner, node.Content, depth, false)
	body := strings.TrimRight(inner.String(), "\n")

	sb.WriteString("[")
	sb.WriteString(macroName)
	sb.WriteString(params)
	sb.WriteString("]\n")
	if body != "" {
		sb.WriteString(body)
		sb.WriteString("\n")
	}
	sb.WriteString("[/")
	sb.WriteString(macroName)
	sb.WriteString("]\n")
}

// extractExtensionParams builds a bracket-macro parameter string from extension attrs.
func extractExtensionParams(node *adf.Node) string {
	params, ok := node.Attrs["parameters"]
	if !ok {
		return ""
	}

	paramMap, ok := params.(map[string]any)
	if !ok {
		return ""
	}

	// Build "macroParams" structure: each param is {value: "..."}.
	var parts []string
	for name, entry := range paramMap {
		if name == "macroMetadata" {
			continue
		}
		if m, ok := entry.(map[string]any); ok {
			if v, ok := m["value"]; ok {
				parts = append(parts, fmt.Sprintf("%s=%v", name, v))
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

// renderInlineNodes renders a slice of inline ADF nodes (text, marks) as markdown.
func renderInlineNodes(sb *strings.Builder, nodes []*adf.Node) {
	for _, node := range nodes {
		renderInlineNode(sb, node)
	}
}

func renderInlineNode(sb *strings.Builder, node *adf.Node) {
	switch node.Type {
	case "text":
		renderTextWithMarks(sb, node)
	case "hardBreak":
		sb.WriteString("  \n")
	case "inlineCard":
		if url, ok := node.Attrs["url"]; ok {
			if s, ok := url.(string); ok {
				sb.WriteString(s)
			}
		}
	default:
		// Unknown inline node: output text if present.
		if node.Text != "" {
			sb.WriteString(node.Text)
		}
	}
}

// renderTextWithMarks wraps text in appropriate markdown formatting
// based on the marks applied to the text node.
func renderTextWithMarks(sb *strings.Builder, node *adf.Node) {
	text := node.Text
	if text == "" {
		return
	}

	// Check for link mark — needs special handling.
	var linkMark *adf.Mark
	hasCode := false
	hasStrong := false
	hasEm := false
	hasStrike := false

	for _, mark := range node.Marks {
		switch mark.Type {
		case "link":
			linkMark = mark
		case "code":
			hasCode = true
		case "strong":
			hasStrong = true
		case "em":
			hasEm = true
		case "strike":
			hasStrike = true
		}
	}

	// Code mark takes precedence — no nesting inside backticks.
	if hasCode {
		sb.WriteString("`")
		sb.WriteString(text)
		sb.WriteString("`")
		return
	}

	// Build the formatted text with marks.
	if hasStrike {
		text = "~~" + text + "~~"
	}
	if hasEm {
		text = "*" + text + "*"
	}
	if hasStrong {
		text = "**" + text + "**"
	}

	if linkMark != nil {
		href := ""
		if h, ok := linkMark.Attrs["href"]; ok {
			if s, ok := h.(string); ok {
				href = s
			}
		}
		if href != "" {
			sb.WriteString("[")
			sb.WriteString(text)
			sb.WriteString("](")
			sb.WriteString(href)
			sb.WriteString(")")
			return
		}
	}

	sb.WriteString(text)
}
