// Package artifact provides types and helpers for output artifact projection.
// Commands use these types to produce intentional, curated output rather than
// raw API responses.
//
// The artifact package is pure types and helpers. Rendering lives in the view package.
package artifact

// Type represents an output artifact type.
type Type int

const (
	// Agent is the default artifact type: action-oriented, LLM-friendly output
	// with essential fields for triage and next actions.
	Agent Type = iota

	// Full is inspection-oriented output with additional fields like dates,
	// authors, and version information.
	Full
)

// String returns the string representation of the artifact type.
func (t Type) String() string {
	switch t {
	case Agent:
		return "agent"
	case Full:
		return "full"
	default:
		return "unknown"
	}
}

// Mode determines artifact type from the --full flag.
func Mode(full bool) Type {
	if full {
		return Full
	}
	return Agent
}

// IsAgent returns true if the artifact type is Agent.
func (t Type) IsAgent() bool { return t == Agent }

// IsFull returns true if the artifact type is Full.
func (t Type) IsFull() bool { return t == Full }
