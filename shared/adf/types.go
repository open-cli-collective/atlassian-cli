package adf

import "strings"

// Document represents an Atlassian Document Format document.
type Document struct {
	Type    string  `json:"type"`
	Version int     `json:"version"`
	Content []*Node `json:"content"`
}

// Node represents a node in an ADF document.
type Node struct {
	Type    string         `json:"type"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Content []*Node        `json:"content,omitempty"`
	Text    string         `json:"text,omitempty"`
	Marks   []*Mark        `json:"marks,omitempty"`
}

// Mark represents text formatting (bold, italic, link, etc.) in ADF.
type Mark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// ToPlainText extracts plain text from a Document.
func (d *Document) ToPlainText() string {
	if d == nil {
		return ""
	}
	return extractTextWithDepth(d.Content, 0)
}

func extractTextWithDepth(nodes []*Node, depth int) string {
	var result string
	for _, node := range nodes {
		switch node.Type {
		case "heading":
			result += "\n"
			if len(node.Content) > 0 {
				result += extractTextWithDepth(node.Content, depth)
			}
			result += "\n"
		case "codeBlock":
			result += "\n"
			if len(node.Content) > 0 {
				result += extractTextWithDepth(node.Content, depth)
			}
			result += "\n"
		case "bulletList", "orderedList":
			if len(node.Content) > 0 {
				result += extractTextWithDepth(node.Content, depth)
			}
			result += "\n"
		case "listItem":
			indent := ""
			for i := 0; i < depth; i++ {
				indent += "  "
			}
			result += indent + "- "
			if len(node.Content) > 0 {
				result += extractTextWithDepth(node.Content, depth+1)
			}
		case "blockquote":
			if len(node.Content) > 0 {
				inner := extractTextWithDepth(node.Content, depth)
				for _, line := range splitLines(inner) {
					result += "> " + line + "\n"
				}
			}
		case "rule":
			result += "---\n"
		case "paragraph":
			if len(node.Content) > 0 {
				result += extractTextWithDepth(node.Content, depth)
			}
			result += "\n"
		case "hardBreak":
			result += "\n"
		case "inlineCard":
			result += attrString(node, "url")
		case "mention":
			result += attrString(node, "text")
		case "blockCard", "embedCard":
			if url := attrString(node, "url"); url != "" {
				result += url + "\n"
			}
		default:
			if node.Text != "" {
				result += node.Text
			}
			if len(node.Content) > 0 {
				result += extractTextWithDepth(node.Content, depth)
			}
		}
	}
	return result
}

func attrString(node *Node, key string) string {
	s, _ := node.Attrs[key].(string)
	return s
}

// splitLines splits text into lines, stripping trailing empty lines.
func splitLines(s string) []string {
	lines := strings.Split(s, "\n")
	// Trim trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
