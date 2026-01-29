package md

import (
	"encoding/json"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// ADFDocument represents an Atlassian Document Format document.
type ADFDocument struct {
	Type    string     `json:"type"`
	Version int        `json:"version"`
	Content []*ADFNode `json:"content"`
}

// ADFNode represents a node in an ADF document.
type ADFNode struct {
	Type    string                 `json:"type"`
	Attrs   map[string]interface{} `json:"attrs,omitempty"`
	Content []*ADFNode             `json:"content,omitempty"`
	Text    string                 `json:"text,omitempty"`
	Marks   []*ADFMark             `json:"marks,omitempty"`
}

// ADFMark represents a text mark (formatting) in ADF.
type ADFMark struct {
	Type  string                 `json:"type"`
	Attrs map[string]interface{} `json:"attrs,omitempty"`
}

// adfParser is a goldmark parser configured for ADF conversion.
var adfParser = goldmark.New(
	goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
)

// ToADF converts markdown content to Atlassian Document Format (ADF) JSON.
// The returned string is a JSON-encoded ADF document.
func ToADF(markdown []byte) (string, error) {
	doc := &ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []*ADFNode{},
	}

	if len(markdown) == 0 {
		result, err := json.Marshal(doc)
		if err != nil {
			return "", err
		}
		return string(result), nil
	}

	reader := text.NewReader(markdown)
	astDoc := adfParser.Parser().Parse(reader)

	// Walk the AST and convert to ADF
	converter := &adfConverter{source: markdown}
	doc.Content = converter.convertChildren(astDoc)

	result, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// adfConverter holds state during AST conversion.
type adfConverter struct {
	source []byte
}

// convertChildren converts all children of an AST node to ADF nodes.
func (c *adfConverter) convertChildren(n ast.Node) []*ADFNode {
	var nodes []*ADFNode
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if node := c.convertNode(child); node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// convertNode converts a single AST node to an ADF node.
func (c *adfConverter) convertNode(n ast.Node) *ADFNode {
	switch node := n.(type) {
	case *ast.Paragraph:
		return c.convertParagraph(node)
	case *ast.Heading:
		return c.convertHeading(node)
	case *ast.List:
		return c.convertList(node)
	case *ast.ListItem:
		return c.convertListItem(node)
	case *ast.FencedCodeBlock:
		return c.convertFencedCodeBlock(node)
	case *ast.CodeBlock:
		return c.convertCodeBlock(node)
	case *ast.Blockquote:
		return c.convertBlockquote(node)
	case *ast.ThematicBreak:
		return &ADFNode{Type: "rule"}
	case *extast.Table:
		return c.convertTable(node)
	case *ast.TextBlock:
		return c.convertTextBlock(node)
	default:
		// For unknown block types, try to get text content
		return nil
	}
}

func (c *adfConverter) convertParagraph(n *ast.Paragraph) *ADFNode {
	content := c.convertInlineChildren(n)
	if len(content) == 0 {
		return nil
	}
	return &ADFNode{
		Type:    "paragraph",
		Content: content,
	}
}

func (c *adfConverter) convertHeading(n *ast.Heading) *ADFNode {
	return &ADFNode{
		Type:    "heading",
		Attrs:   map[string]interface{}{"level": n.Level},
		Content: c.convertInlineChildren(n),
	}
}

func (c *adfConverter) convertList(n *ast.List) *ADFNode {
	listType := "bulletList"
	var attrs map[string]interface{}
	if n.IsOrdered() {
		listType = "orderedList"
		attrs = map[string]interface{}{"order": n.Start}
	}

	return &ADFNode{
		Type:    listType,
		Attrs:   attrs,
		Content: c.convertChildren(n),
	}
}

func (c *adfConverter) convertListItem(n *ast.ListItem) *ADFNode {
	var content []*ADFNode
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch ch := child.(type) {
		case *ast.TextBlock:
			// Convert TextBlock to paragraph in list context
			para := c.convertTextBlockToParagraph(ch)
			if para != nil {
				content = append(content, para)
			}
		case *ast.Paragraph:
			para := c.convertParagraph(ch)
			if para != nil {
				content = append(content, para)
			}
		case *ast.List:
			list := c.convertList(ch)
			if list != nil {
				content = append(content, list)
			}
		default:
			if node := c.convertNode(child); node != nil {
				content = append(content, node)
			}
		}
	}

	return &ADFNode{
		Type:    "listItem",
		Content: content,
	}
}

func (c *adfConverter) convertTextBlock(n *ast.TextBlock) *ADFNode {
	// TextBlock in non-list context - convert to paragraph
	return c.convertTextBlockToParagraph(n)
}

func (c *adfConverter) convertTextBlockToParagraph(n *ast.TextBlock) *ADFNode {
	content := c.convertInlineChildren(n)
	if len(content) == 0 {
		return nil
	}
	return &ADFNode{
		Type:    "paragraph",
		Content: content,
	}
}

func (c *adfConverter) convertFencedCodeBlock(n *ast.FencedCodeBlock) *ADFNode {
	var code strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		code.Write(line.Value(c.source))
	}

	// Trim trailing newline
	codeStr := strings.TrimSuffix(code.String(), "\n")

	node := &ADFNode{
		Type: "codeBlock",
		Content: []*ADFNode{
			{Type: "text", Text: codeStr},
		},
	}

	// Add language attribute if present
	if lang := string(n.Language(c.source)); lang != "" {
		node.Attrs = map[string]interface{}{"language": lang}
	}

	return node
}

