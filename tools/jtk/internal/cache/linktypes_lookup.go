package cache

import (
	"context"
	"time"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// GetLinkTypesCacheFirst returns []api.IssueLinkType from the linktypes cache
// when fresh, falling back to a live client.GetIssueLinkTypes call otherwise.
//
// Follows the same pattern as GetFieldsCacheFirst — see that function's doc
// comment for design rationale. Freshness is checked against the registry TTL.
func GetLinkTypesCacheFirst(ctx context.Context, client *api.Client) ([]api.IssueLinkType, error) {
	entry, err := Lookup("linktypes")
	if err != nil {
		return client.GetIssueLinkTypes(ctx)
	}

	env, err := ReadResource[[]api.IssueLinkType]("linktypes")
	if err != nil {
		return client.GetIssueLinkTypes(ctx)
	}

	switch Classify(env.FetchedAt, entry.TTL, time.Now()) {
	case StatusFresh, StatusManual:
		return env.Data, nil
	case StatusStale, StatusUninitialized:
		return client.GetIssueLinkTypes(ctx)
	case StatusUnavailable:
		return client.GetIssueLinkTypes(ctx)
	}
	return client.GetIssueLinkTypes(ctx)
}
