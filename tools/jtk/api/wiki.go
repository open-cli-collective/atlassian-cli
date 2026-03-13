package api //nolint:revive // package name is intentional

import (
	"regexp"
	"strings"
)

// Pre-compiled patterns for wiki text formatting conversion.
// Outer patterns match the delimiter + surrounding whitespace/boundaries.
// Inner patterns extract the content between delimiters.
// Boundary patterns for wiki formatting delimiters. We allow whitespace,
// common punctuation, and string boundaries around formatting delimiters
// to support patterns like "(-deleted-)" while rejecting compound words
// like "signal-webapp-frontend".
//
// The before/after sets are intentionally asymmetric: before allows opening
// punctuation (parens, brackets, quotes) while after allows closing punctuation
// (parens, period, comma, quotes, etc.). This mirrors natural prose patterns.
//
// Note: ^ and $ here are inside (?:...) alternations, so they anchor to
// start/end of the entire input string, not line boundaries. This is
// intentional — line-level matching is handled by the multiline (?m) flag
// in wikiPatterns, not here.
const (
	wikiBoundaryBefore = `(?:^|[\s(["'])`
	wikiBoundaryAfter  = `(?:[\s).,"'!?;:\]]|$)`
)

var (
	wikiStrikeOuter    = regexp.MustCompile(wikiBoundaryBefore + `-([^\s-][^-]*[^\s-]|[^\s-])-` + wikiBoundaryAfter)
	wikiStrikeInner    = regexp.MustCompile(`-([^-]+)-`)
	wikiUnderlineOuter = regexp.MustCompile(wikiBoundaryBefore + `\+([^\s+][^+]*[^\s+]|[^\s+])\+` + wikiBoundaryAfter)
	wikiUnderlineInner = regexp.MustCompile(`\+([^+]+)\+`)
)

// replaceWikiFormatting replaces wiki-style inline formatting with the given
// open/close tags. The outer pattern must include surrounding whitespace or
// boundary anchors to avoid matching inside compound words. The inner pattern
// extracts the content between delimiters.
//
// Boundaries allow whitespace, common punctuation (parens, brackets, quotes),
// and string edges around delimiters. This supports wiki patterns like
// "(-deleted-)" while rejecting compound words like "signal-webapp-frontend".
func replaceWikiFormatting(text string, outer, inner *regexp.Regexp, openTag, closeTag string) string {
	replacer := func(match string) string {
		sub := inner.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		prefix := ""
		suffix := ""
		delim := sub[0][0] // first byte of inner match; assumes single-byte ASCII delimiter
		if len(match) > 0 && match[0] != delim {
			prefix = string(match[0])
		}
		if len(match) > 0 && match[len(match)-1] != delim {
			suffix = string(match[len(match)-1])
		}
		return prefix + openTag + sub[1] + closeTag + suffix
	}

	// Run replacement twice to handle adjacent spans where the first match
	// consumes a shared whitespace boundary (e.g., "-one- -two-").
	result := outer.ReplaceAllStringFunc(text, replacer)
	return outer.ReplaceAllStringFunc(result, replacer)
}

// wikiPatterns defines regex patterns for Jira wiki markup detection
var wikiPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^h[1-6]\.\s`),                  // h1. h2. etc
	regexp.MustCompile(`\{\{[^}]+\}\}`),                    // {{monospace}}
	regexp.MustCompile(`\{code[^}]*\}[\s\S]*?\{code\}`),    // {code}...{code}
	regexp.MustCompile(`\{noformat\}[\s\S]*?\{noformat\}`), // {noformat}...{noformat}
	regexp.MustCompile(`\{quote\}[\s\S]*?\{quote\}`),       // {quote}...{quote}
	regexp.MustCompile(`\[([^\]|]+)\|([^\]]+)\]`),          // [text|url]
	regexp.MustCompile(`\![^\s!]+\!`),                      // !image.png!
	regexp.MustCompile(`(?m)^bq\.\s`),                      // bq. blockquote
	regexp.MustCompile(`(?m)^\*+\s`),                       // * bullet (could be markdown too)
	regexp.MustCompile(`(?m)^#+\s+[^#]`),                   // # numbered list (not markdown heading)
}

// IsWikiMarkup detects if text contains Jira wiki markup patterns.
// Returns true if wiki markup is detected, false if it appears to be
// plain text or markdown.
func IsWikiMarkup(text string) bool {
	// Quick check for obvious wiki patterns
	for _, pattern := range wikiPatterns {
		if pattern.MatchString(text) {
			// For bullet patterns, verify it's not markdown
			if strings.HasPrefix(pattern.String(), `(?m)^\*+\s`) {
				// Check if it looks more like wiki (no blank line before)
				continue
			}
			// For # pattern, make sure it's numbered list not markdown heading
			if strings.HasPrefix(pattern.String(), `(?m)^#+\s+[^#]`) {
				// In wiki markup, # is numbered list; in markdown it's heading
				// Check context to decide
				if looksLikeWikiNumberedList(text) {
					return true
				}
				continue
			}
			return true
		}
	}
	return false
}

