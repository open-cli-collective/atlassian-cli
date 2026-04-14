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

	"github.com/open-cli-collective/atlassian-go/artifact"
)

// Format represents an output format.
type Format string

// Output format constants.
const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatPlain Format = "plain"
)

// RenderPolicy controls output formatting style.
type RenderPolicy int

// Rendering policy constants.
const (
	PolicyDefault RenderPolicy = iota // Human-oriented (padded tables, ✓ decorators)
	PolicyAgent                       // Token-efficient (pipe-delimited, plain text)
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
	Policy  RenderPolicy // Zero value is PolicyDefault
	Out     io.Writer
	Err     io.Writer
}

// New creates a new View with the given format.
// If noColor is true, colorized output is disabled.
func New(format Format, noColor bool) *View {
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

// SetPolicy sets the rendering policy.
func (v *View) SetPolicy(p RenderPolicy) {
	v.Policy = p
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

	if v.Policy == PolicyAgent {
		return v.agentTable(headers, rows)
	}
	return v.humanTable(headers, rows)
}

// agentTable renders a pipe-delimited table for token-efficient agent output.
// Escapes pipes in cell content to avoid delimiter confusion.
func (v *View) agentTable(headers []string, rows [][]string) error {
	_, _ = fmt.Fprintln(v.Out, strings.Join(headers, " | "))
	for _, row := range rows {
		normalized := make([]string, len(row))
		for i, cell := range row {
			// Normalize newlines to spaces, escape pipes to avoid delimiter confusion
			cell = strings.ReplaceAll(cell, "\n", " ")
			cell = strings.ReplaceAll(cell, "|", "\\|")
			normalized[i] = cell
		}
		_, _ = fmt.Fprintln(v.Out, strings.Join(normalized, " | "))
	}
	return nil
}

// humanTable renders a padded table using tabwriter for human-readable output.
func (v *View) humanTable(headers []string, rows [][]string) error {
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
func (v *View) JSON(data any) error {
	enc := json.NewEncoder(v.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// RenderArtifact outputs an intentional artifact as JSON.
// Callers should check v.Format == FormatJSON before calling.
func (v *View) RenderArtifact(data any) error {
	enc := json.NewEncoder(v.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// RenderArtifactList outputs a list of artifacts with metadata.
// Callers should check v.Format == FormatJSON before calling.
func (v *View) RenderArtifactList(result *artifact.ListResult) error {
	enc := json.NewEncoder(v.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
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
func (v *View) Render(headers []string, rows [][]string, jsonData any) error {
	switch v.Format {
	case FormatJSON:
		return v.JSON(jsonData)
	case FormatPlain:
		return v.Plain(rows)
	case FormatTable:
		return v.Table(headers, rows)
	default:
		return v.Table(headers, rows)
	}
}

// Success prints a success message with a green checkmark.
// With PolicyAgent, prints plain text without the checkmark.
func (v *View) Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if v.Policy == PolicyAgent {
		_, _ = fmt.Fprintln(v.Out, msg)
		return
	}
	if v.NoColor {
		_, _ = fmt.Fprintln(v.Out, "✓ "+msg)
	} else {
		_, _ = fmt.Fprintln(v.Out, color.GreenString("✓ %s", msg))
	}
}

// Error prints an error message with a red X.
func (v *View) Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if v.NoColor {
		_, _ = fmt.Fprintln(v.Err, "✗ "+msg)
	} else {
		_, _ = fmt.Fprintln(v.Err, color.RedString("✗ %s", msg))
	}
}

// Warning prints a warning message with a yellow warning sign.
func (v *View) Warning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if v.Policy == PolicyAgent {
		_, _ = fmt.Fprintln(v.Err, msg)
		return
	}
	if v.NoColor {
		_, _ = fmt.Fprintln(v.Err, "⚠ "+msg)
	} else {
		_, _ = fmt.Fprintln(v.Err, color.YellowString("⚠ %s", msg))
	}
}

// Info prints an informational message.
func (v *View) Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintln(v.Out, msg)
}

// Print prints a message without newline.
func (v *View) Print(format string, args ...any) {
	_, _ = fmt.Fprintf(v.Out, format, args...)
}

// Println prints a message with newline.
func (v *View) Println(format string, args ...any) {
	_, _ = fmt.Fprintln(v.Out, fmt.Sprintf(format, args...))
}

// ListMeta contains pagination metadata for list results.
type ListMeta struct {
	Count   int  `json:"count"`
	HasMore bool `json:"hasMore"`
}

// ListResponse wraps list results with metadata for JSON output.
type ListResponse struct {
	Results []map[string]string `json:"results"`
	Meta    ListMeta            `json:"_meta"`
}

// RenderList renders tabular data with pagination metadata.
// For JSON output, wraps results in an object with _meta field.
// For other formats, delegates to Table.
func (v *View) RenderList(headers []string, rows [][]string, hasMore bool) error {
	if v.Format == FormatJSON {
		return v.renderListAsJSON(headers, rows, hasMore)
	}
	return v.Table(headers, rows)
}

func (v *View) renderListAsJSON(headers []string, rows [][]string, hasMore bool) error {
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

	response := ListResponse{
		Results: results,
		Meta: ListMeta{
			Count:   len(results),
			HasMore: hasMore,
		},
	}

	return v.JSON(response)
}

// RenderKeyValue renders a key-value pair.
// For JSON format, outputs as a JSON object.
// For other formats, outputs as "key: value" with bold key.
func (v *View) RenderKeyValue(key, value string) {
	if v.Format == FormatJSON {
		_, _ = fmt.Fprintf(v.Out, `{"%s": "%s"}`+"\n", key, value)
		return
	}
	if v.NoColor {
		_, _ = fmt.Fprintf(v.Out, "%s: %s\n", key, value)
	} else {
		bold := color.New(color.Bold)
		_, _ = bold.Fprintf(v.Out, "%s: ", key)
		_, _ = fmt.Fprintln(v.Out, value)
	}
}

// KeyValue represents a single key-value pair for rendering.
type KeyValue struct {
	Key   string
	Value string
}

// RenderKeyValues renders multiple key-value pairs for default/table/plain text output.
// Delegates to RenderKeyValue() for each pair to maintain single source of truth.
// Note: This helper is NOT for JSON output. Commands should handle JSON at the command
// level (branching on v.Format == FormatJSON before calling this).
func (v *View) RenderKeyValues(pairs []KeyValue) {
	for _, kv := range pairs {
		v.RenderKeyValue(kv.Key, kv.Value)
	}
}

// RenderText renders plain text.
func (v *View) RenderText(text string) {
	_, _ = fmt.Fprintln(v.Out, text)
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
