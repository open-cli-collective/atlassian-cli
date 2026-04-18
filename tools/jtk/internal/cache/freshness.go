package cache

import (
	"fmt"
	"time"
)

// Status is the coarse freshness classification used by `jtk refresh --status`.
type Status int

const (
	StatusUninitialized Status = iota // no envelope on disk
	StatusFresh                       // on disk, FetchedAt + TTL still in the future
	StatusStale                       // on disk, FetchedAt + TTL elapsed
	StatusManual                      // TTL == "manual"; never reported stale
	StatusUnavailable                 // Registry Entry.Available(client) reported false
)

// String returns the status label used in `--status` output.
func (s Status) String() string {
	switch s {
	case StatusUninitialized:
		return "uninitialized"
	case StatusFresh:
		return "fresh"
	case StatusStale:
		return "stale"
	case StatusManual:
		return "manual"
	case StatusUnavailable:
		return "unavailable"
	default:
		return "unknown"
	}
}

const ttlManual = "manual"

// parseTTL returns the TTL as a time.Duration. The sentinel "manual" returns (0, true).
func parseTTL(ttl string) (time.Duration, bool, error) {
	if ttl == ttlManual {
		return 0, true, nil
	}
	d, err := time.ParseDuration(ttl)
	if err != nil {
		return 0, false, fmt.Errorf("parsing TTL %q: %w", ttl, err)
	}
	return d, false, nil
}

// Classify inspects the envelope's FetchedAt + TTL at `now` and returns a Status.
// The FetchedAt.IsZero() case is the "uninitialized" / "touched" state and returns StatusStale
// (callers that want to distinguish "never fetched" from "touched" check IsZero themselves).
func Classify(fetchedAt time.Time, ttl string, now time.Time) Status {
	d, manual, err := parseTTL(ttl)
	if err != nil {
		return StatusStale
	}
	if manual {
		return StatusManual
	}
	if fetchedAt.IsZero() || now.Sub(fetchedAt) >= d {
		return StatusStale
	}
	return StatusFresh
}

// Age returns a short human-readable age ("8h", "3d", "2m") for `--status` output.
// Returns "-" when fetchedAt is zero.
func Age(fetchedAt time.Time, now time.Time) string {
	if fetchedAt.IsZero() {
		return "-"
	}
	delta := now.Sub(fetchedAt)
	if delta < 0 {
		delta = 0
	}
	switch {
	case delta >= 24*time.Hour:
		return fmt.Sprintf("%dd", int(delta/(24*time.Hour)))
	case delta >= time.Hour:
		return fmt.Sprintf("%dh", int(delta/time.Hour))
	case delta >= time.Minute:
		return fmt.Sprintf("%dm", int(delta/time.Minute))
	default:
		return fmt.Sprintf("%ds", int(delta/time.Second))
	}
}
