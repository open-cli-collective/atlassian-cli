package api

import (
	"regexp"
	"strings"
)

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

// looksLikeWikiNumberedList checks if # usage looks like wiki numbered lists
func looksLikeWikiNumberedList(text string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Wiki numbered lists: # item, ## nested item
		// Markdown headings: # Title (usually followed by content, not lists)
		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") {
			// If the line after # is short and there are multiple such lines,
			// it's likely a wiki numbered list
			rest := strings.TrimLeft(trimmed, "# ")
			if len(rest) < 80 && !strings.Contains(rest, "#") {
				// Count consecutive # lines
				count := 0
				for _, l := range lines {
					if strings.HasPrefix(strings.TrimSpace(l), "#") {
						count++
					}
				}
				if count >= 2 {
					return true
				}
			}
		}
	}
	return false
}

// WikiToMarkdown converts Jira wiki markup to markdown format.
// This enables users to paste wiki-formatted text and have it properly
// converted to ADF for Jira Cloud.
func WikiToMarkdown(wiki string) string {
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
	strikePattern := regexp.MustCompile(`(?:^|[^-])-([^\s-][^-]*[^\s-]|[^\s-])-(?:[^-]|$)`)
	text = strikePattern.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the content between dashes
		innerPattern := regexp.MustCompile(`-([^-]+)-`)
		inner := innerPattern.FindStringSubmatch(match)
		if len(inner) >= 2 {
			prefix := ""
			suffix := ""
			if len(match) > 0 && match[0] != '-' {
				prefix = string(match[0])
			}
			if len(match) > 0 && match[len(match)-1] != '-' {
				suffix = string(match[len(match)-1])
			}
			return prefix + "~~" + inner[1] + "~~" + suffix
		}
		return match
	})

	// Underline: +text+ -> <u>text</u> (no markdown equivalent, use HTML)
	underlinePattern := regexp.MustCompile(`\+([^\s+][^+]*[^\s+]|[^\s+])\+`)
	text = underlinePattern.ReplaceAllString(text, "<u>$1</u>")

	// Subscript: ~text~ -> <sub>text</sub>
	subPattern := regexp.MustCompile(`~([^\s~][^~]*[^\s~]|[^\s~])~`)
	text = subPattern.ReplaceAllString(text, "<sub>$1</sub>")

	// Superscript: ^text^ -> <sup>text</sup>
	supPattern := regexp.MustCompile(`\^([^\s^][^^]*[^\s^]|[^\s^])\^`)
	text = supPattern.ReplaceAllString(text, "<sup>$1</sup>")

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
