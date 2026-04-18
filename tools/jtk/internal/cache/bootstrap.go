package cache

import (
	"context"
	"fmt"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// seedUsersPageSize is the page size used when paginating the users endpoint.
// Jira accepts up to 100 per call; smaller chunks just mean more round-trips.
const seedUsersPageSize = 100

// seedUsersMax is a safety ceiling. Jira's platform caps user enumeration at
// ~1000 users regardless of what the client requests; this guard prevents a
// misbehaving server from spinning forever.
const seedUsersMax = 2000

// SeedUsers paginates the users endpoint and writes the "users" envelope.
//
// Used by both `jtk refresh users` and (via the shared cache package) by `jtk init`
// in #246. Returns the number of users seeded.
//
// Accepted limitation: Jira's user-search platform endpoint returns at most ~1000
// users. Large instances will have partial coverage; #246's sampling-based
// enhancements are the path to broader coverage.
func SeedUsers(ctx context.Context, c *api.Client) (int, error) {
	if c == nil {
		return 0, fmt.Errorf("api client is nil")
	}

	all := make([]api.User, 0, seedUsersPageSize)
	startAt := 0
	for startAt < seedUsersMax {
		page, err := c.ListUsersPage(ctx, startAt, seedUsersPageSize)
		if err != nil {
			return 0, fmt.Errorf("listing users page (startAt=%d): %w", startAt, err)
		}
		if len(page) == 0 {
			break
		}
		all = append(all, page...)
		if len(page) < seedUsersPageSize {
			// short page => last page
			break
		}
		startAt += len(page)
	}

	// Look up the TTL from the registry rather than hardcoding it — keeps the
	// registry as the single source of truth for cache metadata.
	ttl := usersTTL()
	if err := WriteResource("users", ttl, all); err != nil {
		return 0, fmt.Errorf("writing users envelope: %w", err)
	}
	return len(all), nil
}

// usersTTL returns the registered TTL for the "users" cache, falling back to
// "24h" if the registry hasn't been populated (e.g., during early init in
// tests that swap the registry).
func usersTTL() string {
	if e, err := Lookup("users"); err == nil {
		return e.TTL
	}
	return "24h"
}
