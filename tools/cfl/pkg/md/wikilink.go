// wikilink.go provides wiki-link syntax support for Confluence internal page links.
//
// Syntax:
//   - [[Page Title]]          — link to page in same space
//   - [[SPACE:Page Title]]    — link to page in different space
package md

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// WikiLink represents a parsed wiki-link.
type WikiLink struct {
	SpaceKey string // empty for same-space links
	Title    string // page title
}

// wikiLinkPlaceholderPrefix is used to mark where wiki-links should be inserted
// after goldmark processing. Uses a format that won't be interpreted as markdown.
const wikiLinkPlaceholderPrefix = "CFWIKILINK"
const wikiLinkPlaceholderSuffix = "ENDWL"

// wikiLinkPattern matches [[...]] syntax (non-greedy, no nested brackets).
var wikiLinkPattern = regexp.MustCompile(`\[\[([^\[\]]+)\]\]`)

// ParseWikiLink parses the inner content of a [[...]] wiki-link.
func ParseWikiLink(inner string) WikiLink {
	inner = strings.TrimSpace(inner)
	if idx := strings.Index(inner, ":"); idx > 0 {
		candidate := inner[:idx]
		// Space keys are uppercase alphanumeric (typically 2-10 chars)
		if isSpaceKey(candidate) {
			return WikiLink{
				SpaceKey: candidate,
				Title:    strings.TrimSpace(inner[idx+1:]),
			}
		}
	}
	return WikiLink{Title: inner}
}

// isSpaceKey returns true if s looks like a Confluence space key
// (uppercase letters, digits, hyphens, underscores; 1-255 chars).
func isSpaceKey(s string) bool {
	if len(s) == 0 || len(s) > 255 {
		return false
	}
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '~' {
			return false
		}
	}
	return true
}

// RenderWikiLinkToStorage converts a WikiLink to Confluence storage format XML.
func RenderWikiLinkToStorage(wl WikiLink) string {
	var sb strings.Builder
	sb.WriteString(`<ac:link>`)
	sb.WriteString(`<ri:page ri:content-title="`)
	sb.WriteString(escapeXML(wl.Title))
	sb.WriteString(`"`)
	if wl.SpaceKey != "" {
		sb.WriteString(` ri:space-key="`)
		sb.WriteString(escapeXML(wl.SpaceKey))
		sb.WriteString(`"`)
	}
	sb.WriteString(` />`)
	sb.WriteString(`<ac:plain-text-link-body><![CDATA[`)
	sb.WriteString(wl.Title)
	sb.WriteString(`]]></ac:plain-text-link-body>`)
	sb.WriteString(`</ac:link>`)
	return sb.String()
}

// RenderWikiLinkToBracket converts a WikiLink back to [[...]] syntax.
func RenderWikiLinkToBracket(wl WikiLink) string {
	if wl.SpaceKey != "" {
		return "[[" + wl.SpaceKey + ":" + wl.Title + "]]"
	}
	return "[[" + wl.Title + "]]"
}

// formatWikiLinkPlaceholder creates a placeholder string for a wiki-link.
func formatWikiLinkPlaceholder(id int) string {
	return fmt.Sprintf("%s%d%s", wikiLinkPlaceholderPrefix, id, wikiLinkPlaceholderSuffix)
}

// preprocessWikiLinks replaces [[...]] syntax with inline placeholders before
// goldmark processing. Returns the processed text and a map of placeholder ID
// to WikiLink. Code regions (fenced blocks, inline code) are excluded.
func preprocessWikiLinks(input []byte) ([]byte, map[int]WikiLink) {
	// Protect code regions so wiki-link regex doesn't match inside them
	protected, codeRegions := protectCodeRegions(input)

	result, links := preprocessWikiLinksRaw(protected)

	// Restore code regions
	result = restoreCodeRegions(result, codeRegions)

	return result, links
}

// preprocessWikiLinksRaw replaces [[...]] syntax with placeholders without
// code-region protection. Use this when the caller has already protected
// code regions (e.g., ToConfluenceStorage which protects once for all preprocessors).
func preprocessWikiLinksRaw(input []byte) ([]byte, map[int]WikiLink) {
	links := make(map[int]WikiLink)
	counter := 0

	result := wikiLinkPattern.ReplaceAllFunc(input, func(match []byte) []byte {
		inner := string(match[2 : len(match)-2]) // strip [[ and ]]
		wl := ParseWikiLink(inner)
		if wl.Title == "" {
			return match // leave malformed links alone
		}
		id := counter
		counter++
		links[id] = wl
		return []byte(formatWikiLinkPlaceholder(id))
	})

	return result, links
}

// postprocessWikiLinksStorage replaces wiki-link placeholders with Confluence
// storage format XML in HTML output.
func postprocessWikiLinksStorage(html string, links map[int]WikiLink) string {
	for id, wl := range links {
		placeholder := formatWikiLinkPlaceholder(id)
		xml := RenderWikiLinkToStorage(wl)
		html = strings.Replace(html, placeholder, xml, 1)
	}
	return html
}

