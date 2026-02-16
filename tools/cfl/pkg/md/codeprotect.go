// Package md provides bidirectional Markdown-Confluence conversion.
package md

import (
	"fmt"
	"strings"
)

const codeProtectPrefix = "CFCODE"
const codeProtectSuffix = "ENDC"

// codeRegion records a protected code region's original content.
type codeRegion struct {
	id      int
	content string
}

// formatCodePlaceholder returns the placeholder string for a code region.
func formatCodePlaceholder(id int) string {
	return fmt.Sprintf("%s%d%s", codeProtectPrefix, id, codeProtectSuffix)
}

// protectCodeRegions replaces fenced code blocks and inline code spans with
// alphanumeric placeholders. Returns the modified input and the list of
// regions that were replaced, for later restoration.
//
// Fenced blocks: lines starting with ``` or ~~~ (with optional language tag)
// through the matching closing fence.
//
// Inline code: backtick-delimited spans (`...` or “...“).
func protectCodeRegions(input []byte) ([]byte, []codeRegion) {
	s := string(input)
	var regions []codeRegion
	var out strings.Builder
	out.Grow(len(s))
	i := 0

	for i < len(s) {
		// Check for fenced code block at start of line (or start of input)
		if (i == 0 || s[i-1] == '\n') && i < len(s) {
			fence, fenceLen := detectFence(s, i)
			if fenceLen > 0 {
				// Find the closing fence
				endIdx := findClosingFence(s, i+fenceLen, fence)
				region := codeRegion{
					id:      len(regions),
					content: s[i:endIdx],
				}
				regions = append(regions, region)
				out.WriteString(formatCodePlaceholder(region.id))
				i = endIdx
				continue
			}
		}

		// Check for inline code spans (backtick)
		if s[i] == '`' {
			// Count consecutive backticks for the delimiter
			delimLen := 0
			for i+delimLen < len(s) && s[i+delimLen] == '`' {
				delimLen++
			}

			// Look for matching closing delimiter
			closeIdx := findClosingBackticks(s, i+delimLen, delimLen)
			if closeIdx >= 0 {
				endIdx := closeIdx + delimLen
				region := codeRegion{
					id:      len(regions),
					content: s[i:endIdx],
				}
				regions = append(regions, region)
				out.WriteString(formatCodePlaceholder(region.id))
				i = endIdx
				continue
			}
			// No matching close — emit the backticks as-is
			out.WriteString(s[i : i+delimLen])
			i += delimLen
			continue
		}

		out.WriteByte(s[i])
		i++
	}

	return []byte(out.String()), regions
}

// restoreCodeRegions replaces code placeholders with their original content.
func restoreCodeRegions(input []byte, regions []codeRegion) []byte {
	s := string(input)
	for _, r := range regions {
		placeholder := formatCodePlaceholder(r.id)
		s = strings.Replace(s, placeholder, r.content, 1)
	}
	return []byte(s)
}

// detectFence checks if position i starts a fenced code block opener.
// Returns the fence string (e.g. "```") and the total length consumed
// (including the optional language tag and newline), or 0 if not a fence.
func detectFence(s string, i int) (string, int) {
	if i >= len(s) {
		return "", 0
	}

	fenceChar := s[i]
	if fenceChar != '`' && fenceChar != '~' {
		return "", 0
	}

	// Count fence characters (need at least 3)
	fenceLen := 0
	for i+fenceLen < len(s) && s[i+fenceLen] == fenceChar {
		fenceLen++
	}
	if fenceLen < 3 {
		return "", 0
	}

	fence := s[i : i+fenceLen]

	// Consume the rest of the opening line (language tag, etc.)
	end := i + fenceLen
	for end < len(s) && s[end] != '\n' {
		end++
	}
	if end < len(s) {
		end++ // consume the newline
	}

	return fence, end - i
}

// findClosingFence finds the end of a fenced code block, returning the
// position just past the closing fence line (including its newline).
func findClosingFence(s string, start int, fence string) int {
	i := start
	for i < len(s) {
		// Find the start of the next line
		lineStart := i
		// Check if this line starts with the closing fence
		if strings.HasPrefix(s[lineStart:], fence) {
			// Verify the rest of the line is only whitespace
			end := lineStart + len(fence)
			closingValid := true
			for end < len(s) && s[end] != '\n' {
				if s[end] != ' ' && s[end] != '\t' {
					closingValid = false
					break
				}
				end++
			}
			if closingValid {
				if end < len(s) {
					end++ // consume newline
				}
				return end
			}
		}
		// Advance to next line
		for i < len(s) && s[i] != '\n' {
			i++
		}
		if i < len(s) {
			i++ // skip newline
		}
	}
	// No closing fence found — protect to end of input
	return len(s)
}

// findClosingBackticks finds the position where a matching run of backticks
// starts. Returns -1 if no match is found.
func findClosingBackticks(s string, start, delimLen int) int {
	i := start
	for i < len(s) {
		if s[i] == '`' {
			runStart := i
			runLen := 0
			for i < len(s) && s[i] == '`' {
				runLen++
				i++
			}
			if runLen == delimLen {
				return runStart
			}
			continue
		}
		// CommonMark allows inline code to span lines, but in practice
		// wiki-link syntax won't span lines, so stop at newlines to avoid
		// greedily consuming too much content on unclosed backticks.
		if s[i] == '\n' {
			return -1
		}
		i++
	}
	return -1
}
