package api

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownToADF converts markdown text to an Atlassian Document Format document.
// Supports: headings (h1-h6), paragraphs, bold, italic, strikethrough, code,
// code blocks, bullet lists, numbered lists, links, blockquotes, and tables.
func MarkdownToADF(markdown string) *ADFDocument {
	if markdown == "" {
		return nil
	}

	source := []byte(markdown)
	reader := text.NewReader(source)
	// Use goldmark with table and strikethrough extensions
	md := goldmark.New(goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
	))
	doc := md.Parser().Parse(reader)

	content := convertNodes(doc, source)

	// If no content was parsed, fall back to simple text
	if len(content) == 0 {
		return &ADFDocument{
			Type:    "doc",
			Version: 1,
			Content: []ADFNode{
				{
					Type: "paragraph",
					Content: []ADFNode{
						{Type: "text", Text: markdown},
					},
				},
			},
		}
	}

	return &ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: content,
	}
}

func convertNodes(parent ast.Node, source []byte) []ADFNode {
	var nodes []ADFNode

	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if node := convertNode(child, source); node != nil {
			nodes = append(nodes, *node)
		}
	}

	return nodes
}

func convertNode(node ast.Node, source []byte) *ADFNode {
	switch n := node.(type) {
	case *ast.Heading:
		return convertHeading(n, source)
	case *ast.Paragraph:
		return convertParagraph(n, source)
	case *ast.CodeBlock:
		return convertCodeBlock(n, source)
	case *ast.FencedCodeBlock:
		return convertFencedCodeBlock(n, source)
	case *ast.List:
		return convertList(n, source)
	case *ast.Blockquote:
		return convertBlockquote(n, source)
	case *ast.ThematicBreak:
		return &ADFNode{Type: "rule"}
	case *east.Table:
		return convertTable(n, source)
	default:
		return nil
	}
}

func convertHeading(heading *ast.Heading, source []byte) *ADFNode {
	return &ADFNode{
		Type:    "heading",
		Attrs:   map[string]interface{}{"level": heading.Level},
		Content: convertInlineContent(heading, source),
	}
}

func convertParagraph(para *ast.Paragraph, source []byte) *ADFNode {
	content := convertInlineContent(para, source)
	if len(content) == 0 {
		return nil
	}
	return &ADFNode{
		Type:    "paragraph",
		Content: content,
	}
}

func convertCodeBlock(cb *ast.CodeBlock, source []byte) *ADFNode {
	var buf bytes.Buffer
	for i := 0; i < cb.Lines().Len(); i++ {
		line := cb.Lines().At(i)
		buf.Write(line.Value(source))
	}
	return &ADFNode{
		Type: "codeBlock",
		Content: []ADFNode{
			{Type: "text", Text: strings.TrimSuffix(buf.String(), "\n")},
		},
	}
}

func convertFencedCodeBlock(fcb *ast.FencedCodeBlock, source []byte) *ADFNode {
	var buf bytes.Buffer
	for i := 0; i < fcb.Lines().Len(); i++ {
		line := fcb.Lines().At(i)
		buf.Write(line.Value(source))
	}

	node := &ADFNode{
		Type: "codeBlock",
		Content: []ADFNode{
			{Type: "text", Text: strings.TrimSuffix(buf.String(), "\n")},
		},
	}

	// Add language attribute if specified
	if lang := string(fcb.Language(source)); lang != "" {
		node.Attrs = map[string]interface{}{"language": lang}
	}

	return node
}

func convertList(list *ast.List, source []byte) *ADFNode {
	listType := "bulletList"
	if list.IsOrdered() {
		listType = "orderedList"
	}

	var items []ADFNode
	for child := list.FirstChild(); child != nil; child = child.NextSibling() {
		if listItem, ok := child.(*ast.ListItem); ok {
			items = append(items, convertListItem(listItem, source))
		}
	}

	return &ADFNode{
		Type:    listType,
		Content: items,
	}
}

func convertListItem(item *ast.ListItem, source []byte) ADFNode {
	var content []ADFNode

	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		switch c := child.(type) {
		case *ast.TextBlock:
			// TextBlock is the inline content of a list item
			inlineContent := convertInlineContent(c, source)
			if len(inlineContent) > 0 {
				content = append(content, ADFNode{
					Type:    "paragraph",
					Content: inlineContent,
				})
			}
		case *ast.Paragraph:
			if para := convertParagraph(c, source); para != nil {
				content = append(content, *para)
			}
		case *ast.List:
			// Nested list
			if nestedList := convertList(c, source); nestedList != nil {
				content = append(content, *nestedList)
			}
		}
	}

	return ADFNode{
		Type:    "listItem",
		Content: content,
	}
}

