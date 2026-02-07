package adf

import (
	"encoding/json"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// mdParser is a goldmark parser configured for ADF conversion.
var mdParser = goldmark.New(
	goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
	),
)

// ToDocument converts markdown text to an ADF Document struct.
// Returns nil for empty input. If parsing yields no content,
// falls back to a single paragraph with the raw text.
func ToDocument(markdown string) *Document {
	if markdown == "" {
		return nil
	}

	source := []byte(markdown)
	reader := text.NewReader(source)
	astDoc := mdParser.Parser().Parse(reader)

	converter := &converter{source: source}
	content := converter.convertChildren(astDoc)

	if len(content) == 0 {
		return &Document{
			Type:    "doc",
			Version: 1,
			Content: []*Node{
				{
					Type:    "paragraph",
					Content: []*Node{{Type: "text", Text: markdown}},
				},
			},
		}
	}

	return &Document{
		Type:    "doc",
		Version: 1,
		Content: content,
	}
}

// ToJSON converts markdown to an ADF JSON string.
// Returns an empty document JSON for empty input.
func ToJSON(markdown []byte) (string, error) {
	doc := &Document{
		Type:    "doc",
		Version: 1,
		Content: []*Node{},
	}

	if len(markdown) == 0 {
		result, err := json.Marshal(doc)
		if err != nil {
			return "", err
		}
		return string(result), nil
	}

	reader := text.NewReader(markdown)
	astDoc := mdParser.Parser().Parse(reader)

	c := &converter{source: markdown}
	doc.Content = c.convertChildren(astDoc)

	result, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// converter holds state during AST-to-ADF conversion.
type converter struct {
	source []byte
}

// convertChildren converts all children of an AST node to ADF nodes.
func (c *converter) convertChildren(n ast.Node) []*Node {
	var nodes []*Node
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if node := c.convertNode(child); node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// convertNode converts a single AST node to an ADF node.
func (c *converter) convertNode(n ast.Node) *Node {
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
		return &Node{Type: "rule"}
	case *extast.Table:
		return c.convertTable(node)
	case *ast.TextBlock:
		return c.convertTextBlockToParagraph(node)
	default:
		return nil
	}
}

func (c *converter) convertParagraph(n *ast.Paragraph) *Node {
	content := c.convertInlineChildren(n)
	if len(content) == 0 {
		return nil
	}
	return &Node{
		Type:    "paragraph",
		Content: content,
	}
}

func (c *converter) convertHeading(n *ast.Heading) *Node {
	return &Node{
		Type:    "heading",
		Attrs:   map[string]interface{}{"level": n.Level},
		Content: c.convertInlineChildren(n),
	}
}

func (c *converter) convertList(n *ast.List) *Node {
	listType := "bulletList"
	var attrs map[string]interface{}
	if n.IsOrdered() {
		listType = "orderedList"
		attrs = map[string]interface{}{"order": n.Start}
	}

	return &Node{
		Type:    listType,
		Attrs:   attrs,
		Content: c.convertChildren(n),
	}
}

func (c *converter) convertListItem(n *ast.ListItem) *Node {
	var content []*Node
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch ch := child.(type) {
		case *ast.TextBlock:
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

	return &Node{
		Type:    "listItem",
		Content: content,
	}
}

func (c *converter) convertTextBlockToParagraph(n *ast.TextBlock) *Node {
	content := c.convertInlineChildren(n)
	if len(content) == 0 {
		return nil
	}
	return &Node{
		Type:    "paragraph",
		Content: content,
	}
}

func (c *converter) convertFencedCodeBlock(n *ast.FencedCodeBlock) *Node {
	var code strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		code.Write(line.Value(c.source))
	}

	codeStr := strings.TrimSuffix(code.String(), "\n")

	node := &Node{
		Type: "codeBlock",
		Content: []*Node{
			{Type: "text", Text: codeStr},
		},
	}

	if lang := string(n.Language(c.source)); lang != "" {
		node.Attrs = map[string]interface{}{"language": lang}
	}

	return node
}

func (c *converter) convertCodeBlock(n *ast.CodeBlock) *Node {
	var code strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		code.Write(line.Value(c.source))
	}

	codeStr := strings.TrimSuffix(code.String(), "\n")

	return &Node{
		Type: "codeBlock",
		Content: []*Node{
			{Type: "text", Text: codeStr},
		},
	}
}

func (c *converter) convertBlockquote(n *ast.Blockquote) *Node {
	return &Node{
		Type:    "blockquote",
		Content: c.convertChildren(n),
	}
}

func (c *converter) convertTable(n *extast.Table) *Node {
	var rows []*Node
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

	return &Node{
		Type:    "table",
		Attrs:   map[string]interface{}{"layout": "default"},
		Content: rows,
	}
}