// looksLikeWikiNumberedList checks if # usage looks like wiki numbered lists.
// Wiki numbered lists use single # (e.g., "# item") and are consecutive lines
// without blank lines between them. Markdown headings use # too, but have
// content paragraphs or blank lines between them.
func looksLikeWikiNumberedList(text string) bool {
	lines := strings.Split(text, "\n")

	// Any multi-hash heading (##, ###, etc.) means this is markdown, not wiki.
	// Known tradeoff: Jira wiki supports ## for nested numbered lists, but in
	// practice this is rare compared to markdown ## headings. We prioritize
	// not mangling markdown content over detecting every wiki edge case.
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) >= 2 && trimmed[0] == '#' && trimmed[1] == '#' {
			return false
		}
	}

	// Count consecutive "# text" lines. Wiki numbered lists are consecutive;
	// markdown h1 headings have blank lines and content between them.
	// Known tradeoff: consecutive h1 headings with no blank line between them
	// (e.g., "# A\n# B") will be detected as wiki. This is acceptable because
	// such formatting is malformed markdown but valid wiki numbered lists.
	consecutive := 0
	maxConsecutive := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") && len(trimmed[2:]) < 80 {
			consecutive++
			if consecutive > maxConsecutive {
				maxConsecutive = consecutive
			}
		} else if trimmed == "" {
			// Blank lines break consecutive runs — markdown headings have these
			consecutive = 0
		} else {
			consecutive = 0
		}
	}
	return maxConsecutive >= 2
}

// WikiToADFMarkdown converts Jira wiki markup to a markdown dialect tuned for the
// MarkdownToADF / goldmark-extras pipeline. The output is NOT general-purpose
// markdown — it relies on adf.ToDocumentWiki (the extended goldmark parser) to
// interpret certain patterns:
//
//   - ~text~ and ^text^ pass through unchanged for goldmark subscript/superscript
//   - +text+ is converted to ++text++ for goldmark Insert (→ ADF underline)
//   - -text- is converted to ~~text~~ for goldmark Delete (→ ADF strikethrough)
//
// Callers MUST use adf.ToDocumentWiki (not adf.ToDocument) to parse the output,
// otherwise ~text~ and ^text^ will not produce the expected ADF marks.
func WikiToADFMarkdown(wiki string) string {
	if wiki == "" {
		return ""
	}

	result := wiki

	// Convert headings: h1. Title -> # Title
	result = convertWikiHeadings(result)

	// Convert code blocks: {code:java}...{code} -> ```java...```
	result = convertWikiCodeBlocks(result)

	// Convert noformat blocks: {noformat}...{noformat} -> ```...```
	result = convertWikiNoformat(result)

	// Convert quote blocks: {quote}...{quote} -> > ...
	result = convertWikiQuoteBlocks(result)

	// Convert monospace: {{text}} -> `text`
	result = convertWikiMonospace(result)

	// Convert links: [text|url] -> [text](url)
	result = convertWikiLinks(result)

	// Convert images: !image.png! -> ![](image.png)
	result = convertWikiImages(result)

	// Convert text formatting
	result = convertWikiTextFormatting(result)

	// Convert blockquotes: bq. text -> > text
	result = convertWikiBlockquotes(result)

	// Convert lists
	result = convertWikiLists(result)

	// Convert horizontal rules: ---- -> ---
	result = convertWikiHorizontalRules(result)

	return result
}

// convertWikiHeadings converts h1. through h6. to markdown headings
func convertWikiHeadings(text string) string {
	// h1. Title -> # Title
	for i := 1; i <= 6; i++ {
		pattern := regexp.MustCompile(`(?m)^h` + string(rune('0'+i)) + `\.\s*(.*)$`)
		prefix := strings.Repeat("#", i)
		text = pattern.ReplaceAllString(text, prefix+" $1")
	}
	return text
}

// convertWikiCodeBlocks converts {code}...{code} to fenced code blocks
func convertWikiCodeBlocks(text string) string {
	// {code:language}content{code} or {code}content{code}
	// Use negative lookbehind to avoid matching {code inside {{code}}
	// Since Go regex doesn't support lookbehind, we use a workaround:
	// Match from start of line or after non-{ character
	pattern := regexp.MustCompile(`(?s)(^|[^{])\{code(?::([a-zA-Z0-9]+))?\}(.*?)\{code\}`)
	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		submatches := pattern.FindStringSubmatch(match)
		prefix := ""
		lang := ""
		content := ""
		if len(submatches) >= 4 {
			prefix = submatches[1]
			lang = submatches[2]
			content = submatches[3]
		}
		// Trim leading/trailing newlines from content
		content = strings.TrimPrefix(content, "\n")
		content = strings.TrimSuffix(content, "\n")
		return prefix + "```" + lang + "\n" + content + "\n```"
	})
}

