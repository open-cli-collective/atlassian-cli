// Package cache provides caching functionality for jtk resources.
//
// Tier-1 (the on-disk envelope, atomic writes, freshness classification) is a
// thin facade over github.com/open-cli-collective/cli-common/cache — the
// shared state component (working-with-state.md §5b). The public API here is
// kept byte-stable so jtk's tier-2 (registry/fetchers/lookups/invalidate) and
// the ~25 downstream test files do not churn; only the internals delegate.
package cache

import (
	"errors"
	"fmt"

	cccache "github.com/open-cli-collective/cli-common/cache"
)

// Version is the on-disk envelope schema version (delegated to cli-common so
// a schema bump self-heals identically everywhere).
const Version = cccache.Version

// ErrCacheMiss is re-exported so existing errors.Is(err, cache.ErrCacheMiss)
// call sites keep matching.
var ErrCacheMiss = cccache.ErrCacheMiss

// Envelope is an alias for the shared envelope, so callers that reference
// cache.Envelope / .FetchedAt / .TTL / .Data are unchanged.
type Envelope[T any] = cccache.Envelope[T]

// ReadResource reads the envelope for name.
//   - (envelope, nil) on success.
//   - (zero, ErrCacheMiss) if absent / version- or identity-mismatched and no
//     valid legacy ~/.jtk/cache envelope can be promoted.
//   - (zero, error) on path/instance resolution, I/O, or decode failure.
//
// On a miss against the new (os.UserCacheDir()/jtk) root, exactly one valid
// legacy envelope is promoted in place if present (the one-time, per-resource,
// non-destructive re-migration — see migrate.go). Reads never check freshness.
func ReadResource[T any](name string) (Envelope[T], error) {
	loc, err := locator()
	if err != nil {
		return Envelope[T]{}, err
	}
	env, err := cccache.ReadResource[T](loc, name)
	if errors.Is(err, ErrCacheMiss) {
		if penv, ok := promoteLegacyOnMiss[T](loc, name); ok {
			return penv, nil
		}
		return Envelope[T]{}, ErrCacheMiss
	}
	return env, err
}

// WriteResource atomically writes an envelope for name. Resource, Instance,
// Version, and a fresh FetchedAt are set by the shared writer; ttl is the
// caller's hard-coded per-resource value.
func WriteResource[T any](name, ttl string, data T) error {
	loc, err := locator()
	if err != nil {
		return err
	}
	return cccache.WriteResource(loc, name, ttl, data)
}

// atomicWriteEnvelope writes a caller-supplied envelope verbatim (preserving
// its FetchedAt — used by writeRaw/Touch to persist the zeroed "stale"
// marker). The shared verbatim writer derives the file from env.Resource;
// the legacy first parameter is retained for call-site stability (writeRaw
// and the *_lookup_test.go helpers pass it positionally). The name ==
// env.Resource invariant is enforced (not just assumed) so a future caller
// passing a mismatched name fails loudly instead of silently writing to the
// wrong cache file.
func atomicWriteEnvelope[T any](name string, env Envelope[T]) error {
	if name != "" && name != env.Resource {
		return fmt.Errorf("cache: atomicWriteEnvelope name %q does not match envelope resource %q", name, env.Resource)
	}
	loc, err := locator()
	if err != nil {
		return err
	}
	return cccache.WriteEnvelope(loc, env)
}
