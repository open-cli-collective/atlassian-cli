package api //nolint:revive // package name is intentional

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
)

func TestIsWikiMarkup(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "h1 heading",
			input:    "h1. This is a heading",
			expected: true,
		},
		{
			name:     "h2 heading",
			input:    "h2. Another heading",
			expected: true,
		},
		{
			name:     "monospace",
			input:    "Some {{inline code}} here",
			expected: true,
		},
		{
			name:     "code block",
			input:    "{code:java}\npublic class Test {}\n{code}",
			expected: true,
		},
		{
			name:     "wiki link",
			input:    "Check out [this link|https://example.com]",
			expected: true,
		},
		{
			name:     "wiki image",
			input:    "See !screenshot.png!",
			expected: true,
		},
		{
			name:     "blockquote",
			input:    "bq. This is a quote",
			expected: true,
		},
		{
			name:     "noformat block",
			input:    "{noformat}some text{noformat}",
			expected: true,
		},
		{
			name:     "quote block",
			input:    "{quote}quoted text{quote}",
			expected: true,
		},
		{
			name:     "plain markdown",
			input:    "# Heading\n\nSome **bold** text",
			expected: false,
		},
		{
			name:     "markdown link",
			input:    "Check [this](https://example.com)",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "plain text",
			input:    "Just some plain text without any formatting",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWikiMarkup(tt.input)
			testutil.Equal(t, result, tt.expected)
		})
	}
}

func TestWikiToADFMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "h1 heading",
			input:    "h1. Main Title",
			expected: "# Main Title",
		},
		{
			name:     "h2 heading",
			input:    "h2. Section",
			expected: "## Section",
		},
		{
			name:     "h3 heading",
			input:    "h3. Subsection",
			expected: "### Subsection",
		},
		{
			name:     "multiple headings",
			input:    "h1. Title\nh2. Section\nh3. Subsection",
			expected: "# Title\n## Section\n### Subsection",
		},
		{
			name:     "monospace",
			input:    "Use {{git status}} to check",
			expected: "Use `git status` to check",
		},
		{
			name:     "code block without language",
			input:    "{code}\nfunc main() {}\n{code}",
			expected: "```\nfunc main() {}\n```",
		},
		{
			name:     "code block with language",
			input:    "{code:go}\nfunc main() {}\n{code}",
			expected: "```go\nfunc main() {}\n```",
		},
		{
			name:     "noformat block",
			input:    "{noformat}\nsome preformatted text\n{noformat}",
			expected: "```\nsome preformatted text\n```",
		},
		{
			name:     "wiki link",
			input:    "See [Google|https://google.com] for more",
			expected: "See [Google](https://google.com) for more",
		},
		{
			name:     "wiki image",
			input:    "Screenshot: !image.png!",
			expected: "Screenshot: ![](image.png)",
		},
		{
			name:     "wiki image with alt",
			input:    "!diagram.png|alt=Architecture!",
			expected: "![Architecture](diagram.png)",
		},
		{
			name:     "blockquote line",
			input:    "bq. This is quoted",
			expected: "> This is quoted",
		},
		{
			name:     "quote block",
			input:    "{quote}\nFirst line\nSecond line\n{quote}",
			expected: "> First line\n> Second line",
		},
		{
			name:     "bullet list",
			input:    "* Item 1\n* Item 2\n* Item 3",
			expected: "- Item 1\n- Item 2\n- Item 3",
		},
		{
			name:     "nested bullet list",
			input:    "* Item 1\n** Nested 1\n** Nested 2\n* Item 2",
			expected: "- Item 1\n  - Nested 1\n  - Nested 2\n- Item 2",
		},
		{
			name:     "horizontal rule",
			input:    "Before\n----\nAfter",
			expected: "Before\n---\nAfter",
		},
		{
			name:     "complex document",
			input:    "h1. Guide\n\nThis is about {{code}}.\n\n{code:python}\nprint('hello')\n{code}\n\nSee [docs|https://example.com].",
			expected: "# Guide\n\nThis is about `code`.\n\n```python\nprint('hello')\n```\n\nSee [docs](https://example.com).",
		},
		{
			name:     "subscript passes through for goldmark extras",
			input:    "h1. Formula\n\nH~2~O",
			expected: "# Formula\n\nH~2~O",
		},
		{
			name:     "superscript passes through for goldmark extras",
			input:    "h1. Math\n\nx^2^ squared",
			expected: "# Math\n\nx^2^ squared",
		},
		{
			name:     "underline converted to double plus for goldmark extras",
			input:    "h1. Note\n\nThis is +important+ text",
			expected: "# Note\n\nThis is ++important++ text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WikiToADFMarkdown(tt.input)
			testutil.Equal(t, result, tt.expected)
		})
	}
}

func TestWikiToADFMarkdownPreservesMarkdown(t *testing.T) {
	// Markdown input should pass through mostly unchanged
	// (some edge cases may have minor differences)
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "markdown heading",
			input: "# Title",
		},
		{
			name:  "markdown bold",
			input: "Some **bold** text",
		},
		{
			name:  "markdown code",
			input: "Use `code` here",
		},
		{
			name:  "markdown link",
			input: "[Google](https://google.com)",
		},
		{
			name:  "markdown list",
			input: "- Item 1\n- Item 2",
		},
		{
			name:  "hyphenated compound words preserved",
			input: "Deploy signal-webapp-frontend to ui-components",
		},
		{
			name:  "tilde in text preserved",
			input: "Introduce a three~tier system with ui~components",
		},
		{
			name:  "file paths with hyphens preserved",
			input: "See docs/2026-03-12-v3-theming-design.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WikiToADFMarkdown(tt.input)
			testutil.Equal(t, result, tt.input)
		})
	}
}

