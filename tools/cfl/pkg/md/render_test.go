package md

import (
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestRenderMacroToXML_SimpleTOC(t *testing.T) {
	t.Parallel()
	node := &MacroNode{Name: "toc"}
	xml := RenderMacroToXML(node)

	testutil.Contains(t, xml, `ac:name="toc"`)
	testutil.Contains(t, xml, `ac:schema-version="1"`)
	testutil.Contains(t, xml, `</ac:structured-macro>`)
}

func TestRenderMacroToXML_TOCWithParams(t *testing.T) {
	t.Parallel()
	node := &MacroNode{
		Name:       "toc",
		Parameters: map[string]string{"maxLevel": "3", "minLevel": "1"},
	}
	xml := RenderMacroToXML(node)

	testutil.Contains(t, xml, `<ac:parameter ac:name="maxLevel">3</ac:parameter>`)
	testutil.Contains(t, xml, `<ac:parameter ac:name="minLevel">1</ac:parameter>`)
}

func TestRenderMacroToXML_PanelWithBody(t *testing.T) {
	t.Parallel()
	node := &MacroNode{
		Name: "info",
		Body: "<p>Content</p>",
	}
	xml := RenderMacroToXML(node)

	testutil.Contains(t, xml, `ac:name="info"`)
	testutil.Contains(t, xml, `<ac:rich-text-body>`)
	testutil.Contains(t, xml, `<p>Content</p>`)
	testutil.Contains(t, xml, `</ac:rich-text-body>`)
}

func TestRenderMacroToXML_CodeWithCDATA(t *testing.T) {
	t.Parallel()
	node := &MacroNode{
		Name:       "code",
		Parameters: map[string]string{"language": "go"},
		Body:       `fmt.Println("hello")`,
	}
	xml := RenderMacroToXML(node)

	testutil.Contains(t, xml, `ac:name="code"`)
	testutil.Contains(t, xml, `<ac:plain-text-body><![CDATA[`)
	testutil.Contains(t, xml, `fmt.Println("hello")`)
	testutil.Contains(t, xml, `]]></ac:plain-text-body>`)
}

func TestRenderMacroToXML_EscapesXML(t *testing.T) {
	t.Parallel()
	node := &MacroNode{
		Name:       "toc",
		Parameters: map[string]string{"title": "A & B <test>"},
	}
	xml := RenderMacroToXML(node)

	testutil.Contains(t, xml, `A &amp; B &lt;test&gt;`)
}

func TestRenderMacroToBracket_SimpleTOC(t *testing.T) {
	t.Parallel()
	node := &MacroNode{Name: "toc"}
	bracket := RenderMacroToBracket(node)

	testutil.Equal(t, "[TOC]", bracket)
}

func TestRenderMacroToBracket_TOCWithParams(t *testing.T) {
	t.Parallel()
	node := &MacroNode{
		Name:       "toc",
		Parameters: map[string]string{"maxLevel": "3"},
	}
	bracket := RenderMacroToBracket(node)

	testutil.Contains(t, bracket, "[TOC")
	testutil.Contains(t, bracket, "maxLevel=3")
	testutil.Contains(t, bracket, "]")
}

func TestRenderMacroToBracket_PanelWithBody(t *testing.T) {
	t.Parallel()
	node := &MacroNode{
		Name:       "info",
		Parameters: map[string]string{"title": "Important"},
		Body:       "Content here",
	}
	bracket := RenderMacroToBracket(node)

	testutil.Contains(t, bracket, "[INFO")
	testutil.Contains(t, bracket, "title=Important")
	testutil.Contains(t, bracket, "Content here")
	testutil.Contains(t, bracket, "[/INFO]")
}

func TestRenderMacroToBracket_QuotedValues(t *testing.T) {
	t.Parallel()
	node := &MacroNode{
		Name:       "info",
		Parameters: map[string]string{"title": "Hello World"},
	}
	bracket := RenderMacroToBracket(node)

	testutil.Contains(t, bracket, `title="Hello World"`)
}

func TestRenderMacroToBracketOpen_SimpleTOC(t *testing.T) {
	t.Parallel()
	node := &MacroNode{Name: "toc"}
	bracket := RenderMacroToBracketOpen(node)
	testutil.Equal(t, "[TOC]", bracket)
}

func TestRenderMacroToBracketOpen_WithParams(t *testing.T) {
	t.Parallel()
	node := &MacroNode{
		Name:       "info",
		Parameters: map[string]string{"title": "Hello World"},
	}
	bracket := RenderMacroToBracketOpen(node)
	testutil.Contains(t, bracket, "[INFO")
	testutil.Contains(t, bracket, `title="Hello World"`)
	testutil.True(t, strings.HasSuffix(bracket, "]"))
}

func TestFormatPlaceholder(t *testing.T) {
	t.Parallel()
	testutil.Equal(t, "CFMACRO0END", FormatPlaceholder(0))
	testutil.Equal(t, "CFMACRO42END", FormatPlaceholder(42))
}
