// Package present provides presentation models and pure rendering for CLI output.
package present

// RenderMode is the authoritative rendering configuration.
// Both legacy view.Policy and new present.Style derive from this.
//
// RenderMode exists at the CLI root/options level as the single source of truth.
// Tool-specific options (e.g., root.Options.RenderMode()) return this type.
type RenderMode int

// RenderMode constants.
const (
	RenderModeHuman RenderMode = iota // Padded tables, decorators
	RenderModeAgent                   // Pipe-delimited, plain text, token-efficient
)

// Style controls rendering format for the pure Render() function.
//
// Style is derived from RenderMode via StyleFromMode(). While currently 1:1
// with RenderMode, Style is a separate type to allow the renderer API to
// evolve independently of CLI configuration. For example, the renderer could
// add finer-grained styles (e.g., StyleCompact) without changing the CLI mode.
//
// During migration from view.View to present.Render, both view.RenderPolicy
// and present.Style coexist. They derive from the same RenderMode, ensuring
// a single knob controls both legacy and new paths. See root.Options.RenderMode().
type Style int

// Style constants.
const (
	StyleHuman Style = iota // Padded tables, decorators (checkmark, warning)
	StyleAgent              // Pipe-delimited, plain text, token-efficient
)

// StyleFromMode converts RenderMode to Style.
// Currently a 1:1 mapping; may diverge as rendering needs evolve.
func StyleFromMode(m RenderMode) Style {
	if m == RenderModeAgent {
		return StyleAgent
	}
	return StyleHuman
}

// RenderedOutput contains the rendered text for each output stream.
type RenderedOutput struct {
	Stdout string // Primary result/artifact
	Stderr string // Warnings, diagnostics, errors
}

// OutputModel is the complete presentation output for a command.
type OutputModel struct {
	Sections []Section
}

// Section is a polymorphic output section.
type Section interface {
	sectionMarker()
}

// DetailSection displays key-value pairs (single record detail view).
type DetailSection struct {
	Fields []Field
}

func (*DetailSection) sectionMarker() {}

// Field is a labeled value.
type Field struct {
	Label string
	Value string
}

// TableSection displays tabular data (list views).
type TableSection struct {
	Headers []string
	Rows    []Row
}

func (*TableSection) sectionMarker() {}

// Row is a table row.
type Row struct {
	Cells []string
}

// Stream indicates which output stream a section targets.
type Stream int

// Stream constants.
const (
	StreamStdout Stream = iota // Primary result/artifact
	StreamStderr               // Diagnostics, advisory, progress, commentary
)

// MessageSection displays a status message (mutations, confirmations).
type MessageSection struct {
	Kind    MessageKind
	Message string
	Stream  Stream // Explicit stream routing (zero value = StreamStdout)
}

func (*MessageSection) sectionMarker() {}

// MessageKind indicates the type of status message.
type MessageKind int

// MessageKind constants for status message types.
const (
	MessageInfo MessageKind = iota
	MessageSuccess
	MessageWarning
	MessageError
)