// convertWikiNoformat converts {noformat}...{noformat} to fenced code blocks
func convertWikiNoformat(text string) string {
	pattern := regexp.MustCompile(`(?s)\{noformat\}(.*?)\{noformat\}`)
	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		submatches := pattern.FindStringSubmatch(match)
		content := ""
		if len(submatches) >= 2 {
			content = submatches[1]
		}
		content = strings.TrimPrefix(content, "\n")
		content = strings.TrimSuffix(content, "\n")
		return "```\n" + content + "\n```"
	})
}

// convertWikiQuoteBlocks converts {quote}...{quote} to markdown blockquotes
func convertWikiQuoteBlocks(text string) string {
	pattern := regexp.MustCompile(`(?s)\{quote\}(.*?)\{quote\}`)
	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		submatches := pattern.FindStringSubmatch(match)
		content := ""
		if len(submatches) >= 2 {
			content = submatches[1]
		}
		content = strings.TrimSpace(content)
		// Add > prefix to each line
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lines[i] = "> " + line
		}
		return strings.Join(lines, "\n")
	})
}

// convertWikiMonospace converts {{text}} to `text`
func convertWikiMonospace(text string) string {
	pattern := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	return pattern.ReplaceAllString(text, "`$1`")
}

// convertWikiLinks converts [text|url] to [text](url)
func convertWikiLinks(text string) string {
	// [link text|http://example.com] -> [link text](http://example.com)
	pattern := regexp.MustCompile(`\[([^\]|]+)\|([^\]]+)\]`)
	return pattern.ReplaceAllString(text, "[$1]($2)")
}

// convertWikiImages converts !image.png! to ![](image.png)
func convertWikiImages(text string) string {
	// !image.png! -> ![](image.png)
	// !image.png|alt=text! -> ![text](image.png)
	pattern := regexp.MustCompile(`!([^\s!|]+)(?:\|([^!]+))?!`)
	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		submatches := pattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		src := submatches[1]
		alt := ""
		if len(submatches) >= 3 && submatches[2] != "" {
			// Parse alt=text or other attributes
			attrs := submatches[2]
			if strings.HasPrefix(attrs, "alt=") {
				alt = strings.TrimPrefix(attrs, "alt=")
			}
		}
		return "![" + alt + "](" + src + ")"
	})
}

// convertWikiTextFormatting converts wiki text formatting to markdown
func convertWikiTextFormatting(text string) string {
	// Bold: *text* -> **text** (but not if already markdown **)
	// Need to be careful not to convert markdown ** to ****
	// Wiki uses single *, markdown uses double **
	// Only convert if it's clearly wiki style (word boundaries)

	// Strikethrough: -text- -> ~~text~~
	// Require whitespace or start/end of string around the delimiters to avoid
	// matching hyphens in compound words like "signal-webapp-frontend".
	text = replaceWikiFormatting(text, wikiStrikeOuter, wikiStrikeInner, "~~", "~~")

	// Underline: +text+ -> ++text++ (goldmark extras Insert extension)
	// Require whitespace or start/end of string around delimiters.
	text = replaceWikiFormatting(text, wikiUnderlineOuter, wikiUnderlineInner, "++", "++")

	// Citation: ??text?? -> <cite>text</cite>
	citePattern := regexp.MustCompile(`\?\?([^?]+)\?\?`)
	text = citePattern.ReplaceAllString(text, "<cite>$1</cite>")

	return text
}

// convertWikiBlockquotes converts bq. lines to markdown blockquotes
func convertWikiBlockquotes(text string) string {
	// bq. text -> > text
	pattern := regexp.MustCompile(`(?m)^bq\.\s*(.*)$`)
	return pattern.ReplaceAllString(text, "> $1")
}

// convertWikiLists converts wiki lists to markdown lists
func convertWikiLists(text string) string {
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		converted := convertWikiListLine(line)
		result = append(result, converted)
	}

	return strings.Join(result, "\n")
}

// convertWikiListLine converts a single wiki list line to markdown
func convertWikiListLine(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]

	// Bullet lists: * item, ** nested -> - item, - nested (with indent)
	if strings.HasPrefix(trimmed, "* ") {
		return indent + "- " + trimmed[2:]
	}
	if strings.HasPrefix(trimmed, "** ") {
		return indent + "  - " + trimmed[3:]
	}
	if strings.HasPrefix(trimmed, "*** ") {
		return indent + "    - " + trimmed[4:]
	}

	// Note: We intentionally do NOT convert wiki # numbered lists here
	// because # at the start of a line is ambiguous between:
	// - Wiki numbered list: # item
	// - Markdown heading: # Title
	// Users should use "1. item" for numbered lists to avoid ambiguity.

	return line
}

// convertWikiHorizontalRules converts ---- to ---
func convertWikiHorizontalRules(text string) string {
	// Wiki uses ---- for horizontal rule, markdown uses ---
	pattern := regexp.MustCompile(`(?m)^----+$`)
	return pattern.ReplaceAllString(text, "---")
}
