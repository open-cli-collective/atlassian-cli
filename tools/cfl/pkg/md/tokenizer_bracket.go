// tokenizer_bracket.go implements tokenization for [MACRO]...[/MACRO] bracket syntax.
package md

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenizeBrackets scans input for bracket macro syntax and returns a token stream.
// Recognized forms:
//   - [MACRO] or [MACRO params] - open tag
//   - [/MACRO] - close tag
//   - [MACRO/] - self-closing (no body)
//
// Text between macros is returned as BracketTokenText tokens.
// Unknown macro names are still tokenized (validation happens in parser).
func TokenizeBrackets(input string) ([]BracketToken, error) {
	var tokens []BracketToken
	pos := 0
	textStart := 0

	for pos < len(input) {
		// Look for opening bracket
		if input[pos] == '[' {
			// Emit any accumulated text before this bracket
			if pos > textStart {
				tokens = append(tokens, BracketToken{
					Type:     BracketTokenText,
					Text:     input[textStart:pos],
					Position: textStart,
				})
			}

			// Try to parse a macro tag
			token, endPos, err := parseBracketTag(input, pos)
			if err != nil {
				// Not a valid macro tag - treat '[' as text
				pos++
				continue
			}

			tokens = append(tokens, token)
			pos = endPos
			textStart = pos
		} else {
			pos++
		}
	}

	// Emit any remaining text
	if textStart < len(input) {
		tokens = append(tokens, BracketToken{
			Type:     BracketTokenText,
			Text:     input[textStart:],
			Position: textStart,
		})
	}

	return tokens, nil
}

// parseBracketTag attempts to parse a macro tag starting at pos.
// Returns the token, the position after the tag, and any error.
func parseBracketTag(input string, pos int) (BracketToken, int, error) {
	if pos >= len(input) || input[pos] != '[' {
		return BracketToken{}, pos, fmt.Errorf("expected '['")
	}

	startPos := pos
	pos++ // skip '['

	// Check for close tag [/MACRO]
	isCloseTag := false
	if pos < len(input) && input[pos] == '/' {
		isCloseTag = true
		pos++
	}

	// Parse macro name (letters only, case-insensitive)
	nameStart := pos
	for pos < len(input) && isValidMacroNameChar(rune(input[pos])) {
		pos++
	}
	if pos == nameStart {
		return BracketToken{}, startPos, fmt.Errorf("empty macro name")
	}
	macroName := input[nameStart:pos]

	// For close tags, expect immediate ']'
	if isCloseTag {
		if pos >= len(input) || input[pos] != ']' {
			return BracketToken{}, startPos, fmt.Errorf("unclosed close tag")
		}
		pos++ // skip ']'
		return BracketToken{
			Type:            BracketTokenCloseTag,
			MacroName:       strings.ToUpper(macroName),
			OriginalName:    macroName,
			OriginalTagText: input[startPos:pos],
			Position:        startPos,
		}, pos, nil
	}

	// For open tags, check for self-close [MACRO/] or params and closing bracket
	// Skip whitespace after macro name
	for pos < len(input) && unicode.IsSpace(rune(input[pos])) {
		pos++
	}

	// Check for self-closing [MACRO/]
	if pos < len(input) && input[pos] == '/' {
		pos++
		if pos >= len(input) || input[pos] != ']' {
			return BracketToken{}, startPos, fmt.Errorf("expected ']' after '/'")
		}
		pos++ // skip ']'
		return BracketToken{
			Type:            BracketTokenSelfClose,
			MacroName:       strings.ToUpper(macroName),
			OriginalName:    macroName,
			Parameters:      make(map[string]string),
			OriginalTagText: input[startPos:pos],
			Position:        startPos,
		}, pos, nil
	}

	// Parse parameters until ']' or '/'
	paramStart := pos
	params, endPos, err := parseParametersUntilClose(input, pos)
	if err != nil {
		return BracketToken{}, startPos, err
	}
	_ = paramStart // used for debugging if needed

	// Check if this is a self-closing tag with parameters (e.g., [TOC maxLevel=3/])
	if endPos < len(input) && input[endPos] == '/' {
		endPos++ // skip '/'
		if endPos >= len(input) || input[endPos] != ']' {
			return BracketToken{}, startPos, fmt.Errorf("expected ']' after '/'")
		}
		endPos++ // skip ']'
		return BracketToken{
			Type:            BracketTokenSelfClose,
			MacroName:       strings.ToUpper(macroName),
			OriginalName:    macroName,
			Parameters:      params,
			OriginalTagText: input[startPos:endPos],
			Position:        startPos,
		}, endPos, nil
	}

	return BracketToken{
		Type:            BracketTokenOpenTag,
		MacroName:       strings.ToUpper(macroName),
		OriginalName:    macroName,
		Parameters:      params,
		OriginalTagText: input[startPos:endPos],
		Position:        startPos,
	}, endPos, nil
}

