package projection

import (
	"sort"
	"strings"

	"github.com/open-cli-collective/atlassian-go/present"
)

// ProjectTable returns a new TableSection containing only the columns whose
// spec Header matches one of selected, in the order of selected. Input is
// not mutated. Unmatched header positions are dropped.
//
// selected is the result of Resolve; it already has identity prepended and
// is in the user's token order. Callers should only invoke ProjectTable
// when projectionApplied is true.
func ProjectTable(section *present.TableSection, selected []ColumnSpec) *present.TableSection {
	if section == nil {
		return nil
	}
	headerIndex := make(map[string]int, len(section.Headers))
	for i, h := range section.Headers {
		headerIndex[strings.ToLower(h)] = i
	}

	keepIdx := make([]int, 0, len(selected))
	headers := make([]string, 0, len(selected))
	for _, c := range selected {
		idx, ok := headerIndex[strings.ToLower(c.Header)]
		if !ok {
			continue
		}
		keepIdx = append(keepIdx, idx)
		headers = append(headers, section.Headers[idx])
	}

	rows := make([]present.Row, len(section.Rows))
	for i, r := range section.Rows {
		cells := make([]string, 0, len(keepIdx))
		for _, idx := range keepIdx {
			if idx < len(r.Cells) {
				cells = append(cells, r.Cells[idx])
			} else {
				cells = append(cells, "")
			}
		}
		rows[i] = present.Row{Cells: cells}
	}

	return &present.TableSection{Headers: headers, Rows: rows}
}

// ProjectDetail returns a new DetailSection keeping only fields whose Label
// matches a selected spec Header (case-insensitive). Input is not mutated.
// Order follows selected, not the original DetailSection order.
func ProjectDetail(section *present.DetailSection, selected []ColumnSpec) *present.DetailSection {
	if section == nil {
		return nil
	}
	labelIndex := make(map[string]present.Field, len(section.Fields))
	for _, f := range section.Fields {
		labelIndex[strings.ToLower(f.Label)] = f
	}
	out := make([]present.Field, 0, len(selected))
	for _, c := range selected {
		if f, ok := labelIndex[strings.ToLower(c.Header)]; ok {
			out = append(out, f)
		}
	}
	return &present.DetailSection{Fields: out}
}

// ApplyToTableInModel rewrites the first TableSection of model in place to
// keep only the selected columns. No-op when the model carries no
// TableSection (e.g., an all-MessageSection model). Commands pass the
// OutputModel a presenter built plus the `selected` slice from Resolve —
// this helper was duplicated across cmd/issues, cmd/users, and cmd/projects
// before extraction.
func ApplyToTableInModel(model *present.OutputModel, selected []ColumnSpec) {
	if model == nil {
		return
	}
	for i, s := range model.Sections {
		if ts, ok := s.(*present.TableSection); ok {
			model.Sections[i] = ProjectTable(ts, selected)
			return
		}
	}
}

// ApplyToDetailInModel is the DetailSection counterpart of
// ApplyToTableInModel. No-op when the model carries no DetailSection.
func ApplyToDetailInModel(model *present.OutputModel, selected []ColumnSpec) {
	if model == nil {
		return
	}
	for i, s := range model.Sections {
		if ds, ok := s.(*present.DetailSection); ok {
			model.Sections[i] = ProjectDetail(ds, selected)
			return
		}
	}
}

// HasExtendedFields returns true if any of the selected specs correspond to
// Extended-only fields in the registry. This lets commands gate expensive
// fetches on whether the user explicitly requested extended fields via
// --fields, not just whether --extended is active.
func HasExtendedFields(selected []ColumnSpec, registry Registry) bool {
	extendedHeaders := make(map[string]struct{})
	for _, c := range registry {
		if c.Extended {
			extendedHeaders[c.Header] = struct{}{}
		}
	}
	for _, c := range selected {
		if _, ok := extendedHeaders[c.Header]; ok {
			return true
		}
	}
	return false
}

// DeriveFetchFields returns the minimum Jira field-ID set needed to render
// the selected specs. Output is sorted and deduplicated so API requests are
// stable across runs. Synthetic specs (empty FieldID, empty Fetch) are
// ignored.
//
// Identity is expected to be part of selected already (Resolve prepends it).
// No special-casing here.
func DeriveFetchFields(selected []ColumnSpec) []string {
	seen := make(map[string]struct{}, len(selected)*2)
	for _, c := range selected {
		for _, f := range c.fetchFields() {
			seen[f] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for f := range seen {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}
