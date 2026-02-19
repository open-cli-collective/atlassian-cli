package md

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestParseWikiLink(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := ParseWikiLink(tt.input)
			testutil.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSpaceKey(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			testutil.Equal(t, tt.expected, isSpaceKey(tt.input))
		})
	}
}

func TestRenderWikiLinkToStorage(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := RenderWikiLinkToStorage(tt.wl)
			testutil.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderWikiLinkToBracket(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := RenderWikiLinkToBracket(tt.wl)
			testutil.Equal(t, tt.expected, result)
		})
	}
}

func TestPreprocessWikiLinks(t *testing.T) {
	t.Parallel()
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
				testutil.Contains(t, output, "See ")
				testutil.Contains(t, output, " for details")
				testutil.Contains(t, output, wikiLinkPlaceholderPrefix)
				testutil.Equal(t, WikiLink{Title: "My Page"}, links[0])
			},
		},
		{
			name:          "multiple wiki links",
			input:         "See [[Page A]] and [[DEV:Page B]]",
			expectedLinks: 2,
			checkOutput: func(t *testing.T, _ string, links map[int]WikiLink) {
				testutil.Equal(t, WikiLink{Title: "Page A"}, links[0])
				testutil.Equal(t, WikiLink{SpaceKey: "DEV", Title: "Page B"}, links[1])
			},
		},
		{
			name:          "no wiki links",
			input:         "Just regular text",
			expectedLinks: 0,
			checkOutput: func(t *testing.T, output string, _ map[int]WikiLink) {
				testutil.Equal(t, "Just regular text", output)
			},
		},
		{
			name:          "empty wiki link ignored",
			input:         "See [[]] for details",
			expectedLinks: 0,
			checkOutput: func(t *testing.T, output string, _ map[int]WikiLink) {
				testutil.Contains(t, output, "[[]]")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, links := preprocessWikiLinks([]byte(tt.input))
			testutil.Equal(t, tt.expectedLinks, len(links))
			tt.checkOutput(t, string(output), links)
		})
	}
}

func TestToConfluenceStorage_WikiLinks(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result, err := ToConfluenceStorage([]byte(tt.markdown))
			testutil.RequireNoError(t, err)
			for _, s := range tt.contains {
				testutil.Contains(t, result, s)
			}
		})
	}
}