func (c *adfConverter) convertCodeBlock(n *ast.CodeBlock) *ADFNode {
	var code strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		code.Write(line.Value(c.source))
	}

	codeStr := strings.TrimSuffix(code.String(), "\n")

	return &ADFNode{
		Type: "codeBlock",
		Content: []*ADFNode{
			{Type: "text", Text: codeStr},
		},
	}
}

func (c *adfConverter) convertBlockquote(n *ast.Blockquote) *ADFNode {
	return &ADFNode{
		Type:    "blockquote",
		Content: c.convertChildren(n),
	}
}

func (c *adfConverter) convertTable(n *extast.Table) *ADFNode {
	var rows []*ADFNode
	isFirstRow := true

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if row, ok := child.(*extast.TableRow); ok {
			rows = append(rows, c.convertTableRow(row, isFirstRow))
			isFirstRow = false
		} else if header, ok := child.(*extast.TableHeader); ok {
			rows = append(rows, c.convertTableHeader(header))
			isFirstRow = false
		}
	}

	return &ADFNode{
		Type:    "table",
		Attrs:   map[string]interface{}{"layout": "default"},
		Content: rows,
	}
}

func (c *adfConverter) convertTableHeader(n *extast.TableHeader) *ADFNode {
	var cells []*ADFNode
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if cell, ok := child.(*extast.TableCell); ok {
			cells = append(cells, c.convertTableCell(cell, true))
		}
	}
	return &ADFNode{
		Type:    "tableRow",
		Content: cells,
	}
}

func (c *adfConverter) convertTableRow(n *extast.TableRow, isHeader bool) *ADFNode {
	var cells []*ADFNode
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if cell, ok := child.(*extast.TableCell); ok {
			cells = append(cells, c.convertTableCell(cell, isHeader))
		}
	}
	return &ADFNode{
		Type:    "tableRow",
		Content: cells,
	}
}

func (c *adfConverter) convertTableCell(n *extast.TableCell, isHeader bool) *ADFNode {
	cellType := "tableCell"
	if isHeader {
		cellType = "tableHeader"
	}

	// Get cell content
	content := c.convertInlineChildren(n)
	var para *ADFNode
	if len(content) > 0 {
		para = &ADFNode{Type: "paragraph", Content: content}
	} else {
		para = &ADFNode{Type: "paragraph", Content: []*ADFNode{{Type: "text", Text: ""}}}
	}

	return &ADFNode{
		Type:    cellType,
		Attrs:   map[string]interface{}{"colspan": 1, "rowspan": 1},
		Content: []*ADFNode{para},
	}
}

