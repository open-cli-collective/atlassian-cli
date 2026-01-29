// parser.go defines shared types for macro parsing in both directions.
package md

import (
	"fmt"
	"log"
)

// SegmentType indicates whether a segment is text or a macro.
type SegmentType int

const (
	SegmentText  SegmentType = iota // plain text/HTML content
	SegmentMacro                    // parsed macro node
)

// Segment represents either text content or a parsed macro.
type Segment struct {
	Type  SegmentType
	Text  string     // set when Type == SegmentText
	Macro *MacroNode // set when Type == SegmentMacro
}

// ParseResult contains the parsed output: a sequence of segments
// that alternate between text content and macro nodes.
type ParseResult struct {
	Segments []Segment
	Warnings []string // any warnings generated during parsing
}

// AddTextSegment appends a text segment, merging with previous text if possible.
func (pr *ParseResult) AddTextSegment(text string) {
	if text == "" {
		return
	}
	// Merge adjacent text segments
	if len(pr.Segments) > 0 && pr.Segments[len(pr.Segments)-1].Type == SegmentText {
		pr.Segments[len(pr.Segments)-1].Text += text
		return
	}
	pr.Segments = append(pr.Segments, Segment{
		Type: SegmentText,
		Text: text,
	})
}

// AddMacroSegment appends a macro segment.
func (pr *ParseResult) AddMacroSegment(macro *MacroNode) {
	pr.Segments = append(pr.Segments, Segment{
		Type:  SegmentMacro,
		Macro: macro,
	})
}

// AddWarning logs a warning and stores it in the result.
func (pr *ParseResult) AddWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	pr.Warnings = append(pr.Warnings, msg)
	log.Printf("WARN: "+format, args...)
}

// GetMacros returns all MacroNodes from the parse result.
func (pr *ParseResult) GetMacros() []*MacroNode {
	var macros []*MacroNode
	for _, seg := range pr.Segments {
		if seg.Type == SegmentMacro && seg.Macro != nil {
			macros = append(macros, seg.Macro)
		}
	}
	return macros
}