func TestToADF_WikiLinks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		markdown string
		check    func(t *testing.T, jsonStr string)
	}{
		{
			name:     "same-space wiki link produces link mark",
			markdown: "See [[My Page]] for details.",
			check: func(t *testing.T, jsonStr string) {
				testutil.Contains(t, jsonStr, `"type":"link"`)
				testutil.Contains(t, jsonStr, `confluence-wiki:///My%20Page`)
				testutil.Contains(t, jsonStr, `"text":"My Page"`)
			},
		},
		{
			name:     "cross-space wiki link produces link mark with space",
			markdown: "Check [[DEV:Architecture]] for info.",
			check: func(t *testing.T, jsonStr string) {
				testutil.Contains(t, jsonStr, `confluence-wiki://DEV/Architecture`)
				testutil.Contains(t, jsonStr, `"text":"Architecture"`)
			},
		},
		{
			name:     "wiki link in heading",
			markdown: "# See [[My Page]]",
			check: func(t *testing.T, jsonStr string) {
				testutil.Contains(t, jsonStr, `"type":"heading"`)
				testutil.Contains(t, jsonStr, `"type":"link"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ToADF([]byte(tt.markdown))
			testutil.RequireNoError(t, err)

			// Verify it's valid JSON
			var doc map[string]any
			testutil.RequireNoError(t, json.Unmarshal([]byte(result), &doc))

			tt.check(t, result)
		})
	}
}

func TestConvertACLinksToPlaceholders(t *testing.T) {
	t.Parallel()
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
				testutil.Contains(t, output, wikiLinkFromHTMLPlaceholderPrefix)
				testutil.Equal(t, WikiLink{Title: "My Page"}, links[0])
			},
		},
		{
			name: "cross-space link",
			html: `<p>Check <ac:link><ri:page ri:content-title="Architecture" ri:space-key="DEV" />` +
				`<ac:plain-text-link-body><![CDATA[Architecture]]></ac:plain-text-link-body></ac:link></p>`,
			expectedLinks: 1,
			checkOutput: func(t *testing.T, _ string, links map[int]WikiLink) {
				testutil.Equal(t, WikiLink{SpaceKey: "DEV", Title: "Architecture"}, links[0])
			},
		},
		{
			name: "multiple links",
			html: `<p><ac:link><ri:page ri:content-title="Page A" /></ac:link> and ` +
				`<ac:link><ri:page ri:content-title="Page B" ri:space-key="ENG" /></ac:link></p>`,
			expectedLinks: 2,
			checkOutput: func(t *testing.T, _ string, links map[int]WikiLink) {
				testutil.Equal(t, WikiLink{Title: "Page A"}, links[0])
				testutil.Equal(t, WikiLink{SpaceKey: "ENG", Title: "Page B"}, links[1])
			},
		},
		{
			name: "escaped XML title",
			html: `<ac:link><ri:page ri:content-title="Page &amp; &quot;Stuff&quot;" />` +
				`<ac:plain-text-link-body><![CDATA[Page & "Stuff"]]></ac:plain-text-link-body></ac:link>`,
			expectedLinks: 1,
			checkOutput: func(t *testing.T, _ string, links map[int]WikiLink) {
				testutil.Equal(t, `Page & "Stuff"`, links[0].Title)
			},
		},
		{
			name:          "no ac:link elements",
			html:          `<p>Just plain HTML</p>`,
			expectedLinks: 0,
			checkOutput: func(t *testing.T, output string, _ map[int]WikiLink) {
				testutil.Equal(t, `<p>Just plain HTML</p>`, output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, links := convertACLinksToPlaceholders(tt.html)
			testutil.Equal(t, tt.expectedLinks, len(links))
			tt.checkOutput(t, output, links)
		})
	}
}

func TestConvertACLinksToMarkdownLinks(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := convertACLinksToMarkdownLinks(tt.html)
			testutil.Equal(t, tt.expected, result)
		})
	}
}

func TestRoundtrip_WikiLinks_Storage(t *testing.T) {
	t.Parallel()
	// Test: markdown with wiki links → storage → markdown with wiki links
	input := "See [[My Page]] and [[DEV:Architecture]] for details."

	// Convert to storage format
	storage, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)
	testutil.Contains(t, storage, `ri:content-title="My Page"`)
	testutil.Contains(t, storage, `ri:content-title="Architecture"`)
	testutil.Contains(t, storage, `ri:space-key="DEV"`)

	// Convert back to markdown with --show-macros
	markdown, err := FromConfluenceStorageWithOptions(storage, ConvertOptions{ShowMacros: true})
	testutil.RequireNoError(t, err)
	testutil.Contains(t, markdown, "[[My Page]]")
	testutil.Contains(t, markdown, "[[DEV:Architecture]]")
}

func TestRoundtrip_WikiLinks_WithMacros(t *testing.T) {
	t.Parallel()
	// Wiki links + macros should both survive full roundtrip
	input := "[TOC]\n\nSee [[My Page]] for details.\n\n[INFO]\nImportant info about [[DEV:Architecture]]\n[/INFO]"

	// Forward: markdown → storage
	storage, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)

	// Verify both macros and wiki links are present in storage
	testutil.Contains(t, storage, `ac:name="toc"`)
	testutil.Contains(t, storage, `ri:content-title="My Page"`)
	testutil.Contains(t, storage, `ri:content-title="Architecture"`)

	// Reverse: storage → markdown with show-macros
	markdown, err := FromConfluenceStorageWithOptions(storage, ConvertOptions{ShowMacros: true})
	testutil.RequireNoError(t, err)

	// Verify wiki links survived roundtrip
	testutil.Contains(t, markdown, "[[My Page]]")
	testutil.Contains(t, markdown, "[[DEV:Architecture]]")
	// Verify macros survived roundtrip
	testutil.Contains(t, markdown, "[TOC]")
	testutil.Contains(t, markdown, "[INFO]")
	testutil.Contains(t, markdown, "[/INFO]")
}

func TestFromConfluenceStorage_WikiLinks_Default(t *testing.T) {
	t.Parallel()
	// Without --show-macros, ac:link should become plain text link
	html := `<p>See <ac:link><ri:page ri:content-title="My Page" />` +
		`<ac:plain-text-link-body><![CDATA[My Page]]></ac:plain-text-link-body></ac:link> for details.</p>`

	result, err := FromConfluenceStorage(html)
	testutil.RequireNoError(t, err)
	// Should contain the link text (as a markdown link or plain text)
	testutil.Contains(t, result, "My Page")
	// Should NOT contain wiki-link syntax
	testutil.NotContains(t, result, "[[")
}

func TestFromConfluenceStorage_WikiLinks_ShowMacros(t *testing.T) {
	t.Parallel()
	html := `<p>See <ac:link><ri:page ri:content-title="My Page" />` +
		`<ac:plain-text-link-body><![CDATA[My Page]]></ac:plain-text-link-body></ac:link> for details.</p>`

	result, err := FromConfluenceStorageWithOptions(html, ConvertOptions{ShowMacros: true})
	testutil.RequireNoError(t, err)
	testutil.Contains(t, result, "[[My Page]]")
}

func TestPreprocessWikiLinksForADF(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			result := preprocessWikiLinksForADF([]byte(tt.input))
			testutil.Equal(t, tt.expected, string(result))
		})
	}
}

func TestPreprocessWikiLinks_CodeBlockProtection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		input         string
		expectedLinks int
		checkOutput   func(t *testing.T, output string, links map[int]WikiLink)
	}{
		{
			name:          "wiki link in fenced code block not converted",
			input:         "```\n[[My Page]]\n```",
			expectedLinks: 0,
			checkOutput: func(t *testing.T, output string, _ map[int]WikiLink) {
				testutil.Contains(t, output, "[[My Page]]")
			},
		},
		{
			name:          "wiki link in inline code not converted",
			input:         "Use `[[My Page]]` syntax",
			expectedLinks: 0,
			checkOutput: func(t *testing.T, output string, _ map[int]WikiLink) {
				testutil.Contains(t, output, "`[[My Page]]`")
			},
		},
		{
			name:          "wiki link outside code block still converted",
			input:         "```\n[[Code Page]]\n```\n\nSee [[Real Page]] here.",
			expectedLinks: 1,
			checkOutput: func(t *testing.T, output string, links map[int]WikiLink) {
				testutil.Contains(t, output, "[[Code Page]]")
				testutil.NotContains(t, output, "[[Real Page]]")
				testutil.Equal(t, WikiLink{Title: "Real Page"}, links[0])
			},
		},
		{
			name:          "mixed inline code and real wiki link",
			input:         "Use `[[syntax]]` to link to [[Real Page]]",
			expectedLinks: 1,
			checkOutput: func(t *testing.T, output string, links map[int]WikiLink) {
				testutil.Contains(t, output, "`[[syntax]]`")
				testutil.Equal(t, WikiLink{Title: "Real Page"}, links[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, links := preprocessWikiLinks([]byte(tt.input))
			testutil.Equal(t, tt.expectedLinks, len(links))
			tt.checkOutput(t, string(output), links)
		})
	}
}

func TestToConfluenceStorage_WikiLinksInCodeBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		markdown    string
		contains    []string
		notContains []string
	}{
		{
			name:     "wiki link in fenced code block preserved as text",
			markdown: "```\nUse [[My Page]] syntax\n```",
			contains: []string{
				"<code>", "[[My Page]]",
			},
			notContains: []string{
				"<ac:link>",
			},
		},
		{
			name:     "wiki link in inline code preserved",
			markdown: "Use `[[My Page]]` syntax to create links.",
			contains: []string{
				"<code>[[My Page]]</code>",
			},
			notContains: []string{
				"<ac:link>",
			},
		},
		{
			name:     "wiki link outside code block still converted",
			markdown: "```\n[[Code Example]]\n```\n\nSee [[Real Page]].",
			contains: []string{
				"[[Code Example]]",
				"<ac:link>",
				`ri:content-title="Real Page"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ToConfluenceStorage([]byte(tt.markdown))
			testutil.RequireNoError(t, err)
			for _, s := range tt.contains {
				testutil.Contains(t, result, s)
			}
			for _, s := range tt.notContains {
				testutil.NotContains(t, result, s)
			}
		})
	}
}

