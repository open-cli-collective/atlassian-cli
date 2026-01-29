package md

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderMacroToXML_SimpleTOC(t *testing.T) {
	node := &MacroNode{Name: "toc"}
	xml := RenderMacroToXML(node)

	assert.Contains(t, xml, `ac:name="toc"`)
	assert.Contains(t, xml, `ac:schema-version="1"`)
	assert.Contains(t, xml, `</ac:structured-macro>`)
}

func TestRenderMacroToXML_TOCWithParams(t *testing.T) {
	node := &MacroNode{
		Name:       "toc",
		Parameters: map[string]string{"maxLevel": "3", "minLevel": "1"},
	}
	xml := RenderMacroToXML(node)

	assert.Contains(t, xml, `<ac:parameter ac:name="maxLevel">3</ac:parameter>`)
	assert.Contains(t, xml, `<ac:parameter ac:name="minLevel">1</ac:parameter>`)
}

func TestRenderMacroToXML_PanelWithBody(t *testing.T) {
	node := &MacroNode{
		Name: "info",
		Body: "<p>Content</p>",
	}
	xml := RenderMacroToXML(node)

	assert.Contains(t, xml, `ac:name="info"`)
	assert.Contains(t, xml, `<ac:rich-text-body>`)
	assert.Contains(t, xml, `<p>Content</p>`)
	assert.Contains(t, xml, `</ac:rich-text-body>`)
}

func TestRenderMacroToXML_CodeWithCDATA(t *testing.T) {
	node := &MacroNode{
		Name:       "code",
		Parameters: map[string]string{"language": "go"},
		Body:       `fmt.Println("hello")`,
	}
	xml := RenderMacroToXML(node)

	assert.Contains(t, xml, `ac:name="code"`)
	assert.Contains(t, xml, `<ac:plain-text-body><![CDATA[`)
	assert.Contains(t, xml, `fmt.Println("hello")`)
	assert.Contains(t, xml, `]]></ac:plain-text-body>`)
}

func TestRenderMacroToXML_EscapesXML(t *testing.T) {
	node := &MacroNode{
		Name:       "toc",
		Parameters: map[string]string{"title": "A & B <test>"},
	}
	xml := RenderMacroToXML(node)

	assert.Contains(t, xml, `A &amp; B &lt;test&gt;`)
}

func TestRenderMacroToBracket_SimpleTOC(t *testing.T) {
	node := &MacroNode{Name: "toc"}
	bracket := RenderMacroToBracket(node)

	assert.Equal(t, "[TOC]", bracket)
}

func TestRenderMacroToBracket_TOCWithParams(t *testing.T) {
	node := &MacroNode{
		Name:       "toc",
		Parameters: map[string]string{"maxLevel": "3"},
	}
	bracket := RenderMacroToBracket(node)

	assert.Contains(t, bracket, "[TOC")
	assert.Contains(t, bracket, "maxLevel=3")
	assert.Contains(t, bracket, "]")
}

func TestRenderMacroToBracket_PanelWithBody(t *testing.T) {
	node := &MacroNode{
		Name:       "info",
		Parameters: map[string]string{"title": "Important"},
		Body:       "Content here",
	}
	bracket := RenderMacroToBracket(node)

	assert.Contains(t, bracket, "[INFO")
	assert.Contains(t, bracket, "title=Important")
	assert.Contains(t, bracket, "Content here")
	assert.Contains(t, bracket, "[/INFO]")
}

func TestRenderMacroToBracket_QuotedValues(t *testing.T) {
	node := &MacroNode{
		Name:       "info",
		Parameters: map[string]string{"title": "Hello World"},
	}
	bracket := RenderMacroToBracket(node)

	assert.Contains(t, bracket, `title="Hello World"`)
}

func TestRenderMacroToBracketOpen_SimpleTOC(t *testing.T) {
	node := &MacroNode{Name: "toc"}
	bracket := RenderMacroToBracketOpen(node)
	assert.Equal(t, "[TOC]", bracket)
}

func TestRenderMacroToBracketOpen_WithParams(t *testing.T) {
	node := &MacroNode{
		Name:       "info",
		Parameters: map[string]string{"title": "Hello World"},
	}
	bracket := RenderMacroToBracketOpen(node)
	assert.Contains(t, bracket, "[INFO")
	assert.Contains(t, bracket, `title="Hello World"`)
	assert.True(t, strings.HasSuffix(bracket, "]"))
}

func TestFormatPlaceholder(t *testing.T) {
	assert.Equal(t, "CFMACRO0END", FormatPlaceholder(0))
	assert.Equal(t, "CFMACRO42END", FormatPlaceholder(42))
}
