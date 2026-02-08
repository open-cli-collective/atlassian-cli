package md

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWikiLink(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected WikiLink
	}{
		{
			name:     "simple page title",
			input:    "My Page",
			expected: WikiLink{Title: "My Page"},
		},
		{
			name:     "page with space key",
			input:    "DEV:My Page",
			expected: WikiLink{SpaceKey: "DEV", Title: "My Page"},
		},
		{
			name:     "page with long space key",
			input:    "ENGINEERING:Architecture Decisions",
			expected: WikiLink{SpaceKey: "ENGINEERING", Title: "Architecture Decisions"},
		},
		{
			name:     "page with spaces trimmed",
			input:    "  My Page  ",
			expected: WikiLink{Title: "My Page"},
		},
		{
			name:     "uppercase prefix is treated as space key",
			input:    "FAQ:How to do things",
			expected: WikiLink{SpaceKey: "FAQ", Title: "How to do things"},
		},
		{
			name:     "lowercase not treated as space key",
			input:    "dev:My Page",
			expected: WikiLink{Title: "dev:My Page"},
		},
		{
			name:     "space key with numbers",
			input:    "TEAM1:Standup Notes",
			expected: WikiLink{SpaceKey: "TEAM1", Title: "Standup Notes"},
		},
		{
			name:     "space key with tilde",
			input:    "~USERSPACE:My Notes",
			expected: WikiLink{SpaceKey: "~USERSPACE", Title: "My Notes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseWikiLink(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSpaceKey(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"DEV", true},
		{"ENGINEERING", true},
		{"TEAM1", true},
		{"A", true},
		{"~USERSPACE", true},
		{"dev", false},      // lowercase
		{"My Space", false}, // spaces
		{"", false},         // empty
		{"FAQ", true},       // uppercase
		{"DEV-OPS", true},   // hyphen
		{"DEV_OPS", true},   // underscore
		{"DEV.OPS", false},  // period
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, isSpaceKey(tt.input))
		})
	}
}

func TestRenderWikiLinkToStorage(t *testing.T) {
	tests := []struct {
		name     string
		wl       WikiLink
		expected string
	}{
		{
			name: "same space link",
			wl:   WikiLink{Title: "My Page"},
			expected: `<ac:link><ri:page ri:content-title="My Page" />` +
				`<ac:plain-text-link-body><![CDATA[My Page]]></ac:plain-text-link-body></ac:link>`,
		},
		{
			name: "cross space link",
			wl:   WikiLink{SpaceKey: "DEV", Title: "My Page"},
			expected: `<ac:link><ri:page ri:content-title="My Page" ri:space-key="DEV" />` +
				`<ac:plain-text-link-body><![CDATA[My Page]]></ac:plain-text-link-body></ac:link>`,
		},
		{
			name: "title with special chars",
			wl:   WikiLink{Title: `Page "with" <special> chars`},
			expected: `<ac:link><ri:page ri:content-title="Page &quot;with&quot; &lt;special&gt; chars" />` +
				`<ac:plain-text-link-body><![CDATA[Page "with" <special> chars]]></ac:plain-text-link-body></ac:link>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderWikiLinkToStorage(tt.wl)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderWikiLinkToBracket(t *testing.T) {
	tests := []struct {
		name     string
		wl       WikiLink
		expected string
	}{
		{
			name:     "same space",
			wl:       WikiLink{Title: "My Page"},
			expected: "[[My Page]]",
		},
		{
			name:     "cross space",
			wl:       WikiLink{SpaceKey: "DEV", Title: "My Page"},
			expected: "[[DEV:My Page]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderWikiLinkToBracket(tt.wl)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreprocessWikiLinks(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedLinks int
		checkOutput   func(t *testing.T, output string, links map[int]WikiLink)
	}{
		{
			name:          "single wiki link",
			input:         "See [[My Page]] for details",
			expectedLinks: 1,
			checkOutput: func(t *testing.T, output string, links map[int]WikiLink) {
				assert.Contains(t, output, "See ")
				assert.Contains(t, output, " for details")
				assert.Contains(t, output, wikiLinkPlaceholderPrefix)
				assert.Equal(t, WikiLink{Title: "My Page"}, links[0])
			},
		},
		{
			name:          "multiple wiki links",
			input:         "See [[Page A]] and [[DEV:Page B]]",
			expectedLinks: 2,
			checkOutput: func(t *testing.T, output string, links map[int]WikiLink) {
				assert.Equal(t, WikiLink{Title: "Page A"}, links[0])
				assert.Equal(t, WikiLink{SpaceKey: "DEV", Title: "Page B"}, links[1])
			},
		},
		{
			name:          "no wiki links",
			input:         "Just regular text",
			expectedLinks: 0,
			checkOutput: func(t *testing.T, output string, links map[int]WikiLink) {
				assert.Equal(t, "Just regular text", output)
			},
		},
		{
			name:          "empty wiki link ignored",
			input:         "See [[]] for details",
			expectedLinks: 0,
			checkOutput: func(t *testing.T, output string, links map[int]WikiLink) {
				assert.Contains(t, output, "[[]]")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, links := preprocessWikiLinks([]byte(tt.input))
			assert.Equal(t, tt.expectedLinks, len(links))
			tt.checkOutput(t, string(output), links)
		})
	}
}

func TestToConfluenceStorage_WikiLinks(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		contains []string
	}{
		{
			name:     "inline wiki link",
			markdown: "See [[My Page]] for details.",
			contains: []string{
				`<ac:link>`,
				`ri:content-title="My Page"`,
				`<![CDATA[My Page]]>`,
				`</ac:link>`,
			},
		},
		{
			name:     "cross-space wiki link",
			markdown: "Check [[DEV:Architecture]] for info.",
			contains: []string{
				`ri:content-title="Architecture"`,
				`ri:space-key="DEV"`,
			},
		},
		{
			name:     "wiki link with macros",
			markdown: "[TOC]\n\nSee [[My Page]] for details.",
			contains: []string{
				`ac:name="toc"`,
				`ri:content-title="My Page"`,
			},
		},
		{
			name:     "multiple wiki links in paragraph",
			markdown: "See [[Page A]] and [[Page B]].",
			contains: []string{
				`ri:content-title="Page A"`,
				`ri:content-title="Page B"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToConfluenceStorage([]byte(tt.markdown))
			require.NoError(t, err)
			for _, s := range tt.contains {
				assert.Contains(t, result, s)
			}
		})
	}
}

