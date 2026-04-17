package issues

import (
	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// deriveFetchFields computes the Jira API "fields" list for issues list /
// issues search based on the current command state. Unlike the legacy
// resolveFields helper, it does NOT accept user --fields input — the
// display projection is resolved upstream via projection.Resolve, and
// fetch-pruning is a derived optimization.
//
// Precedence:
//  1. outputFormat == "json" → "*all" (preserves full payload fidelity).
//  2. projected → projection.DeriveFetchFields(selected) (both extended
//     and allFields are ignored; the selected specs alone drive fetch).
//  3. extended || allFields → api.DefaultSearchFields.
//  4. otherwise → api.ListSearchFields.
func deriveFetchFields(
	selected []projection.ColumnSpec,
	projected bool,
	extended bool,
	allFields bool,
	outputFormat string,
) []string {
	if outputFormat == "json" {
		return []string{"*all"}
	}
	if projected {
		return projection.DeriveFetchFields(selected)
	}
	if extended || allFields {
		return append([]string(nil), api.DefaultSearchFields...)
	}
	return append([]string(nil), api.ListSearchFields...)
}