func TestToADF_WikiLinksInCodeBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		markdown string
		check    func(t *testing.T, jsonStr string)
	}{
		{
			name:     "wiki link in fenced code block not converted to link",
			markdown: "```\n[[My Page]]\n```",
			check: func(t *testing.T, jsonStr string) {
				testutil.Contains(t, jsonStr, "codeBlock")
				testutil.Contains(t, jsonStr, "[[My Page]]")
				testutil.NotContains(t, jsonStr, "confluence-wiki://")
			},
		},
		{
			name:     "wiki link in inline code not converted to link",
			markdown: "Use `[[My Page]]` syntax.",
			check: func(t *testing.T, jsonStr string) {
				testutil.Contains(t, jsonStr, "[[My Page]]")
				testutil.NotContains(t, jsonStr, "confluence-wiki://")
			},
		},
		{
			name:     "wiki link outside code converted, inside preserved",
			markdown: "```\n[[Code Page]]\n```\n\nSee [[Real Page]].",
			check: func(t *testing.T, jsonStr string) {
				testutil.Contains(t, jsonStr, "[[Code Page]]")
				testutil.Contains(t, jsonStr, "confluence-wiki:///Real%20Page")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ToADF([]byte(tt.markdown))
			testutil.RequireNoError(t, err)

			var doc map[string]any
			testutil.RequireNoError(t, json.Unmarshal([]byte(result), &doc))

			tt.check(t, result)
		})
	}
}

