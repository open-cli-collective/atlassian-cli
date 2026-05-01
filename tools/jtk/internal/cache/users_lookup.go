package cache

import (
	"context"
	"time"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// GetUserCacheFirst returns a user from the users cache when fresh and
// expand is empty (default mode), falling back to a live GetUser call
// otherwise.
//
// Non-empty expand (e.g., "groups,applicationRoles" for --extended) always
// goes live because the cache stores bulk-enumerated users without expanded
// fields like group/applicationRole counts.
//
// A fresh cache where the account ID is absent falls back to live — the
// users cache is a partial enumeration and may not contain all users.
func GetUserCacheFirst(ctx context.Context, client *api.Client, accountID, expand string) (*api.User, error) {
	if expand != "" {
		return client.GetUser(ctx, accountID, expand)
	}

	entry, err := Lookup("users")
	if err != nil {
		return client.GetUser(ctx, accountID, "")
	}

	env, err := ReadResource[[]api.User]("users")
	if err != nil {
		return client.GetUser(ctx, accountID, "")
	}

	switch Classify(env.FetchedAt, entry.TTL, time.Now()) {
	case StatusFresh, StatusManual:
		for i := range env.Data {
			if env.Data[i].AccountID == accountID {
				return &env.Data[i], nil
			}
		}
		return client.GetUser(ctx, accountID, "")
	case StatusStale, StatusUninitialized:
		return client.GetUser(ctx, accountID, "")
	case StatusUnavailable:
		return client.GetUser(ctx, accountID, "")
	}
	return client.GetUser(ctx, accountID, "")
}
