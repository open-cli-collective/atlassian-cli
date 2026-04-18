package resolve

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors used internally by two-phase lookups. They flag "cache
// wasn't loaded" vs. "cache was loaded but the input didn't match" so the
// top-level resolver can decide whether to refresh and retry.
var (
	errCacheEmpty = errors.New("resolve: cache empty")
	errNoMatch    = errors.New("resolve: no match in cache")
)

// AmbiguousCandidate is one row in an ambiguous-match failure listing.
type AmbiguousCandidate struct {
	ID          string
	DisplayName string
	Extra       string
}

// AmbiguousMatchError is returned when input matches multiple cached entities.
// Its Error() output matches the multi-line block documented in #230 so cobra
// renders it directly to stderr.
type AmbiguousMatchError struct {
	Entity  string
	Input   string
	Matches []AmbiguousCandidate
	Hint    string
}

func (e *AmbiguousMatchError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Ambiguous %s %q — %d matches:\n", e.Entity, e.Input, len(e.Matches))
	for _, m := range e.Matches {
		fmt.Fprintf(&b, "  %s | %s", m.ID, m.DisplayName)
		if m.Extra != "" {
			fmt.Fprintf(&b, " | %s", m.Extra)
		}
		b.WriteByte('\n')
	}
	if e.Hint != "" {
		b.WriteString(e.Hint)
	}
	return b.String()
}

// NotFoundError is returned when input doesn't match any cached entity and
// doesn't qualify for shape-based pass-through.
//
// Scope is a free-text qualifier appended after the quoted input for entity
// types that need parent context (e.g., issue type scoped to a project key).
// Suggestions, when populated, lists valid candidates the caller already
// knows about so the user doesn't have to run a discovery command.
type NotFoundError struct {
	Entity      string
	Input       string
	Scope       string
	RefreshHint string
	Suggestions []string
}

func (e *NotFoundError) Error() string {
	msg := fmt.Sprintf("Unknown %s %q", e.Entity, e.Input)
	if e.Scope != "" {
		msg += " " + e.Scope
	}
	msg += " — not found in cache"
	if len(e.Suggestions) > 0 {
		msg += fmt.Sprintf(". Available: %s", strings.Join(e.Suggestions, ", "))
	}
	if e.RefreshHint != "" {
		msg += fmt.Sprintf(". Try `%s` if this %s was recently added.", e.RefreshHint, e.Entity)
	}
	return msg
}

// isMissOrNoMatch reports whether err is a cache-miss/no-match signal and
// therefore a candidate for one-shot refresh-and-retry.
func isMissOrNoMatch(err error) bool {
	return errors.Is(err, errCacheEmpty) || errors.Is(err, errNoMatch)
}