func TestToADF_WikiLinks(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		check    func(t *testing.T, jsonStr string)
	}{
		{
			name:     "same-space wiki link produces link mark",
			markdown: "See [[My Page]] for details.",
			check: func(t *testing.T, jsonStr string) {
				assert.Contains(t, jsonStr, `"type":"link"`)
				assert.Contains(t, jsonStr, `confluence-wiki:///My%20Page`)
				assert.Contains(t, jsonStr, `"text":"My Page"`)
			},
		},
		{
			name:     "cross-space wiki link produces link mark with space",
			markdown: "Check [[DEV:Architecture]] for info.",
			check: func(t *testing.T, jsonStr string) {
				assert.Contains(t, jsonStr, `confluence-wiki://DEV/Architecture`)
				assert.Contains(t, jsonStr, `"text":"Architecture"`)
			},
		},
		{
			name:     "wiki link in heading",
			markdown: "# See [[My Page]]",
			check: func(t *testing.T, jsonStr string) {
				assert.Contains(t, jsonStr, `"type":"heading"`)
				assert.Contains(t, jsonStr, `"type":"link"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToADF([]byte(tt.markdown))
			require.NoError(t, err)

			// Verify it's valid JSON
			var doc map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(result), &doc))

			tt.check(t, result)
		})
	}
}

func TestConvertACLinksToPlaceholders(t *testing.T) {
	tests := []struct {
		name          string
		html          string
		expectedLinks int
		checkOutput   func(t *testing.T, output string, links map[int]WikiLink)
	}{
		{
			name: "same-space link",
			html: `<p>See <ac:link><ri:page ri:content-title="My Page" />` +
				`<ac:plain-text-link-body><![CDATA[My Page]]></ac:plain-text-link-body></ac:link> for details.</p>`,
			expectedLinks: 1,
			checkOutput: func(t *testing.T, output string, links map[int]WikiLink) {
				assert.Contains(t, output, wikiLinkFromHTMLPlaceholderPrefix)
				assert.Equal(t, WikiLink{Title: "My Page"}, links[0])
			},
		},
		{
			name: "cross-space link",
			html: `<p>Check <ac:link><ri:page ri:content-title="Architecture" ri:space-key="DEV" />` +
				`<ac:plain-text-link-body><![CDATA[Architecture]]></ac:plain-text-link-body></ac:link></p>`,
			expectedLinks: 1,
			checkOutput: func(t *testing.T, output string, links map[int]WikiLink) {
				assert.Equal(t, WikiLink{SpaceKey: "DEV", Title: "Architecture"}, links[0])
			},
		},
		{
			name: "multiple links",
			html: `<p><ac:link><ri:page ri:content-title="Page A" /></ac:link> and ` +
				`<ac:link><ri:page ri:content-title="Page B" ri:space-key="ENG" /></ac:link></p>`,
			expectedLinks: 2,
			checkOutput: func(t *testing.T, output string, links map[int]WikiLink) {
				assert.Equal(t, WikiLink{Title: "Page A"}, links[0])
				assert.Equal(t, WikiLink{SpaceKey: "ENG", Title: "Page B"}, links[1])
			},
		},
		{
			name: "escaped XML title",
			html: `<ac:link><ri:page ri:content-title="Page &amp; &quot;Stuff&quot;" />` +
				`<ac:plain-text-link-body><![CDATA[Page & "Stuff"]]></ac:plain-text-link-body></ac:link>`,
			expectedLinks: 1,
			checkOutput: func(t *testing.T, output string, links map[int]WikiLink) {
				assert.Equal(t, `Page & "Stuff"`, links[0].Title)
			},
		},
		{
			name:          "no ac:link elements",
			html:          `<p>Just plain HTML</p>`,
			expectedLinks: 0,
			checkOutput: func(t *testing.T, output string, links map[int]WikiLink) {
				assert.Equal(t, `<p>Just plain HTML</p>`, output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, links := convertACLinksToPlaceholders(tt.html)
			assert.Equal(t, tt.expectedLinks, len(links))
			tt.checkOutput(t, output, links)
		})
	}
}

func TestConvertACLinksToMarkdownLinks(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name: "same-space link",
			html: `<p>See <ac:link><ri:page ri:content-title="My Page" />` +
				`<ac:plain-text-link-body><![CDATA[My Page]]></ac:plain-text-link-body></ac:link></p>`,
			expected: `<p>See <a href="#">My Page</a></p>`,
		},
		{
			name: "cross-space link uses title from attribute",
			html: `<p><ac:link><ri:page ri:content-title="Architecture" ri:space-key="DEV" />` +
				`<ac:plain-text-link-body><![CDATA[Architecture]]></ac:plain-text-link-body></ac:link></p>`,
			expected: `<p><a href="#">Architecture</a></p>`,
		},
		{
			name: "multiple links in same paragraph",
			html: `<p><ac:link><ri:page ri:content-title="A" /></ac:link> and ` +
				`<ac:link><ri:page ri:content-title="B" /></ac:link></p>`,
			expected: `<p><a href="#">A</a> and <a href="#">B</a></p>`,
		},
		{
			name:     "no links unchanged",
			html:     `<p>Plain text</p>`,
			expected: `<p>Plain text</p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertACLinksToMarkdownLinks(tt.html)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRoundtrip_WikiLinks_Storage(t *testing.T) {
	// Test: markdown with wiki links → storage → markdown with wiki links
	input := "See [[My Page]] and [[DEV:Architecture]] for details."

	// Convert to storage format
	storage, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, storage, `ri:content-title="My Page"`)
	assert.Contains(t, storage, `ri:content-title="Architecture"`)
	assert.Contains(t, storage, `ri:space-key="DEV"`)

	// Convert back to markdown with --show-macros
	markdown, err := FromConfluenceStorageWithOptions(storage, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)
	assert.Contains(t, markdown, "[[My Page]]")
	assert.Contains(t, markdown, "[[DEV:Architecture]]")
}

func TestRoundtrip_WikiLinks_WithMacros(t *testing.T) {
	// Wiki links + macros should both survive full roundtrip
	input := "[TOC]\n\nSee [[My Page]] for details.\n\n[INFO]\nImportant info about [[DEV:Architecture]]\n[/INFO]"

	// Forward: markdown → storage
	storage, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)

	// Verify both macros and wiki links are present in storage
	assert.Contains(t, storage, `ac:name="toc"`)
	assert.Contains(t, storage, `ri:content-title="My Page"`)
	assert.Contains(t, storage, `ri:content-title="Architecture"`)

	// Reverse: storage → markdown with show-macros
	markdown, err := FromConfluenceStorageWithOptions(storage, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)

	// Verify wiki links survived roundtrip
	assert.Contains(t, markdown, "[[My Page]]")
	assert.Contains(t, markdown, "[[DEV:Architecture]]")
	// Verify macros survived roundtrip
	assert.Contains(t, markdown, "[TOC]")
	assert.Contains(t, markdown, "[INFO]")
	assert.Contains(t, markdown, "[/INFO]")
}

func TestFromConfluenceStorage_WikiLinks_Default(t *testing.T) {
	// Without --show-macros, ac:link should become plain text link
	html := `<p>See <ac:link><ri:page ri:content-title="My Page" />` +
		`<ac:plain-text-link-body><![CDATA[My Page]]></ac:plain-text-link-body></ac:link> for details.</p>`

	result, err := FromConfluenceStorage(html)
	require.NoError(t, err)
	// Should contain the link text (as a markdown link or plain text)
	assert.Contains(t, result, "My Page")
	// Should NOT contain wiki-link syntax
	assert.NotContains(t, result, "[[")
}

func TestFromConfluenceStorage_WikiLinks_ShowMacros(t *testing.T) {
	html := `<p>See <ac:link><ri:page ri:content-title="My Page" />` +
		`<ac:plain-text-link-body><![CDATA[My Page]]></ac:plain-text-link-body></ac:link> for details.</p>`

	result, err := FromConfluenceStorageWithOptions(html, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)
	assert.Contains(t, result, "[[My Page]]")
}

func TestPreprocessWikiLinksForADF(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "same-space link",
			input:    "See [[My Page]] for details",
			expected: "See [My Page](confluence-wiki:///My%20Page) for details",
		},
		{
			name:     "cross-space link",
			input:    "Check [[DEV:Architecture]]",
			expected: "Check [Architecture](confluence-wiki://DEV/Architecture)",
		},
		{
			name:     "no wiki links",
			input:    "Just text",
			expected: "Just text",
		},
		{
			name:     "empty link ignored",
			input:    "See [[]]",
			expected: "See [[]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocessWikiLinksForADF([]byte(tt.input))
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestWikiLink_EscapedTitleInStorage(t *testing.T) {
	// Titles with XML-special characters should be properly escaped in storage format
	wl := WikiLink{Title: `Page & "Stuff" <here>`}
	storage := RenderWikiLinkToStorage(wl)
	assert.Contains(t, storage, `ri:content-title="Page &amp; &quot;Stuff&quot; &lt;here&gt;"`)
	// CDATA doesn't need escaping
	assert.Contains(t, storage, `<![CDATA[Page & "Stuff" <here>]]>`)
}

func TestWikiLink_NotConfusedWithMarkdownLinks(t *testing.T) {
	// Standard markdown links should not be affected
	input := "[regular link](https://example.com)"
	output, links := preprocessWikiLinks([]byte(input))
	assert.Equal(t, 0, len(links))
	assert.Equal(t, input, string(output))
}

func TestWikiLink_NotConfusedWithBracketMacros(t *testing.T) {
	// Bracket macros use single brackets and should not be confused with wiki-links
	input := "[TOC]\n\n[[My Page]]"
	output, links := preprocessWikiLinks([]byte(input))
	assert.Equal(t, 1, len(links))
	assert.Contains(t, string(output), "[TOC]") // single bracket preserved
	assert.Equal(t, WikiLink{Title: "My Page"}, links[0])
}

func TestMultipleWikiLinksInLine(t *testing.T) {
	input := "Compare [[Page A]], [[Page B]], and [[DEV:Page C]]."
	storage, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)

	// All three should be present
	assert.Equal(t, 3, strings.Count(storage, "<ac:link>"))
	assert.Contains(t, storage, `ri:content-title="Page A"`)
	assert.Contains(t, storage, `ri:content-title="Page B"`)
	assert.Contains(t, storage, `ri:content-title="Page C"`)
	assert.Contains(t, storage, `ri:space-key="DEV"`)
}

func TestToADF_WikiLink_SpecialCharsInTitle(t *testing.T) {
	// Titles with special characters should be properly URL-encoded in ADF path
	tests := []struct {
		name     string
		markdown string
		check    func(t *testing.T, jsonStr string)
	}{
		{
			name:     "title with ampersand",
			markdown: "See [[Q&A Page]] here.",
			check: func(t *testing.T, jsonStr string) {
				assert.Contains(t, jsonStr, `"type":"link"`)
				assert.Contains(t, jsonStr, `"text":"Q\u0026A Page"`)
			},
		},
		{
			name:     "title with special URL chars",
			markdown: "Check [[Page #1 (draft)]].",
			check: func(t *testing.T, jsonStr string) {
				assert.Contains(t, jsonStr, `"type":"link"`)
				assert.Contains(t, jsonStr, `"text":"Page #1 (draft)"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToADF([]byte(tt.markdown))
			require.NoError(t, err)

			var doc map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(result), &doc))

			tt.check(t, result)
		})
	}
}