func TestPreprocessWikiLinksForADF_CodeBlockProtection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "wiki link in fenced block preserved",
			input:    "```\n[[My Page]]\n```",
			expected: "```\n[[My Page]]\n```",
		},
		{
			name:     "wiki link in inline code preserved",
			input:    "Use `[[My Page]]` syntax",
			expected: "Use `[[My Page]]` syntax",
		},
		{
			name:     "wiki link outside code converted",
			input:    "`[[example]]` and [[Real Page]]",
			expected: "`[[example]]` and [Real Page](confluence-wiki:///Real%20Page)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := preprocessWikiLinksForADF([]byte(tt.input))
			testutil.Equal(t, tt.expected, string(result))
		})
	}
}

func TestWikiLink_EscapedTitleInStorage(t *testing.T) {
	t.Parallel()
	// Titles with XML-special characters should be properly escaped in storage format
	wl := WikiLink{Title: `Page & "Stuff" <here>`}
	storage := RenderWikiLinkToStorage(wl)
	testutil.Contains(t, storage, `ri:content-title="Page &amp; &quot;Stuff&quot; &lt;here&gt;"`)
	// CDATA doesn't need escaping
	testutil.Contains(t, storage, `<![CDATA[Page & "Stuff" <here>]]>`)
}

func TestWikiLink_NotConfusedWithMarkdownLinks(t *testing.T) {
	t.Parallel()
	// Standard markdown links should not be affected
	input := "[regular link](https://example.com)"
	output, links := preprocessWikiLinks([]byte(input))
	testutil.Equal(t, 0, len(links))
	testutil.Equal(t, input, string(output))
}

func TestWikiLink_NotConfusedWithBracketMacros(t *testing.T) {
	t.Parallel()
	// Bracket macros use single brackets and should not be confused with wiki-links
	input := "[TOC]\n\n[[My Page]]"
	output, links := preprocessWikiLinks([]byte(input))
	testutil.Equal(t, 1, len(links))
	testutil.Contains(t, string(output), "[TOC]") // single bracket preserved
	testutil.Equal(t, WikiLink{Title: "My Page"}, links[0])
}

func TestMultipleWikiLinksInLine(t *testing.T) {
	t.Parallel()
	input := "Compare [[Page A]], [[Page B]], and [[DEV:Page C]]."
	storage, err := ToConfluenceStorage([]byte(input))
	testutil.RequireNoError(t, err)

	// All three should be present
	testutil.Equal(t, 3, strings.Count(storage, "<ac:link>"))
	testutil.Contains(t, storage, `ri:content-title="Page A"`)
	testutil.Contains(t, storage, `ri:content-title="Page B"`)
	testutil.Contains(t, storage, `ri:content-title="Page C"`)
	testutil.Contains(t, storage, `ri:space-key="DEV"`)
}

func TestToADF_WikiLink_SpecialCharsInTitle(t *testing.T) {
	t.Parallel()
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
				testutil.Contains(t, jsonStr, `"type":"link"`)
				testutil.Contains(t, jsonStr, `"text":"Q\u0026A Page"`)
			},
		},
		{
			name:     "title with special URL chars",
			markdown: "Check [[Page #1 (draft)]].",
			check: func(t *testing.T, jsonStr string) {
				testutil.Contains(t, jsonStr, `"type":"link"`)
				testutil.Contains(t, jsonStr, `"text":"Page #1 (draft)"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ToADF([]byte(tt.markdown))
			testutil.RequireNoError(t, err)

			var doc map[string]any
			testutil.RequireNoError(t, json.Unmarshal([]byte(result), &doc))

			tt.check(t, result)
		})
	}
}
