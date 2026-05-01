package cache

import (
	"context"
	"strings"
	"time"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// GetSprintsCacheFirst returns all sprints for a board from the sprints cache
// when fresh, applying state filter locally. Falls back to live API otherwise.
//
// Only known single-state values ("active", "closed", "future") and empty
// string are served from cache. Comma-separated states and unknown values
// fall back to live to preserve Jira server-side semantics.
//
// The caller is responsible for sorting and pagination (runList already does
// this after receiving the full sprint list).
func GetSprintsCacheFirst(ctx context.Context, client *api.Client, boardID int, state string, fetcher func(context.Context, *api.Client, int, string) ([]api.Sprint, error)) ([]api.Sprint, error) {
	if !isCacheableState(state) {
		return fetcher(ctx, client, boardID, state)
	}

	entry, err := Lookup("sprints")
	if err != nil {
		return fetcher(ctx, client, boardID, state)
	}

	env, err := ReadResource[map[int][]api.Sprint]("sprints")
	if err != nil {
		return fetcher(ctx, client, boardID, state)
	}

	switch Classify(env.FetchedAt, entry.TTL, time.Now()) {
	case StatusFresh, StatusManual:
		sprints, ok := env.Data[boardID]
		if !ok {
			return fetcher(ctx, client, boardID, state)
		}
		if state == "" {
			return sprints, nil
		}
		var filtered []api.Sprint
		for _, s := range sprints {
			if strings.EqualFold(s.State, state) {
				filtered = append(filtered, s)
			}
		}
		return filtered, nil
	case StatusStale, StatusUninitialized:
		return fetcher(ctx, client, boardID, state)
	case StatusUnavailable:
		return fetcher(ctx, client, boardID, state)
	}
	return fetcher(ctx, client, boardID, state)
}

func isCacheableState(state string) bool {
	switch strings.ToLower(state) {
	case "", "active", "closed", "future":
		return true
	default:
		return false
	}
}
