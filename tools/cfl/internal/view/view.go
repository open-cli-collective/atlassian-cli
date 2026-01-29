// Package view provides output formatting for cfl commands.
package view

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

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

// Renderer renders data in a specific format.
type Renderer struct {
	format  Format
	writer  io.Writer
	noColor bool
}

// NewRenderer creates a new renderer with the specified format.
func NewRenderer(format Format, noColor bool) *Renderer {
	if noColor {
		color.NoColor = true
	}
	return &Renderer{
		format:  format,
		writer:  os.Stdout,
		noColor: noColor,
	}
}

// SetWriter sets the output writer.
func (r *Renderer) SetWriter(w io.Writer) {
	r.writer = w
}

// RenderTable renders data as a table.
func (r *Renderer) RenderTable(headers []string, rows [][]string) {
	if r.format == FormatJSON {
		r.renderTableAsJSON(headers, rows)
		return
	}

	if r.format == FormatPlain {
		r.renderTableAsPlain(headers, rows)
		return
	}

	// Print header
	for i, h := range headers {
		if i > 0 {
			_, _ = fmt.Fprint(r.writer, "  ")
		}
		_, _ = fmt.Fprint(r.writer, h)
	}
	_, _ = fmt.Fprintln(r.writer)

	// Print rows
	for _, row := range rows {
		for i, val := range row {
			if i > 0 {
				_, _ = fmt.Fprint(r.writer, "  ")
			}
			_, _ = fmt.Fprint(r.writer, val)
		}
		_, _ = fmt.Fprintln(r.writer)
	}
}

func (r *Renderer) renderTableAsJSON(headers []string, rows [][]string) {
	var result []map[string]string
	for _, row := range rows {
		item := make(map[string]string)
		for i, header := range headers {
			if i < len(row) {
				item[strings.ToLower(header)] = row[i]
			}
		}
		result = append(result, item)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	_, _ = fmt.Fprintln(r.writer, string(data))
}

func (r *Renderer) renderTableAsPlain(_ []string, rows [][]string) {
	for _, row := range rows {
		for i, val := range row {
			if i > 0 {
				_, _ = fmt.Fprint(r.writer, "\t")
			}
			_, _ = fmt.Fprint(r.writer, val)
		}
		_, _ = fmt.Fprintln(r.writer)
	}
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
// For other formats, delegates to RenderTable.
func (r *Renderer) RenderList(headers []string, rows [][]string, hasMore bool) {
	if r.format == FormatJSON {
		r.renderListAsJSON(headers, rows, hasMore)
		return
	}
	// For non-JSON, delegate to existing RenderTable
	r.RenderTable(headers, rows)
}

func (r *Renderer) renderListAsJSON(headers []string, rows [][]string, hasMore bool) {
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

	data, _ := json.MarshalIndent(response, "", "  ")
	_, _ = fmt.Fprintln(r.writer, string(data))
}

// RenderJSON renders an object as JSON.
func (r *Renderer) RenderJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(r.writer, string(data))
	return nil
}

// RenderText renders plain text.
func (r *Renderer) RenderText(text string) {
	_, _ = fmt.Fprintln(r.writer, text)
}

// RenderKeyValue renders a key-value pair.
func (r *Renderer) RenderKeyValue(key, value string) {
	if r.format == FormatJSON {
		_, _ = fmt.Fprintf(r.writer, `{"%s": "%s"}`+"\n", key, value)
		return
	}
	bold := color.New(color.Bold)
	_, _ = bold.Fprintf(r.writer, "%s: ", key)
	_, _ = fmt.Fprintln(r.writer, value)
}

// Success prints a success message.
func (r *Renderer) Success(msg string) {
	green := color.New(color.FgGreen)
	_, _ = green.Fprintln(r.writer, "✓ "+msg)
}

// Error prints an error message.
func (r *Renderer) Error(msg string) {
	red := color.New(color.FgRed)
	_, _ = red.Fprintln(r.writer, "✗ "+msg)
}

// Warning prints a warning message.
func (r *Renderer) Warning(msg string) {
	yellow := color.New(color.FgYellow)
	_, _ = yellow.Fprintln(r.writer, "⚠ "+msg)
}

// Truncate truncates a string to the specified length.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
