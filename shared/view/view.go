// Package view provides output formatting for Atlassian CLI tools.
package view

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
)

// Format represents an output format.
type Format string

// Output format constants.
const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatPlain Format = "plain"
)

// ValidFormats returns the list of valid output formats.
func ValidFormats() []string {
	return []string{string(FormatTable), string(FormatJSON), string(FormatPlain)}
}

// ValidateFormat checks if a format string is valid.
// Returns an error if the format is not supported.
func ValidateFormat(format string) error {
	switch format {
	case "", string(FormatTable), string(FormatJSON), string(FormatPlain):
		return nil
	default:
		return fmt.Errorf("invalid output format: %q (valid formats: table, json, plain)", format)
	}
}

// View handles output formatting.
type View struct {
	Format  Format
	NoColor bool
	Out     io.Writer
	Err     io.Writer
}

// New creates a new View with the given format.
// If noColor is true, colorized output is disabled.
func New(format Format, noColor bool) *View {
	if noColor {
		color.NoColor = true
	}

	return &View{
		Format:  format,
		NoColor: noColor,
		Out:     os.Stdout,
		Err:     os.Stderr,
	}
}

// NewWithFormat creates a new View from a format string.
// This is a convenience function that accepts string instead of Format.
func NewWithFormat(format string, noColor bool) *View {
	return New(Format(format), noColor)
}

// SetOutput sets the output writer.
func (v *View) SetOutput(w io.Writer) {
	v.Out = w
}

// SetError sets the error writer.
func (v *View) SetError(w io.Writer) {
	v.Err = w
}

// Table renders data as a formatted table with aligned columns.
// For JSON format, use the JSON method instead.
func (v *View) Table(headers []string, rows [][]string) error {
	if v.Format == FormatJSON {
		return v.tableAsJSON(headers, rows)
	}

	if v.Format == FormatPlain {
		return v.Plain(rows)
	}

	w := tabwriter.NewWriter(v.Out, 0, 0, 2, ' ', 0)

	// Print headers with bold formatting
	headerLine := strings.Join(headers, "\t")
	if v.NoColor {
		_, _ = fmt.Fprintln(w, headerLine)
	} else {
		_, _ = fmt.Fprintln(w, color.New(color.Bold).Sprint(headerLine))
	}

	// Print rows
	for _, row := range rows {
		_, _ = fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	return w.Flush()
}

// tableAsJSON renders table data as JSON array of objects.
func (v *View) tableAsJSON(headers []string, rows [][]string) error {
	results := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		item := make(map[string]string)
		for i, header := range headers {
			if i < len(row) {
				item[strings.ToLower(header)] = row[i]
			}
		}
		results = append(results, item)
	}
	return v.JSON(results)
}

// JSON renders data as formatted JSON.
func (v *View) JSON(data interface{}) error {
	enc := json.NewEncoder(v.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// Plain renders rows as tab-separated values without headers.
func (v *View) Plain(rows [][]string) error {
	for _, row := range rows {
		_, _ = fmt.Fprintln(v.Out, strings.Join(row, "\t"))
	}
	return nil
}

// Render renders data based on the current format.
// For table format, uses headers and rows.
// For JSON format, uses jsonData.
// For plain format, uses rows without headers.
func (v *View) Render(headers []string, rows [][]string, jsonData interface{}) error {
	switch v.Format {
	case FormatJSON:
		return v.JSON(jsonData)
	case FormatPlain:
		return v.Plain(rows)
	default:
		return v.Table(headers, rows)
	}
}

// Success prints a success message with a green checkmark.
func (v *View) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if v.NoColor {
		_, _ = fmt.Fprintln(v.Out, "✓ "+msg)
	} else {
		_, _ = fmt.Fprintln(v.Out, color.GreenString("✓ %s", msg))
	}
}

// Error prints an error message with a red X.
func (v *View) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if v.NoColor {
		_, _ = fmt.Fprintln(v.Err, "✗ "+msg)
	} else {
		_, _ = fmt.Fprintln(v.Err, color.RedString("✗ %s", msg))
	}
}

// Warning prints a warning message with a yellow warning sign.
func (v *View) Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if v.NoColor {
		_, _ = fmt.Fprintln(v.Err, "⚠ "+msg)
	} else {
		_, _ = fmt.Fprintln(v.Err, color.YellowString("⚠ %s", msg))
	}
}

// Info prints an informational message.
func (v *View) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintln(v.Out, msg)
}

// Print prints a message without newline.
func (v *View) Print(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(v.Out, format, args...)
}

// Println prints a message with newline.
func (v *View) Println(format string, args ...interface{}) {
	_, _ = fmt.Fprintln(v.Out, fmt.Sprintf(format, args...))
}

// Truncate truncates a string to the specified length, adding "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
