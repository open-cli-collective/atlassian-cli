package text

import "strings"

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