// postprocessWikiLinksADF replaces wiki-link placeholders in ADF JSON text
// nodes. Since placeholders end up inside text node values, this is called
// on the final JSON string. Each placeholder is replaced with a link mark node.
//
// Because ADF link nodes are structural (text with marks), we handle this at
// the preprocessing stage instead: we convert [[...]] to standard markdown
// links with a special scheme so goldmark and the ADF converter produce proper
// link marks.
//
// See preprocessWikiLinksForADF.
func preprocessWikiLinksForADF(input []byte) []byte {
	// Protect code regions so wiki-link regex doesn't match inside them
	protected, codeRegions := protectCodeRegions(input)

	result := wikiLinkPattern.ReplaceAllFunc(protected, func(match []byte) []byte {
		inner := string(match[2 : len(match)-2])
		wl := ParseWikiLink(inner)
		if wl.Title == "" {
			return match
		}
		// Convert to standard markdown link syntax.
		// The ADF converter handles standard links via goldmark's AST.
		// URL-encode the title component so goldmark parses it as a valid link.
		var href string
		if wl.SpaceKey != "" {
			href = "confluence-wiki://" + wl.SpaceKey + "/" + url.PathEscape(wl.Title)
		} else {
			href = "confluence-wiki:///" + url.PathEscape(wl.Title)
		}
		return []byte("[" + wl.Title + "](" + href + ")")
	})

	// Restore code regions
	result = restoreCodeRegions(result, codeRegions)

	return result
}

// acLinkPattern matches <ac:link> elements in Confluence storage format.
var acLinkPattern = regexp.MustCompile(`(?s)<ac:link>\s*<ri:page\s+([^/]*)/>\s*(?:<ac:plain-text-link-body><!\[CDATA\[([^\]]*)\]\]></ac:plain-text-link-body>\s*)?</ac:link>`)

// riPageAttrPattern extracts attributes from <ri:page> elements.
var riPageTitlePattern = regexp.MustCompile(`ri:content-title="([^"]*)"`)
var riPageSpacePattern = regexp.MustCompile(`ri:space-key="([^"]*)"`)

// wikiLinkFromHTMLPlaceholderPrefix is used for wiki-link placeholders in the
// HTML→Markdown direction (distinct from the MD→HTML direction to avoid collisions).
const wikiLinkFromHTMLPlaceholderPrefix = "CFWLVIEW"
const wikiLinkFromHTMLPlaceholderSuffix = "ENDWLV"

// formatWikiLinkFromHTMLPlaceholder creates a placeholder for the HTML→Markdown path.
func formatWikiLinkFromHTMLPlaceholder(id int) string {
	return fmt.Sprintf("%s%d%s", wikiLinkFromHTMLPlaceholderPrefix, id, wikiLinkFromHTMLPlaceholderSuffix)
}

// convertACLinksToPlaceholders converts <ac:link> elements to inline text
// placeholders and returns a map from placeholder ID to WikiLink.
// The placeholders survive the HTML-to-markdown conversion without escaping.
func convertACLinksToPlaceholders(html string) (string, map[int]WikiLink) {
	links := make(map[int]WikiLink)
	counter := 0

	result := acLinkPattern.ReplaceAllStringFunc(html, func(match string) string {
		titleMatch := riPageTitlePattern.FindStringSubmatch(match)
		if len(titleMatch) < 2 || titleMatch[1] == "" {
			return match
		}
		title := unescapeXML(titleMatch[1])

		spaceMatch := riPageSpacePattern.FindStringSubmatch(match)
		spaceKey := ""
		if len(spaceMatch) > 1 {
			spaceKey = unescapeXML(spaceMatch[1])
		}

		id := counter
		counter++
		links[id] = WikiLink{SpaceKey: spaceKey, Title: title}
		return formatWikiLinkFromHTMLPlaceholder(id)
	})

	return result, links
}

// replaceWikiLinkPlaceholders replaces wiki-link placeholders with [[...]] syntax
// in the final markdown output.
func replaceWikiLinkPlaceholders(markdown string, links map[int]WikiLink) string {
	for id, wl := range links {
		placeholder := formatWikiLinkFromHTMLPlaceholder(id)
		bracket := RenderWikiLinkToBracket(wl)
		markdown = strings.Replace(markdown, placeholder, bracket, 1)
	}
	return markdown
}

// convertACLinksToMarkdownLinks converts <ac:link> elements to standard
// markdown-style HTML links for the default (non-show-macros) path.
// This allows the HTML-to-markdown converter to produce [Title](url) links.
func convertACLinksToMarkdownLinks(html string) string {
	return acLinkPattern.ReplaceAllStringFunc(html, func(match string) string {
		titleMatch := riPageTitlePattern.FindStringSubmatch(match)
		if len(titleMatch) < 2 || titleMatch[1] == "" {
			return match
		}
		title := titleMatch[1] // keep escaped for HTML context

		return `<a href="#">` + title + `</a>`
	})
}

// unescapeXML reverses escapeXML for reading attribute values.
func unescapeXML(s string) string {
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&amp;", "&")
	return s
}