func (c *converter) convertTableHeader(n *extast.TableHeader) *Node {
	var cells []*Node
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if cell, ok := child.(*extast.TableCell); ok {
			cells = append(cells, c.convertTableCell(cell, true))
		}
	}
	return &Node{
		Type:    "tableRow",
		Content: cells,
	}
}

func (c *converter) convertTableRow(n *extast.TableRow, isHeader bool) *Node {
	var cells []*Node
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if cell, ok := child.(*extast.TableCell); ok {
			cells = append(cells, c.convertTableCell(cell, isHeader))
		}
	}
	return &Node{
		Type:    "tableRow",
		Content: cells,
	}
}

func (c *converter) convertTableCell(n *extast.TableCell, isHeader bool) *Node {
	cellType := "tableCell"
	if isHeader {
		cellType = "tableHeader"
	}

	content := c.convertInlineChildren(n)
	var para *Node
	if len(content) > 0 {
		para = &Node{Type: "paragraph", Content: content}
	} else {
		para = &Node{Type: "paragraph", Content: []*Node{{Type: "text", Text: ""}}}
	}

	return &Node{
		Type:    cellType,
		Attrs:   map[string]interface{}{"colspan": 1, "rowspan": 1},
		Content: []*Node{para},
	}
}

// convertInlineChildren converts all inline children of an AST node to ADF text nodes.
func (c *converter) convertInlineChildren(n ast.Node) []*Node {
	var nodes []*Node
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		textNodes := c.convertInlineNode(child, nil)
		nodes = append(nodes, textNodes...)
	}
	return nodes
}

// convertInlineNode converts an inline AST node to ADF text node(s).
func (c *converter) convertInlineNode(n ast.Node, marks []*Mark) []*Node {
	switch node := n.(type) {
	case *ast.Text:
		txt := string(node.Segment.Value(c.source))
		if txt == "" {
			return nil
		}
		adfNode := &Node{Type: "text", Text: txt}
		if len(marks) > 0 {
			adfNode.Marks = marks
		}
		var result []*Node
		result = append(result, adfNode)
		if node.HardLineBreak() {
			result = append(result, &Node{Type: "hardBreak"})
		}
		return result

	case *ast.String:
		txt := string(node.Value)
		if txt == "" {
			return nil
		}
		adfNode := &Node{Type: "text", Text: txt}
		if len(marks) > 0 {
			adfNode.Marks = marks
		}
		return []*Node{adfNode}

	case *ast.Emphasis:
		markType := "em"
		if node.Level == 2 {
			markType = "strong"
		}
		newMarks := append(copyMarks(marks), &Mark{Type: markType})
		var nodes []*Node
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			nodes = append(nodes, c.convertInlineNode(child, newMarks)...)
		}
		return nodes

	case *extast.Strikethrough:
		newMarks := append(copyMarks(marks), &Mark{Type: "strike"})
		var nodes []*Node
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			nodes = append(nodes, c.convertInlineNode(child, newMarks)...)
		}
		return nodes

	case *ast.CodeSpan:
		var textBuilder strings.Builder
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if textNode, ok := child.(*ast.Text); ok {
				textBuilder.Write(textNode.Segment.Value(c.source))
			}
		}
		txt := textBuilder.String()
		newMarks := append(copyMarks(marks), &Mark{Type: "code"})
		return []*Node{{Type: "text", Text: txt, Marks: newMarks}}

	case *ast.Link:
		linkMark := &Mark{
			Type:  "link",
			Attrs: map[string]interface{}{"href": string(node.Destination)},
		}
		newMarks := append(copyMarks(marks), linkMark)
		var nodes []*Node
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			nodes = append(nodes, c.convertInlineNode(child, newMarks)...)
		}
		return nodes

	case *ast.AutoLink:
		url := string(node.URL(c.source))
		linkMark := &Mark{
			Type:  "link",
			Attrs: map[string]interface{}{"href": url},
		}
		newMarks := append(copyMarks(marks), linkMark)
		return []*Node{{Type: "text", Text: url, Marks: newMarks}}

	case *ast.RawHTML:
		return nil

	case *ast.Image:
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
		adfNode := &Node{Type: "text", Text: alt}
		if len(marks) > 0 {
			adfNode.Marks = marks
		}
		return []*Node{adfNode}

	default:
		var nodes []*Node
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			nodes = append(nodes, c.convertInlineNode(child, marks)...)
		}
		return nodes
	}
}

// copyMarks creates a copy of the marks slice.
func copyMarks(marks []*Mark) []*Mark {
	if marks == nil {
		return nil
	}
	result := make([]*Mark, len(marks))
	copy(result, marks)
	return result
}