func TestConvertWikiTextFormatting_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strikethrough at start of line",
			input:    "-deleted text-",
			expected: "~~deleted text~~",
		},
		{
			name:     "strikethrough after space",
			input:    "some -deleted- text",
			expected: "some ~~deleted~~ text",
		},
		{
			name:     "hyphenated word not converted",
			input:    "signal-webapp-frontend",
			expected: "signal-webapp-frontend",
		},
		{
			name:     "subscript passes through for goldmark",
			input:    "H ~2~ O",
			expected: "H ~2~ O",
		},
		{
			name:     "tilde in compound word not converted",
			input:    "three~tier",
			expected: "three~tier",
		},
		{
			name:     "caret in compound word not converted",
			input:    "x^2^y",
			expected: "x^2^y",
		},
		{
			name:     "underline converts to double plus for goldmark",
			input:    "this is +important+ text",
			expected: "this is ++important++ text",
		},
		{
			name:     "file path hyphens preserved",
			input:    "2026-03-12-design.md",
			expected: "2026-03-12-design.md",
		},
		{
			name:     "consecutive strikethrough with word between",
			input:    "-one- and -two-",
			expected: "~~one~~ and ~~two~~",
		},
		{
			name:     "adjacent strikethrough single space",
			input:    "-one- -two-",
			expected: "~~one~~ ~~two~~",
		},
		{
			name:     "strikethrough at end of string",
			input:    "remove -this-",
			expected: "remove ~~this~~",
		},
		{
			name:     "subscript at end passes through for goldmark",
			input:    "log ~n~",
			expected: "log ~n~",
		},
		{
			name:     "tilde with number not subscript without closing tilde",
			input:    "migrate ~22 new components",
			expected: "migrate ~22 new components",
		},
		{
			name:     "punctuation adjacent formatting converts",
			input:    "see (-deleted-) here",
			expected: "see (~~deleted~~) here",
		},
		{
			name:     "tab adjacent strikethrough converts",
			input:    "text\t-removed-\tend",
			expected: "text\t~~removed~~\tend",
		},
		{
			name:     "period before delimiter does not trigger (asymmetric boundary)",
			input:    "end.-deleted-",
			expected: "end.-deleted-",
		},
		{
			name:     "square bracket adjacent strikethrough converts",
			input:    "[-deleted-]",
			expected: "[~~deleted~~]",
		},
		{
			name:     "newline adjacent strikethrough converts",
			input:    "text\n-removed-\nend",
			expected: "text\n~~removed~~\nend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertWikiTextFormatting(tt.input)
			testutil.Equal(t, result, tt.expected)
		})
	}
}

func TestIsWikiMarkup_MarkdownHeadings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "markdown h2 headings should not be detected as wiki",
			input:    "## What\n\nSome content\n\n## Scope\n\nMore content",
			expected: false,
		},
		{
			name:     "markdown mixed heading levels",
			input:    "## Overview\n\n### Details\n\n## Summary",
			expected: false,
		},
		{
			name:     "actual wiki numbered list",
			input:    "# First item\n# Second item\n# Third item",
			expected: true,
		},
		{
			name:     "multiple markdown h1 headings should not be detected as wiki",
			input:    "# Title\n\nContent\n\n# Another Section",
			expected: false,
		},
		{
			name:     "h3 headings should not be detected as wiki",
			input:    "### Section A\n\n### Section B\n\n### Section C",
			expected: false,
		},
		{
			name:     "adjacent h1 without blank line is treated as wiki list (intentional)",
			input:    "# First item\n# Second item",
			expected: true,
		},
		{
			name:     "nested wiki ## under # treated as markdown not wiki (intentional false negative)",
			input:    "# Top level\n## Nested item\n## Another nested",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWikiMarkup(tt.input)
			testutil.Equal(t, result, tt.expected)
		})
	}
}

func TestMarkdownToADFWithWikiMarkup(t *testing.T) {
	// Test that wiki markup is properly converted when passed to MarkdownToADF
	tests := []struct {
		name      string
		input     string
		checkType string
		checkAttr any
	}{
		{
			name:      "wiki h1 becomes ADF heading",
			input:     "h1. Hello World",
			checkType: "heading",
			checkAttr: 1,
		},
		{
			name:      "wiki h2 becomes ADF heading",
			input:     "h2. Section Title",
			checkType: "heading",
			checkAttr: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := MarkdownToADF(tt.input)
			testutil.NotNil(t, doc)
			testutil.Equal(t, doc.Type, "doc")
			testutil.NotEmpty(t, doc.Content)

			if tt.checkType == "heading" {
				testutil.Equal(t, doc.Content[0].Type, "heading")
				testutil.Equal(t, doc.Content[0].Attrs["level"], tt.checkAttr)
			}
		})
	}
}
