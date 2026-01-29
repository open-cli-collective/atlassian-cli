// parser_bracket.go parses bracket syntax [MACRO]...[/MACRO] into MacroNode trees.
package md

import (
	"fmt"
	"strings"
)

// childPlaceholderPrefix is used to mark where child macros appear in a parent's body.
// Format: CFCHILD0, CFCHILD1, etc. corresponding to the index in Children array.
const childPlaceholderPrefix = "CFCHILD"

// ParseBracketMacros parses bracket macro syntax and returns a ParseResult.
// Input: markdown with [MACRO]...[/MACRO] syntax
// Output: segments of text and MacroNode trees
func ParseBracketMacros(input string) (*ParseResult, error) {
	tokens, err := TokenizeBrackets(input)
	if err != nil {
		return nil, err
	}

	result := &ParseResult{}
	stack := []*stackFrame{}

	for _, token := range tokens {
		switch token.Type {
		case BracketTokenText:
			if len(stack) > 0 {
				// Inside a macro - accumulate body text
				stack[len(stack)-1].bodyContent += token.Text
			} else {
				// Top level - add as text segment
				result.AddTextSegment(token.Text)
			}

		case BracketTokenOpenTag:
			// Check if this is a known macro
			macroType, known := LookupMacro(token.MacroName)
			if !known {
				// Unknown macro - treat as text
				result.AddWarning("unknown macro: %s", token.MacroName)
				text := reconstructBracketTag(token)
				if len(stack) > 0 {
					stack[len(stack)-1].bodyContent += text
				} else {
					result.AddTextSegment(text)
				}
				continue
			}

			// Create a new stack frame for this macro
			frame := &stackFrame{
				node: &MacroNode{
					Name:       strings.ToLower(token.MacroName),
					Parameters: token.Parameters,
				},
				macroType: macroType,
			}
			stack = append(stack, frame)

			// If macro has no body, close it immediately
			if !macroType.HasBody {
				closeMacro(result, &stack)
			}

		case BracketTokenCloseTag:
			if len(stack) == 0 {
				// Orphan close tag - treat as text
				result.AddWarning("orphan close tag: [/%s]", token.MacroName)
				result.AddTextSegment("[/" + token.MacroName + "]")
				continue
			}

			// Check if close tag matches current open
			current := stack[len(stack)-1]
			if !strings.EqualFold(current.node.Name, token.MacroName) {
				// Mismatched close tag
				result.AddWarning("mismatched close tag: expected [/%s], got [/%s]",
					current.node.Name, token.MacroName)
				// Try to recover by treating as text
				if len(stack) > 1 {
					stack[len(stack)-1].bodyContent += "[/" + token.MacroName + "]"
				} else {
					result.AddTextSegment("[/" + token.MacroName + "]")
				}
				continue
			}

			// Set body content and close macro
			current.node.Body = current.bodyContent
			closeMacro(result, &stack)

		case BracketTokenSelfClose:
			// Self-closing macro (no body)
			macroType, known := LookupMacro(token.MacroName)
			if !known {
				result.AddWarning("unknown macro: %s", token.MacroName)
				text := reconstructBracketTag(token)
				if len(stack) > 0 {
					stack[len(stack)-1].bodyContent += text
				} else {
					result.AddTextSegment(text)
				}
				continue
			}

			node := &MacroNode{
				Name:       strings.ToLower(token.MacroName),
				Parameters: token.Parameters,
			}

			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				// Add placeholder marker in parent's bodyContent to preserve position
				childIndex := len(parent.node.Children)
				placeholder := fmt.Sprintf("%s%d", childPlaceholderPrefix, childIndex)
				parent.bodyContent += placeholder
				// Nested - add as child
				parent.node.Children = append(parent.node.Children, node)
			} else {
				// Top level
				result.AddMacroSegment(node)
			}
			_ = macroType // validated but not needed for self-close
		}
	}

	// Handle any unclosed macros
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		result.AddWarning("unclosed macro: [%s]", current.node.Name)
		// Emit as text instead of macro
		text := reconstructOpenTag(current.node) + current.bodyContent
		stack = stack[:len(stack)-1]
		if len(stack) > 0 {
			stack[len(stack)-1].bodyContent += text
		} else {
			result.AddTextSegment(text)
		}
	}

	return result, nil
}

// stackFrame tracks parsing state for nested macros.
type stackFrame struct {
	node        *MacroNode
	macroType   MacroType
	bodyContent string
}

// closeMacro pops the current macro from the stack and adds it appropriately.
// When adding a child to a parent, it also inserts a placeholder marker in the
// parent's bodyContent so that the child's position is preserved.
func closeMacro(result *ParseResult, stack *[]*stackFrame) {
	if len(*stack) == 0 {
		return
	}

	current := (*stack)[len(*stack)-1]
	*stack = (*stack)[:len(*stack)-1]

	// Parse any nested macros in the body (for body macros that may contain nested content)
	// Note: This handles cases where the body text contains literal macro syntax that wasn't
	// parsed during streaming (e.g., when body text is accumulated separately).
	if current.node.Body != "" && current.macroType.HasBody {
		nested, err := ParseBracketMacros(current.node.Body)
		if err == nil {
			// Check if the body had any nested macros (text would just be text segments)
			for _, seg := range nested.Segments {
				if seg.Type == SegmentMacro {
					current.node.Children = append(current.node.Children, seg.Macro)
				}
			}
			result.Warnings = append(result.Warnings, nested.Warnings...)
		}
	}

	if len(*stack) > 0 {
		parent := (*stack)[len(*stack)-1]
		// Add placeholder marker in parent's bodyContent to preserve position
		childIndex := len(parent.node.Children)
		placeholder := fmt.Sprintf("%s%d", childPlaceholderPrefix, childIndex)
		parent.bodyContent += placeholder
		// Add as child of parent macro
		parent.node.Children = append(parent.node.Children, current.node)
	} else {
		// Top level - add as segment
		result.AddMacroSegment(current.node)
	}
}

// reconstructBracketTag returns the original bracket syntax for a token.
// Uses OriginalTagText if available to preserve original formatting.
func reconstructBracketTag(token BracketToken) string {
	if token.OriginalTagText != "" {
		return token.OriginalTagText
	}
	// Fallback to reconstruction (shouldn't happen with current tokenizer)
	var sb strings.Builder
	sb.WriteString("[")
	if token.Type == BracketTokenCloseTag {
		sb.WriteString("/")
	}
	sb.WriteString(token.MacroName)
	for k, v := range token.Parameters {
		sb.WriteString(" ")
		sb.WriteString(k)
		sb.WriteString("=")
		if strings.Contains(v, " ") {
			sb.WriteString("\"")
			sb.WriteString(v)
			sb.WriteString("\"")
		} else {
			sb.WriteString(v)
		}
	}
	if token.Type == BracketTokenSelfClose {
		sb.WriteString("/")
	}
	sb.WriteString("]")
	return sb.String()
}

// reconstructOpenTag rebuilds the opening tag from a MacroNode.
func reconstructOpenTag(node *MacroNode) string {
	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(strings.ToUpper(node.Name))
	for k, v := range node.Parameters {
		sb.WriteString(" ")
		sb.WriteString(k)
		sb.WriteString("=")
		if strings.Contains(v, " ") {
			sb.WriteString("\"")
			sb.WriteString(v)
			sb.WriteString("\"")
		} else {
			sb.WriteString(v)
		}
	}
	sb.WriteString("]")
	return sb.String()
}
