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
	Compact bool
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
// When Compact is true, null fields, avatar URLs, self-links, and
// Atlassian metadata keys (_links, _expandable) are stripped to
// reduce output size for agent/LLM consumers.
func (v *View) JSON(data any) error {
	if v.Compact {
		data = compactData(data)
	}
	enc := json.NewEncoder(v.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// RenderArtifact outputs an intentional artifact as JSON.
// Unlike JSON(), never applies Compact post-processing because artifacts
// are already intentionally shaped by the command's projection function.
// Callers should check v.Format == FormatJSON before calling.
func (v *View) RenderArtifact(data any) error {
	enc := json.NewEncoder(v.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// RenderArtifactList outputs a list of artifacts with metadata.
// Unlike JSON(), never applies Compact post-processing because artifacts
// are already intentionally shaped by the command's projection function.
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
func (v *View) Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
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

// RenderText renders plain text.
func (v *View) RenderText(text string) {
	_, _ = fmt.Fprintln(v.Out, text)
}

// compactData converts data to a generic map/slice structure and strips
// verbose metadata that inflates output without adding useful content.
// Removes: null-valued fields, avatarUrls, self-link URLs, _links, _expandable,
// and any maps/slices left empty after pruning.
//
// Uses a JSON round-trip (marshal → unmarshal → walk) for simplicity over a
// reflection-based approach. The extra allocation is negligible for CLI output.
func compactData(data any) any {
	// Round-trip through JSON to get a generic map/slice structure.
	raw, err := json.Marshal(data)
	if err != nil {
		return data
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return data
	}
	return pruneValue(generic)
}

// Metadata keys stripped unconditionally in compact mode.
var compactStripKeys = map[string]bool{
	"avatarUrls":  true,
	"_links":      true,
	"_expandable": true,
}

func pruneValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		pruned := make(map[string]any, len(val))
		for k, child := range val {
			if child == nil {
				continue
			}
			if compactStripKeys[k] {
				continue
			}
			// Strip "self" keys that hold API URLs. Covers both Jira v3 (/rest/...)
			// and Confluence v2 (/wiki/api/v2/...) endpoints.
			if k == "self" {
				if s, ok := child.(string); ok && (strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")) {
					continue
				}
			}
			pv := pruneValue(child)
			// Drop fields that became empty due to pruning (e.g., a user
			// object that only had avatarUrls + self becomes {}).
			// Preserve originally-empty collections: [] means "no items"
			// and {} means "empty object" — both are semantically distinct
			// from an absent field.
			if wasNonEmpty(child) && isEmpty(pv) {
				continue
			}
			pruned[k] = pv
		}
		return pruned
	case []any:
		result := make([]any, 0, len(val))
		for _, item := range val {
			pv := pruneValue(item)
			if wasNonEmpty(item) && isEmpty(pv) {
				continue
			}
			result = append(result, pv)
		}
		return result
	default:
		return v
	}
}

// wasNonEmpty returns true if the value is a map or slice with at least one element.
func wasNonEmpty(v any) bool {
	switch val := v.(type) {
	case map[string]any:
		return len(val) > 0
	case []any:
		return len(val) > 0
	default:
		return false
	}
}

// isEmpty returns true for empty maps and empty slices.
func isEmpty(v any) bool {
	switch val := v.(type) {
	case map[string]any:
		return len(val) == 0
	case []any:
		return len(val) == 0
	default:
		return false
	}
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
