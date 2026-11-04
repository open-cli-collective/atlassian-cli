package md

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

// TestRoundtrip verifies that macros survive MD→XHTML→MD conversion.
func TestRoundtrip_TOC(t *testing.T) {
	t.Parallel()
	input := "[TOC maxLevel=3]"

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)
	testutil.Contains(t, xhtml, `ac:name="toc"`)
	testutil.Contains(t, xhtml, `maxLevel`)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	testutil.RequireNoError(t, err)
	testutil.Contains(t, strings.ToUpper(md), "[TOC")
	testutil.Contains(t, md, "maxLevel")
}

func TestRoundtrip_InfoPanel(t *testing.T) {
	t.Parallel()
	input := `[INFO title="Important"]
This is important content.
[/INFO]`

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)
	testutil.Contains(t, xhtml, `ac:name="info"`)
	testutil.Contains(t, xhtml, `<ac:rich-text-body>`)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	testutil.RequireNoError(t, err)
	testutil.Contains(t, strings.ToUpper(md), "[INFO")
	testutil.Contains(t, md, "[/INFO]")
	testutil.Contains(t, md, "important content")
}

func TestRoundtrip_NestedMacros(t *testing.T) {
	t.Parallel()
	input := `[INFO]
Content with [TOC] inside.
[/INFO]`

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)
	testutil.Contains(t, xhtml, `ac:name="info"`)
	testutil.Contains(t, xhtml, `ac:name="toc"`)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	testutil.RequireNoError(t, err)
	testutil.Contains(t, strings.ToUpper(md), "[INFO")
	testutil.Contains(t, strings.ToUpper(md), "[TOC")
}

func TestRoundtrip_AllPanelTypes(t *testing.T) {
	t.Parallel()
	panelTypes := []string{"INFO", "WARNING", "NOTE", "TIP", "EXPAND"}

	for _, pt := range panelTypes {
		t.Run(pt, func(t *testing.T) {
			t.Parallel()
			input := "[" + pt + "]Content[/" + pt + "]"

			xhtml, err := ToConfluenceStorage([]byte(input))
			testutil.RequireNoError(t, err)
			testutil.Contains(t, xhtml, `ac:name="`+strings.ToLower(pt)+`"`)

			md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
			testutil.RequireNoError(t, err)
			testutil.Contains(t, strings.ToUpper(md), "["+pt)
			testutil.Contains(t, strings.ToUpper(md), "[/"+pt+"]")
		})
	}
}

// TestRoundtrip_NestedPosition verifies that nested macro position is preserved
// through the complete MD→XHTML→MD cycle.
func TestRoundtrip_NestedPosition(t *testing.T) {
	t.Parallel()
	input := `[INFO]
Before
[TOC]
After
[/INFO]`

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	testutil.RequireNoError(t, err)

	// Verify order is preserved: Before < TOC < After
	beforeIdx := strings.Index(md, "Before")
	tocIdx := strings.Index(strings.ToUpper(md), "[TOC")
	afterIdx := strings.Index(md, "After")

	testutil.True(t, beforeIdx >= 0, "Before should be present")
	testutil.True(t, tocIdx >= 0, "TOC should be present")
	testutil.True(t, afterIdx >= 0, "After should be present")

	testutil.True(t, beforeIdx < tocIdx, "Before should come before TOC")
	testutil.True(t, tocIdx < afterIdx, "TOC should come before After")
}

func TestRoundtrip_MultipleNestedMacros(t *testing.T) {
	t.Parallel()
	input := `[INFO]
Start
[TOC]
Middle
[TOC maxLevel=2]
End
[/INFO]`

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	testutil.RequireNoError(t, err)

	// All text and macros should be present
	testutil.Contains(t, md, "Start")
	testutil.Contains(t, md, "Middle")
	testutil.Contains(t, md, "End")
	testutil.Contains(t, strings.ToUpper(md), "[TOC")

	// Verify order
	startIdx := strings.Index(md, "Start")
	middleIdx := strings.Index(md, "Middle")
	endIdx := strings.Index(md, "End")

	testutil.True(t, startIdx < middleIdx, "Start should come before Middle")
	testutil.True(t, middleIdx < endIdx, "Middle should come before End")
}

func TestRoundtrip_DeeplyNested(t *testing.T) {
	t.Parallel()
	input := `[INFO]
Outer
[WARNING]
Inner
[TOC]
More inner
[/WARNING]
More outer
[/INFO]`

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)

	// Verify nesting in XHTML
	testutil.Contains(t, xhtml, `ac:name="info"`)
	testutil.Contains(t, xhtml, `ac:name="warning"`)
	testutil.Contains(t, xhtml, `ac:name="toc"`)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	testutil.RequireNoError(t, err)

	// All elements should be present
	testutil.Contains(t, md, "Outer")
	testutil.Contains(t, strings.ToUpper(md), "[INFO")
	testutil.Contains(t, strings.ToUpper(md), "[WARNING")
	testutil.Contains(t, strings.ToUpper(md), "[TOC")
	testutil.Contains(t, md, "Inner")
}

