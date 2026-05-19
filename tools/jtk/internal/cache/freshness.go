package cache

import (
	"time"

	cccache "github.com/open-cli-collective/cli-common/cache"
)

// Status is the coarse freshness classification, re-exported from cli-common
// so downstream references (cache.StatusFresh, present.StatusRow, etc.) are
// unchanged. Classify returns only Fresh|Stale|Manual; Uninitialized is
// caller-derived from a miss and Unavailable is registry-derived (set by
// refresh via Entry.IsAvailable, not by Classify).
type Status = cccache.Status

const (
	StatusUninitialized = cccache.StatusUninitialized
	StatusFresh         = cccache.StatusFresh
	StatusStale         = cccache.StatusStale
	StatusManual        = cccache.StatusManual
	StatusUnavailable   = cccache.StatusUnavailable
)

// Classify inspects an envelope's FetchedAt + TTL at now. Semantics are
// identical to the previous local implementation (the shared version was
// lifted from it): "manual" or an unparseable TTL is handled, a zero or
// elapsed FetchedAt is stale, otherwise fresh.
func Classify(fetchedAt time.Time, ttl string, now time.Time) Status {
	return cccache.Classify(fetchedAt, ttl, now)
}

// Age returns a short human-readable age ("8h", "3d", "2m", "45s") for
// `jtk refresh --status`; "-" when fetchedAt is zero.
func Age(fetchedAt, now time.Time) string {
	return cccache.Age(fetchedAt, now)
}
