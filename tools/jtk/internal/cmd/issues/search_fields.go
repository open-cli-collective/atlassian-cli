package issues

import (
	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// deriveFetchFields computes the Jira API "fields" list for issues list /
// issues search based on the current command state.
//
// Precedence:
//  1. projected → projection.DeriveFetchFields(selected) (both extended
//     and allFields are ignored; the selected specs alone drive fetch).
//  2. extended || allFields → api.DefaultSearchFields.
//  3. otherwise → api.ListSearchFields.
func deriveFetchFields(
	selected []projection.ColumnSpec,
	projected bool,
	extended bool,
	allFields bool,
) []string {
	if projected {
		return projection.DeriveFetchFields(selected)
	}
	if extended || allFields {
		return append([]string(nil), api.DefaultSearchFields...)
	}
	return append([]string(nil), api.ListSearchFields...)
}
