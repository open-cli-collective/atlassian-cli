package issues

import (
	"context"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

// fieldsFetcher returns a function that satisfies the projection.Resolve
// fetchFields parameter by reading from the fields cache first (fresh only)
// and falling back to a live client.GetFields call.
func fieldsFetcher(client *api.Client) func(context.Context) ([]api.Field, error) {
	return func(ctx context.Context) ([]api.Field, error) {
		return cache.GetFieldsCacheFirst(ctx, client)
	}
}
