package issues

import (
	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// deriveFetchFields computes the Jira API "fields" list for issues list /
// issues search based on the current command state.
//
// Precedence:
//  1. projected → projection.DeriveFetchFields(selected).
//  2. otherwise → api.ListSearchFields.
func deriveFetchFields(selected []projection.ColumnSpec, projected bool) []string {
	if projected {
		return projection.DeriveFetchFields(selected)
	}
	return append([]string(nil), api.ListSearchFields...)
}
