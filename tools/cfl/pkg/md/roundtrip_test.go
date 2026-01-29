package md

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoundtrip verifies that macros survive MD→XHTML→MD conversion.
func TestRoundtrip_TOC(t *testing.T) {
	input := "[TOC maxLevel=3]"

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, xhtml, `ac:name="toc"`)
	assert.Contains(t, xhtml, `maxLevel`)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)
	assert.Contains(t, strings.ToUpper(md), "[TOC")
	assert.Contains(t, md, "maxLevel")
}

func TestRoundtrip_InfoPanel(t *testing.T) {
	input := `[INFO title="Important"]
This is important content.
[/INFO]`

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, xhtml, `ac:name="info"`)
	assert.Contains(t, xhtml, `<ac:rich-text-body>`)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)
	assert.Contains(t, strings.ToUpper(md), "[INFO")
	assert.Contains(t, md, "[/INFO]")
	assert.Contains(t, md, "important content")
}

func TestRoundtrip_NestedMacros(t *testing.T) {
	input := `[INFO]
Content with [TOC] inside.
[/INFO]`

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, xhtml, `ac:name="info"`)
	assert.Contains(t, xhtml, `ac:name="toc"`)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)
	assert.Contains(t, strings.ToUpper(md), "[INFO")
	assert.Contains(t, strings.ToUpper(md), "[TOC")
}

func TestRoundtrip_AllPanelTypes(t *testing.T) {
	panelTypes := []string{"INFO", "WARNING", "NOTE", "TIP", "EXPAND"}

	for _, pt := range panelTypes {
		t.Run(pt, func(t *testing.T) {
			input := "[" + pt + "]Content[/" + pt + "]"

			xhtml, err := ToConfluenceStorage([]byte(input))
			require.NoError(t, err)
			assert.Contains(t, xhtml, `ac:name="`+strings.ToLower(pt)+`"`)

			md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
			require.NoError(t, err)
			assert.Contains(t, strings.ToUpper(md), "["+pt)
			assert.Contains(t, strings.ToUpper(md), "[/"+pt+"]")
		})
	}
}

// TestRoundtrip_NestedPosition verifies that nested macro position is preserved
// through the complete MD→XHTML→MD cycle.
func TestRoundtrip_NestedPosition(t *testing.T) {
	input := `[INFO]
Before
[TOC]
After
[/INFO]`

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)

	// Verify order is preserved: Before < TOC < After
	beforeIdx := strings.Index(md, "Before")
	tocIdx := strings.Index(strings.ToUpper(md), "[TOC")
	afterIdx := strings.Index(md, "After")

	assert.True(t, beforeIdx >= 0, "Before should be present")
	assert.True(t, tocIdx >= 0, "TOC should be present")
	assert.True(t, afterIdx >= 0, "After should be present")

	assert.True(t, beforeIdx < tocIdx, "Before should come before TOC")
	assert.True(t, tocIdx < afterIdx, "TOC should come before After")
}

func TestRoundtrip_MultipleNestedMacros(t *testing.T) {
	input := `[INFO]
Start
[TOC]
Middle
[TOC maxLevel=2]
End
[/INFO]`

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)

	// All text and macros should be present
	assert.Contains(t, md, "Start")
	assert.Contains(t, md, "Middle")
	assert.Contains(t, md, "End")
	assert.Contains(t, strings.ToUpper(md), "[TOC")

	// Verify order
	startIdx := strings.Index(md, "Start")
	middleIdx := strings.Index(md, "Middle")
	endIdx := strings.Index(md, "End")

	assert.True(t, startIdx < middleIdx, "Start should come before Middle")
	assert.True(t, middleIdx < endIdx, "Middle should come before End")
}

func TestRoundtrip_DeeplyNested(t *testing.T) {
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
	require.NoError(t, err)

	// Verify nesting in XHTML
	assert.Contains(t, xhtml, `ac:name="info"`)
	assert.Contains(t, xhtml, `ac:name="warning"`)
	assert.Contains(t, xhtml, `ac:name="toc"`)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)

	// All elements should be present
	assert.Contains(t, md, "Outer")
	assert.Contains(t, strings.ToUpper(md), "[INFO")
	assert.Contains(t, strings.ToUpper(md), "[WARNING")
	assert.Contains(t, strings.ToUpper(md), "[TOC")
	assert.Contains(t, md, "Inner")
}

// TestRoundtrip_CloseTagNotDuplicated verifies that panel content appears exactly once
// through the MD→XHTML→MD cycle (close tag is properly consumed, not left as literal text).
func TestRoundtrip_CloseTagNotDuplicated(t *testing.T) {
	input := "[INFO]unique content[/INFO]"

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)

	// Content should appear exactly once in XHTML
	assert.Equal(t, 1, strings.Count(xhtml, "unique content"))

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)

	// Content should appear exactly once in MD
	assert.Equal(t, 1, strings.Count(md, "unique content"))
}

// TestRoundtrip_NestedMacroInParagraph verifies the issue #56 fix.
// Tests that nested self-closing macros survive the MD→XHTML→MD cycle
// even when the XHTML wraps them in <p> tags.
func TestRoundtrip_NestedMacroInParagraph(t *testing.T) {
	// Start with markdown containing nested macro
	input := "[INFO]\n\n[TOC]\n\n[/INFO]\n\n# Header 1"

	// Convert to XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)

	// Verify XHTML has correct structure
	assert.Contains(t, xhtml, "ac:structured-macro")
	assert.Contains(t, xhtml, `ac:name="info"`)
	assert.Contains(t, xhtml, `ac:name="toc"`)

	// Convert back to markdown
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)

	// Verify structure preserved (case-insensitive check for macro names)
	assert.Contains(t, strings.ToUpper(md), "[INFO]")
	assert.Contains(t, strings.ToUpper(md), "[TOC]")
	assert.Contains(t, strings.ToUpper(md), "[/INFO]")
	assert.Contains(t, md, "# Header 1")

	// Verify nesting order is preserved
	infoStart := strings.Index(strings.ToUpper(md), "[INFO]")
	tocPos := strings.Index(strings.ToUpper(md), "[TOC]")
	infoEnd := strings.Index(strings.ToUpper(md), "[/INFO]")

	assert.True(t, infoStart < tocPos, "[INFO] should come before [TOC]")
	assert.True(t, tocPos < infoEnd, "[TOC] should come before [/INFO]")
}
