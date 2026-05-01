package cache

import (
	"context"
	"time"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// GetIssueTypesCacheFirst returns []api.IssueType for a project from the
// issuetypes cache when fresh, falling back to a live API call otherwise.
//
// The issuetypes cache stores map[string][]api.IssueType keyed by project key.
// A fresh cache where the requested project key is absent triggers a live
// fallback — the project may have been added after the last refresh.
//
// Follows the same freshness-against-registry-TTL pattern as
// GetFieldsCacheFirst.
func GetIssueTypesCacheFirst(ctx context.Context, client *api.Client, projectKey string) ([]api.IssueType, error) {
	entry, err := Lookup("issuetypes")
	if err != nil {
		return client.GetProjectIssueTypes(ctx, projectKey)
	}

	env, err := ReadResource[map[string][]api.IssueType]("issuetypes")
	if err != nil {
		return client.GetProjectIssueTypes(ctx, projectKey)
	}

	switch Classify(env.FetchedAt, entry.TTL, time.Now()) {
	case StatusFresh, StatusManual:
		types, ok := env.Data[projectKey]
		if !ok {
			return client.GetProjectIssueTypes(ctx, projectKey)
		}
		return types, nil
	case StatusStale, StatusUninitialized:
		return client.GetProjectIssueTypes(ctx, projectKey)
	case StatusUnavailable:
		return client.GetProjectIssueTypes(ctx, projectKey)
	}
	return client.GetProjectIssueTypes(ctx, projectKey)
}
