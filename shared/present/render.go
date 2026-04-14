package present

import (
	"bytes"
	"strings"
	"text/tabwriter"
)

// Render converts an OutputModel to a string based on the given style.
// This is a pure function with no side effects.
func Render(model *OutputModel, style Style) string {
	var buf bytes.Buffer
	for _, section := range model.Sections {
		buf.WriteString(renderSection(section, style))
	}
	return buf.String()
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
	// Headers
	buf.WriteString(strings.Join(sec.Headers, " | "))
	buf.WriteByte('\n')
	// Rows - cells are already normalized by presenter
	for _, row := range sec.Rows {
		buf.WriteString(strings.Join(row.Cells, " | "))
		buf.WriteByte('\n')
	}
	return buf.String()
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
	case MessageInfo:
		return sec.Message + "\n"
	}
	return sec.Message + "\n"
}
