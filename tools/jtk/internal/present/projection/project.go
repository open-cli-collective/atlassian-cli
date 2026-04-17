package projection

import (
	"sort"

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
		headerIndex[h] = i
	}

	keepIdx := make([]int, 0, len(selected))
	headers := make([]string, 0, len(selected))
	for _, c := range selected {
		idx, ok := headerIndex[c.Header]
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
		labelIndex[lower(f.Label)] = f
	}
	out := make([]present.Field, 0, len(selected))
	for _, c := range selected {
		if f, ok := labelIndex[lower(c.Header)]; ok {
			out = append(out, f)
		}
	}
	return &present.DetailSection{Fields: out}
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

func lower(s string) string {
	// Small helper to avoid importing strings into this file's hot path
	// repeatedly. Go compiler inlines strings.ToLower reliably; kept here
	// for readability at call sites.
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
