// Package projection provides shared `--fields` column projection and
// name resolution for jtk read commands.
//
// Per the JTK Output Specification (#230), `--fields a,b,c` is a display
// projection on list/get commands: it restricts output columns/fields to
// those named, with name resolution that accepts column headers, Jira
// field IDs, or cached human names (in that order). KEY/ID is always
// retained on list outputs; the identifying line is always retained on
// get outputs.
//
// Each flagship presenter exports an ordered Registry of ColumnSpecs.
// Commands call Resolve to map the user's --fields tokens to specs, then
// ProjectTable or ProjectDetail to slice the already-built presentation
// model, and DeriveFetchFields to compute the minimum Jira field-ID set
// needed to render the selection.
package projection

import (
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// ColumnSpec describes one display column (table) or field (detail) for
// projection. Each registered spec knows its presentation header, any
// user-facing aliases, the Jira field ID (if any) that drives it, and
// whether it is the identity column (always retained) or an extended-only
// column (only available when --extended is active).
type ColumnSpec struct {
	Header   string
	Aliases  []string
	FieldID  string
	Identity bool
	Extended bool
	// Fetch lists Jira field IDs required to render this column.
	// When empty and FieldID is non-empty, [FieldID] is assumed.
	// When empty and FieldID is empty, the column contributes nothing
	// to the fetch set (synthetic columns like a constructed URL).
	Fetch []string
	// Dynamic is true for specs created at runtime from Jira field
	// metadata (cache-backed). Commands use this to append values
	// that the hardcoded presenter doesn't know about.
	Dynamic bool
}

// fetchFields returns the Jira field IDs required to render this spec.
// Synthetic specs (FieldID empty, Fetch empty) return nil.
func (c ColumnSpec) fetchFields() []string {
	if len(c.Fetch) > 0 {
		return c.Fetch
	}
	if c.FieldID != "" {
		return []string{c.FieldID}
	}
	return nil
}

// Registry is an ordered list of ColumnSpecs describing the full column
// set for one entity/output shape. The order is the default display order.
type Registry []ColumnSpec

// ForMode returns the registry filtered to the active mode. When extended
// is false, Extended specs are dropped.
func (r Registry) ForMode(extended bool) Registry {
	if extended {
		out := make(Registry, len(r))
		copy(out, r)
		return out
	}
	out := make(Registry, 0, len(r))
	for _, c := range r {
		if c.Extended {
			continue
		}
		out = append(out, c)
	}
	return out
}

// Match looks up a ColumnSpec by token. Resolution order:
//  1. Header match (case-insensitive).
//  2. Any Alias match (case-insensitive).
//  3. FieldID match (exact, case-sensitive — Jira field IDs are canonical).
//  4. api.Field.Name match against fields (case-insensitive). This is the
//     "human name" fallback. fields may be nil when the caller hasn't
//     fetched Jira metadata yet — in that case the fallback is skipped.
//
// Returns (spec, true) on success, (_, false) otherwise.
func (r Registry) Match(token string, fields []api.Field) (ColumnSpec, bool) {
	for _, c := range r {
		if strings.EqualFold(c.Header, token) {
			return c, true
		}
		for _, a := range c.Aliases {
			if strings.EqualFold(a, token) {
				return c, true
			}
		}
	}
	for _, c := range r {
		if c.FieldID != "" && c.FieldID == token {
			return c, true
		}
	}
	if len(fields) == 0 {
		return ColumnSpec{}, false
	}
	// Human-name fallback: scan ALL Jira fields whose Name matches the token
	// and return the first whose ID maps to a ColumnSpec. This avoids
	// cache-order dependency when multiple fields share a name but only
	// one is registered.
	for i := range fields {
		if strings.EqualFold(fields[i].Name, token) {
			fieldID := fields[i].ID
			for _, c := range r {
				if c.FieldID != "" && c.FieldID == fieldID {
					return c, true
				}
			}
		}
	}
	return ColumnSpec{}, false
}
