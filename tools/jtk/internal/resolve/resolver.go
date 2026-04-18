// Package resolve turns human-readable entity references into canonical API
// entities using the instance cache populated by `jtk init` / `jtk refresh`.
//
// Resolution is cache-authoritative: after one targeted refresh on cache
// miss, the cache is the source of truth. There is no live-API fallback
// beyond the "me" special case on User and the refresh-once retry.
package resolve

import (
	"context"
	"fmt"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

// Resolver resolves human-readable references to canonical entities.
type Resolver struct {
	client *api.Client
}

// New constructs a Resolver bound to the given API client. The client is used
// for the "me" special case and for cache-refresh fetchers.
func New(client *api.Client) *Resolver {
	return &Resolver{client: client}
}

// refreshResource runs the registered fetcher for `name`, including its
// transitive dependencies. Resolvers call this once on cache miss before
// giving up with a NotFoundError.
func (r *Resolver) refreshResource(ctx context.Context, name string) error {
	entries, err := cache.SelectWithDeps([]string{name})
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsAvailable(r.client) {
			return fmt.Errorf("%s cache unavailable (scope restriction)", e.Name)
		}
		if _, err := e.Fetch(ctx, r.client); err != nil {
			return fmt.Errorf("refreshing %s: %w", e.Name, err)
		}
	}
	return nil
}

// resolveEntity is the shared lookup → (shape-pass-through | refresh → lookup)
// flow per #230's resolution rules.
//
//   - `lookup` inspects the cache and returns:
//
//   - (result, nil) on unique match.
//
//   - (_, errCacheEmpty) when the envelope doesn't exist.
//
//   - (_, errNoMatch) when the envelope exists but no row matches.
//
//   - (_, *AmbiguousMatchError) on multi-match (returned immediately).
//
//   - (_, other) on I/O failure (propagated).
//
//   - `passThrough` returns (synthetic, true) if the input satisfies the
//     entity's literal-ID shape heuristic. Shape-match inputs short-circuit
//     the refresh — an ID-shaped token that isn't in the cache still passes
//     through to the API so fresh installs, offline use, and projects/users
//     outside the cache horizon keep working.
//
//   - `makeNotFound` builds the entity-specific NotFoundError used when
//     neither cache lookup matches and the input isn't shape-eligible for
//     pass-through.
func resolveEntity[T any](
	ctx context.Context,
	r *Resolver,
	resource string,
	lookup func() (T, error),
	passThrough func() (T, bool),
	makeNotFound func() error,
) (T, error) {
	var zero T

	result, err := lookup()
	if err == nil {
		return result, nil
	}
	if !isMissOrNoMatch(err) {
		return zero, err
	}

	// Shape-based pass-through wins over refresh: if the input looks like
	// a raw ID, the API is the authority — refreshing just to learn "this
	// is an ID not a name" would be wasteful, and we want the CLI to work
	// without a populated cache.
	if v, ok := passThrough(); ok {
		return v, nil
	}

	if rerr := r.refreshResource(ctx, resource); rerr != nil {
		return zero, rerr
	}

	result, err = lookup()
	if err == nil {
		return result, nil
	}
	if !isMissOrNoMatch(err) {
		return zero, err
	}
	return zero, makeNotFound()
}
