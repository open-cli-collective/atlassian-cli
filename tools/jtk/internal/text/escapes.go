// Package text provides text manipulation utilities.
package text

import (
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// InterpretEscapesUnlessRawADF interprets C-style escape sequences in s via
// InterpretEscapes, unless s is itself a raw ADF JSON document (see
// api.IsRawADFDocument). Escape interpretation is a markdown convenience;
// running it over raw ADF JSON first would corrupt the JSON (e.g. turning
// an escaped "\n" inside a JSON string into a literal newline byte) before
// the ADF parser ever sees it, so raw ADF is passed through unmodified
// instead. Call sites that hand free-text description/body/field values to
// NewADFDocument or MarkdownToADF should route them through this helper
// rather than calling InterpretEscapes directly.
func InterpretEscapesUnlessRawADF(s string) string {
	if api.IsRawADFDocument(s) {
		return s
	}
	return InterpretEscapes(s)
}

// InterpretEscapes processes C-style escape sequences in a string.
// This handles the common case where CLI users pass literal \n, \t, or \\
// in flag values and expect them to be interpreted as actual control characters.
func InterpretEscapes(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}

		switch s[i+1] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case '\\':
			b.WriteByte('\\')
		default:
			// Not a recognized escape — keep both characters
			b.WriteByte(s[i])
			b.WriteByte(s[i+1])
		}
		i++ // skip the character after backslash
	}

	return b.String()
}