func convertBlockquote(bq *ast.Blockquote, source []byte) *ADFNode {
	var content []ADFNode
	for child := bq.FirstChild(); child != nil; child = child.NextSibling() {
		if para, ok := child.(*ast.Paragraph); ok {
			if p := convertParagraph(para, source); p != nil {
				content = append(content, *p)
			}
		}
	}
	return &ADFNode{
		Type:    "blockquote",
		Content: content,
	}
}

func convertInlineContent(parent ast.Node, source []byte) []ADFNode {
	var nodes []ADFNode

	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		inlineNodes := convertInlineNode(child, source)
		nodes = append(nodes, inlineNodes...)
	}

	return nodes
}

func convertInlineNode(node ast.Node, source []byte) []ADFNode {
	switch n := node.(type) {
	case *ast.Text:
		text := string(n.Segment.Value(source))
		if text == "" {
			return nil
		}
		nodes := []ADFNode{{Type: "text", Text: text}}
		// Handle hard line breaks (two spaces at end of line)
		if n.HardLineBreak() {
			nodes = append(nodes, ADFNode{Type: "hardBreak"})
		} else if n.SoftLineBreak() {
			// Handle soft line breaks (newlines in source that aren't hard breaks)
			nodes = append(nodes, ADFNode{Type: "text", Text: " "})
		}
		return nodes

	case *ast.String:
		text := string(n.Value)
		if text == "" {
			return nil
		}
		return []ADFNode{{Type: "text", Text: text}}

	case *ast.Emphasis:
		content := convertInlineContent(n, source)
		markType := "em"
		if n.Level == 2 {
			markType = "strong"
		}
		return applyMark(content, markType)

	case *east.Strikethrough:
		content := convertInlineContent(n, source)
		return applyMark(content, "strike")

	case *ast.CodeSpan:
		var buf bytes.Buffer
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if text, ok := child.(*ast.Text); ok {
				buf.Write(text.Segment.Value(source))
			}
		}
		return []ADFNode{{
			Type:  "text",
			Text:  buf.String(),
			Marks: []ADFMark{{Type: "code"}},
		}}

	case *ast.Link:
		content := convertInlineContent(n, source)
		return applyMark(content, "link", map[string]interface{}{
			"href": string(n.Destination),
		})

	case *ast.AutoLink:
		url := string(n.URL(source))
		return []ADFNode{{
			Type:  "text",
			Text:  url,
			Marks: []ADFMark{{Type: "link", Attrs: map[string]interface{}{"href": url}}},
		}}

	case *ast.RawHTML:
		// Skip raw HTML
		return nil

	case *ast.Image:
		// Images in ADF are different - extract alt text from children
		var altBuilder strings.Builder
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if textNode, ok := child.(*ast.Text); ok {
				altBuilder.Write(textNode.Segment.Value(source))
			}
		}
		alt := altBuilder.String()
		if alt == "" {
			alt = string(n.Destination)
		}
		return []ADFNode{{Type: "text", Text: alt}}

	default:
		// For unknown inline nodes, try to extract text from children
		return convertInlineContent(node, source)
	}
}

func applyMark(nodes []ADFNode, markType string, attrs ...map[string]interface{}) []ADFNode {
	mark := ADFMark{Type: markType}
	if len(attrs) > 0 && attrs[0] != nil {
		mark.Attrs = attrs[0]
	}

	for i := range nodes {
		nodes[i].Marks = append(nodes[i].Marks, mark)
	}
	return nodes
}

func convertTable(table *east.Table, source []byte) *ADFNode {
	var rows []ADFNode

	for child := table.FirstChild(); child != nil; child = child.NextSibling() {
		switch row := child.(type) {
		case *east.TableHeader:
			rows = append(rows, convertTableRow(row, source, true))
		case *east.TableRow:
			rows = append(rows, convertTableRow(row, source, false))
		}
	}

	return &ADFNode{
		Type:    "table",
		Content: rows,
	}
}

func convertTableRow(row ast.Node, source []byte, isHeader bool) ADFNode {
	var cells []ADFNode

	for child := row.FirstChild(); child != nil; child = child.NextSibling() {
		if cell, ok := child.(*east.TableCell); ok {
			cellType := "tableCell"
			if isHeader {
				cellType = "tableHeader"
			}

			content := convertInlineContent(cell, source)
			// ADF table cells need paragraph wrappers
			cellContent := []ADFNode{}
			if len(content) > 0 {
				cellContent = append(cellContent, ADFNode{
					Type:    "paragraph",
					Content: content,
				})
			}

			cells = append(cells, ADFNode{
				Type:    cellType,
				Content: cellContent,
			})
		}
	}

	return ADFNode{
		Type:    "tableRow",
		Content: cells,
	}
}