// convertInlineChildren converts all inline children of an AST node to ADF text nodes.
func (c *adfConverter) convertInlineChildren(n ast.Node) []*ADFNode {
	var nodes []*ADFNode
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		textNodes := c.convertInlineNode(child, nil)
		nodes = append(nodes, textNodes...)
	}
	return nodes
}

// convertInlineNode converts an inline AST node to ADF text node(s).
func (c *adfConverter) convertInlineNode(n ast.Node, marks []*ADFMark) []*ADFNode {
	switch node := n.(type) {
	case *ast.Text:
		text := string(node.Segment.Value(c.source))
		if text == "" {
			return nil
		}
		adfNode := &ADFNode{Type: "text", Text: text}
		if len(marks) > 0 {
			adfNode.Marks = marks
		}
		return []*ADFNode{adfNode}

	case *ast.String:
		text := string(node.Value)
		if text == "" {
			return nil
		}
		adfNode := &ADFNode{Type: "text", Text: text}
		if len(marks) > 0 {
			adfNode.Marks = marks
		}
		return []*ADFNode{adfNode}

	case *ast.Emphasis:
		// Determine mark type based on emphasis level
		markType := "em"
		if node.Level == 2 {
			markType = "strong"
		}

		newMarks := append(copyMarks(marks), &ADFMark{Type: markType})
		var nodes []*ADFNode
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			nodes = append(nodes, c.convertInlineNode(child, newMarks)...)
		}
		return nodes

	case *extast.Strikethrough:
		newMarks := append(copyMarks(marks), &ADFMark{Type: "strike"})
		var nodes []*ADFNode
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			nodes = append(nodes, c.convertInlineNode(child, newMarks)...)
		}
		return nodes

	case *ast.CodeSpan:
		// Build text from child text nodes
		var textBuilder strings.Builder
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if textNode, ok := child.(*ast.Text); ok {
				textBuilder.Write(textNode.Segment.Value(c.source))
			}
		}
		text := textBuilder.String()
		newMarks := append(copyMarks(marks), &ADFMark{Type: "code"})
		return []*ADFNode{{Type: "text", Text: text, Marks: newMarks}}

	case *ast.Link:
		linkMark := &ADFMark{
			Type:  "link",
			Attrs: map[string]interface{}{"href": string(node.Destination)},
		}
		newMarks := append(copyMarks(marks), linkMark)
		var nodes []*ADFNode
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			nodes = append(nodes, c.convertInlineNode(child, newMarks)...)
		}
		return nodes

	case *ast.AutoLink:
		url := string(node.URL(c.source))
		linkMark := &ADFMark{
			Type:  "link",
			Attrs: map[string]interface{}{"href": url},
		}
		newMarks := append(copyMarks(marks), linkMark)
		return []*ADFNode{{Type: "text", Text: url, Marks: newMarks}}

	case *ast.RawHTML:
		// Skip raw HTML
		return nil

	case *ast.Image:
		// Images would need special handling - for now return alt text
		// Build alt text from child text nodes (node.Text is deprecated)
		var altBuilder strings.Builder
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if textNode, ok := child.(*ast.Text); ok {
				altBuilder.Write(textNode.Segment.Value(c.source))
			}
		}
		alt := altBuilder.String()
		if alt == "" {
			alt = string(node.Destination)
		}
		adfNode := &ADFNode{Type: "text", Text: alt}
		if len(marks) > 0 {
			adfNode.Marks = marks
		}
		return []*ADFNode{adfNode}

	default:
		// For unknown inline types, try to recurse into children
		var nodes []*ADFNode
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			nodes = append(nodes, c.convertInlineNode(child, marks)...)
		}
		return nodes
	}
}

// copyMarks creates a copy of the marks slice.
func copyMarks(marks []*ADFMark) []*ADFMark {
	if marks == nil {
		return nil
	}
	result := make([]*ADFMark, len(marks))
	copy(result, marks)
	return result
}
