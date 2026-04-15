package present

import (
	"bytes"
	"strings"
	"text/tabwriter"
)

// Render converts an OutputModel to RenderedOutput with proper stream routing.
// This is a pure function with no side effects.
func Render(model *OutputModel, style Style) RenderedOutput {
	if model == nil {
		return RenderedOutput{}
	}
	var stdout, stderr bytes.Buffer
	for _, section := range model.Sections {
		text := renderSection(section, style)
		if isStderrSection(section) {
			stderr.WriteString(text)
		} else {
			stdout.WriteString(text)
		}
	}
	return RenderedOutput{Stdout: stdout.String(), Stderr: stderr.String()}
}

// isStderrSection returns true if this section should go to stderr.
func isStderrSection(s Section) bool {
	if msg, ok := s.(*MessageSection); ok {
		return msg.Stream == StreamStderr // Explicit stream routing
	}
	return false // DetailSection, TableSection -> stdout
}

func renderSection(s Section, style Style) string {
	switch sec := s.(type) {
	case *DetailSection:
		return renderDetail(sec)
	case *TableSection:
		return renderTable(sec, style)
	case *MessageSection:
		return renderMessage(sec, style)
	}
	return ""
}

func renderDetail(sec *DetailSection) string {
	var buf bytes.Buffer
	for _, f := range sec.Fields {
		// Both styles use "Label: Value\n" for key-value pairs
		buf.WriteString(f.Label)
		buf.WriteString(": ")
		buf.WriteString(f.Value)
		buf.WriteByte('\n')
	}
	return buf.String()
}

func renderTable(sec *TableSection, style Style) string {
	if style == StyleAgent {
		return renderAgentTable(sec)
	}
	return renderHumanTable(sec)
}

func renderAgentTable(sec *TableSection) string {
	var buf bytes.Buffer
	// Headers - escape pipes defensively
	escapedHeaders := make([]string, len(sec.Headers))
	for i, h := range sec.Headers {
		escapedHeaders[i] = escapeAgentCell(h)
	}
	buf.WriteString(strings.Join(escapedHeaders, " | "))
	buf.WriteByte('\n')
	// Rows - escape pipes and normalize newlines defensively
	for _, row := range sec.Rows {
		escapedCells := make([]string, len(row.Cells))
		for i, cell := range row.Cells {
			escapedCells[i] = escapeAgentCell(cell)
		}
		buf.WriteString(strings.Join(escapedCells, " | "))
		buf.WriteByte('\n')
	}
	return buf.String()
}

// escapeAgentCell escapes pipe characters and normalizes newlines in cell content.
// This prevents cell content from being structurally indistinguishable from delimiters.
func escapeAgentCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

func renderHumanTable(sec *TableSection) string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	// Headers
	_, _ = w.Write([]byte(strings.Join(sec.Headers, "\t") + "\n"))
	// Rows
	for _, row := range sec.Rows {
		_, _ = w.Write([]byte(strings.Join(row.Cells, "\t") + "\n"))
	}
	_ = w.Flush()
	return buf.String()
}

func renderMessage(sec *MessageSection, style Style) string {
	if style == StyleAgent {
		// Plain text, no decorators
		return sec.Message + "\n"
	}
	// Human style with decorators
	switch sec.Kind {
	case MessageSuccess:
		return "✓ " + sec.Message + "\n"
	case MessageWarning:
		return "⚠ " + sec.Message + "\n"
	case MessageError:
		return "✗ " + sec.Message + "\n"
	case MessageInfo:
		return sec.Message + "\n"
	}
	return sec.Message + "\n"
}