// parseParametersUntilClose parses key=value parameters until ']'.
// Returns the parameter map, position after ']', and any error.
func parseParametersUntilClose(input string, pos int) (map[string]string, int, error) {
	params := make(map[string]string)

	for pos < len(input) {
		// Skip whitespace
		for pos < len(input) && unicode.IsSpace(rune(input[pos])) {
			pos++
		}

		// Check for end of tag
		if pos < len(input) && input[pos] == ']' {
			pos++ // skip ']'
			return params, pos, nil
		}

		// Check for self-close marker
		if pos < len(input) && input[pos] == '/' {
			pos++
			if pos >= len(input) || input[pos] != ']' {
				return nil, pos, fmt.Errorf("expected ']' after '/'")
			}
			// Don't consume ']' here - let caller handle it
			pos-- // back up to '/'
			return params, pos, nil
		}

		// Parse parameter key
		keyStart := pos
		for pos < len(input) && isValidParamKeyChar(rune(input[pos])) {
			pos++
		}
		if pos == keyStart {
			// No key found, might be end of params
			if pos < len(input) && input[pos] == ']' {
				pos++
				return params, pos, nil
			}
			return nil, pos, fmt.Errorf("expected parameter key or ']'")
		}
		key := input[keyStart:pos]

		// Expect '='
		if pos >= len(input) || input[pos] != '=' {
			// Key without value - treat as boolean true
			params[key] = "true"
			continue
		}
		pos++ // skip '='

		// Parse value (may be quoted)
		value, newPos, err := parseParamValue(input, pos)
		if err != nil {
			return nil, pos, err
		}
		params[key] = value
		pos = newPos
	}

	return nil, pos, fmt.Errorf("unclosed bracket tag")
}

// parseParamValue parses a parameter value, handling quoted strings.
// Escaped quotes (\' or \") are unescaped in the returned value.
func parseParamValue(input string, pos int) (string, int, error) {
	if pos >= len(input) {
		return "", pos, fmt.Errorf("unexpected end of input")
	}

	// Check for quoted value
	if input[pos] == '"' || input[pos] == '\'' {
		quoteChar := input[pos]
		pos++ // skip opening quote
		valueStart := pos
		var value strings.Builder

		for pos < len(input) {
			if input[pos] == quoteChar {
				// Append any remaining unescaped content
				if valueStart < pos {
					value.WriteString(input[valueStart:pos])
				}
				pos++ // skip closing quote
				return value.String(), pos, nil
			}
			// Handle escaped quotes
			if input[pos] == '\\' && pos+1 < len(input) && input[pos+1] == quoteChar {
				// Append content up to the backslash, skip the backslash, and continue
				if valueStart < pos {
					value.WriteString(input[valueStart:pos])
				}
				value.WriteByte(quoteChar) // write the unescaped quote
				pos += 2                   // skip backslash and quote
				valueStart = pos
				continue
			}
			pos++
		}
		return "", pos, fmt.Errorf("unclosed quoted value")
	}

	// Unquoted value - read until space or ']'
	valueStart := pos
	for pos < len(input) && !unicode.IsSpace(rune(input[pos])) && input[pos] != ']' && input[pos] != '/' {
		pos++
	}
	return input[valueStart:pos], pos, nil
}

// isValidMacroNameChar returns true if r is valid in a macro name.
func isValidMacroNameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}

// isValidParamKeyChar returns true if r is valid in a parameter key.
func isValidParamKeyChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}