// TestRoundtrip_CloseTagNotDuplicated verifies that panel content appears exactly once
// through the MD→XHTML→MD cycle (close tag is properly consumed, not left as literal text).
func TestRoundtrip_CloseTagNotDuplicated(t *testing.T) {
	t.Parallel()
	input := "[INFO]unique content[/INFO]"

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)

	// Content should appear exactly once in XHTML
	testutil.Equal(t, 1, strings.Count(xhtml, "unique content"))

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	testutil.RequireNoError(t, err)

	// Content should appear exactly once in MD
	testutil.Equal(t, 1, strings.Count(md, "unique content"))
}

// TestRoundtrip_NestedMacroInParagraph verifies the issue #56 fix.
// Tests that nested self-closing macros survive the MD→XHTML→MD cycle
// even when the XHTML wraps them in <p> tags.
func TestRoundtrip_NestedMacroInParagraph(t *testing.T) {
	t.Parallel()
	// Start with markdown containing nested macro
	input := "[INFO]\n\n[TOC]\n\n[/INFO]\n\n# Header 1"

	// Convert to XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)

	// Verify XHTML has correct structure
	testutil.Contains(t, xhtml, "ac:structured-macro")
	testutil.Contains(t, xhtml, `ac:name="info"`)
	testutil.Contains(t, xhtml, `ac:name="toc"`)

	// Convert back to markdown
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	testutil.RequireNoError(t, err)

	// Verify structure preserved (case-insensitive check for macro names)
	testutil.Contains(t, strings.ToUpper(md), "[INFO]")
	testutil.Contains(t, strings.ToUpper(md), "[TOC]")
	testutil.Contains(t, strings.ToUpper(md), "[/INFO]")
	testutil.Contains(t, md, "# Header 1")

	// Verify nesting order is preserved
	infoStart := strings.Index(strings.ToUpper(md), "[INFO]")
	tocPos := strings.Index(strings.ToUpper(md), "[TOC]")
	infoEnd := strings.Index(strings.ToUpper(md), "[/INFO]")

	testutil.True(t, infoStart < tocPos, "[INFO] should come before [TOC]")
	testutil.True(t, tocPos < infoEnd, "[TOC] should come before [/INFO]")
}

// TestRoundtrip_PlainTextThroughStorageRead verifies that a page read as markdown
// and written back keeps its text: the read escapes markdown punctuation and the
// write has to decode it, or every round trip adds visible backslashes.
func TestRoundtrip_PlainTextThroughStorageRead(t *testing.T) {
	t.Parallel()
	storage := `<p>:rotating_light: a_b_c *star* [b] A &gt; B 1 &amp; 2 &lt;tag&gt; back\slash</p>`
	want := `:rotating_light: a_b_c *star* [b] A > B 1 & 2 <tag> back\slash`

	markdown, err := FromConfluenceStorage(storage)
	testutil.RequireNoError(t, err)
	adfJSON, err := ToADF([]byte(markdown))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	testutil.RequireNoError(t, json.Unmarshal([]byte(adfJSON), &doc))
	var got strings.Builder
	for _, n := range doc.Content[0].Content {
		got.WriteString(n.Text)
	}
	testutil.Equal(t, got.String(), want)
}

// A page that shows an entity literally keeps it across a read and write, while the
// character that the same markdown text would otherwise decode to stays a character.
func TestRoundtrip_LiteralEntityThroughStorageRead(t *testing.T) {
	t.Parallel()
	storage := "<p>&amp;gt; &amp;amp; R&amp;D a_b &gt; c `&amp;gt;` <code>&amp;gt;</code></p>"
	want := "&gt; &amp; R&D a_b > c `&gt;` &gt;"

	markdown, err := FromConfluenceStorage(storage)
	testutil.RequireNoError(t, err)
	adfJSON, err := ToADF([]byte(markdown))
	testutil.RequireNoError(t, err)

	var doc ADFDocument
	testutil.RequireNoError(t, json.Unmarshal([]byte(adfJSON), &doc))
	var got strings.Builder
	for _, n := range doc.Content[0].Content {
		got.WriteString(n.Text)
	}
	testutil.Equal(t, got.String(), want)
}

// Only text nodes are marked: a link keeps its query string as it was stored.
func TestFromConfluenceStorage_LiteralEntityMarkerStaysOutOfLinks(t *testing.T) {
	t.Parallel()
	markdown, err := FromConfluenceStorage(`<p><a href="http://x.io/?a=1&amp;b;c=2">l</a></p>`)
	testutil.RequireNoError(t, err)
	testutil.Equal(t, strings.Contains(markdown, "%EE%B7%90"), false)
	testutil.Equal(t, strings.Contains(markdown, literalAmpersand), false)
	testutil.Contains(t, markdown, "a=1")
}
