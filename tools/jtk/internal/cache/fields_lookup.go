package cache

import (
	"context"
	"time"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// GetFieldsCacheFirst returns []api.Field from the fields cache when fresh,
// falling back to a live client.GetFields call otherwise.
//
// Fall-through cases (all result in a live call):
//   - Lookup("fields") fails (unexpected — registry not populated)
//   - ReadResource returns any error: cache miss, ErrNoInstance, I/O, decode
//   - freshness is StatusStale
//
// StatusFresh and StatusManual are treated as authoritative and avoid the
// network call entirely.
//
// Design note: this helper checks freshness against the registry TTL (not the
// envelope's stored TTL field) so it stays in sync with `jtk refresh --status`.
// It diverges from internal/resolve/* which accepts any non-miss as
// authoritative — here #274 requires that cache-first reads only serve fresh
// data. Sibling helpers for #276–#278 should follow the same pattern.
func GetFieldsCacheFirst(ctx context.Context, client *api.Client) ([]api.Field, error) {
	entry, err := Lookup("fields")
	if err != nil {
		return client.GetFields(ctx)
	}

	env, err := ReadResource[[]api.Field]("fields")
	if err != nil {
		return client.GetFields(ctx)
	}

	switch Classify(env.FetchedAt, entry.TTL, time.Now()) {
	case StatusFresh, StatusManual:
		return env.Data, nil
	case StatusStale, StatusUninitialized:
		return client.GetFields(ctx)
	case StatusUnavailable:
		// Classify never emits StatusUnavailable (that status is set by refresh
		// via Entry.IsAvailable, not by Classify). Listed explicitly so the
		// exhaustive linter is satisfied and future callers see the intent.
		return client.GetFields(ctx)
	}
	return client.GetFields(ctx)
}
