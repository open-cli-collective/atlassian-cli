// Package artifact provides output artifact types and projection functions
// for jtk commands. Artifacts are intentionally-shaped output structures
// that support agent (action-oriented) and full (inspection-oriented) modes.
//
// Each resource type has:
//   - An artifact struct with agent fields and full-only fields (omitempty)
//   - A Project<Type> function: (domain, mode) -> artifact
//   - A Project<Type>s helper for slices
package artifact
