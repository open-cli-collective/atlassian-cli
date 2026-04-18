package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// ErrUnknownResource is returned by Lookup when the name is not registered.
var ErrUnknownResource = errors.New("unknown resource")

// Entry describes one cacheable resource.
//
// DependsOn names resources that must be refreshed before this one.
// Available is an optional predicate; if nil, the entry is always available.
// Fetch performs the network call(s), writes the envelope, and returns
// the number of entries written (for the refresh-line count delta).
type Entry struct {
	Name      string
	TTL       string
	DependsOn []string
	Available func(c *api.Client) bool
	Fetch     func(ctx context.Context, c *api.Client) (int, error)
}

// IsAvailable reports whether this entry is runnable against the given client.
// Entries without an Available predicate are always available.
func (e Entry) IsAvailable(c *api.Client) bool {
	if e.Available == nil {
		return true
	}
	return e.Available(c)
}

// entries is the package-level registry populated by fetchers.go at init time.
// Tests swap this slice via withTestEntries.
var entries []Entry

// Entries returns the registered entries in dependency order: entries with no
// DependsOn first, then dependents of already-placed entries. Within a group
// the original declaration order is preserved.
func Entries() []Entry {
	sorted := make([]Entry, 0, len(entries))
	seen := make(map[string]bool, len(entries))

	for _, e := range entries {
		if len(e.DependsOn) == 0 {
			sorted = append(sorted, e)
			seen[e.Name] = true
		}
	}
	for _, e := range entries {
		if seen[e.Name] {
			continue
		}
		ready := true
		for _, dep := range e.DependsOn {
			if !seen[dep] {
				ready = false
				break
			}
		}
		if ready {
			sorted = append(sorted, e)
			seen[e.Name] = true
		}
	}
	// Trailing pass for anything still unseen (transitive or missing deps).
	for _, e := range entries {
		if !seen[e.Name] {
			sorted = append(sorted, e)
		}
	}
	return sorted
}

// Lookup returns the entry with the given name, or ErrUnknownResource wrapped
// with the name.
func Lookup(name string) (Entry, error) {
	for _, e := range entries {
		if e.Name == name {
			return e, nil
		}
	}
	return Entry{}, fmt.Errorf("%w: %s", ErrUnknownResource, name)
}

// SelectWithDeps returns the entries matching the given names expanded to
// include transitive DependsOn, sorted in dependency order (deps before
// dependents). An empty name list returns every registered entry in
// dependency order. Unknown names return ErrUnknownResource.
//
// This is what `jtk refresh <names>` uses so that `refresh statuses` bootstraps
// the `projects` cache dependency automatically and argument order doesn't
// matter.
func SelectWithDeps(names []string) ([]Entry, error) {
	all := Entries()
	if len(names) == 0 {
		return all, nil
	}

	byName := make(map[string]Entry, len(entries))
	for _, e := range entries {
		byName[e.Name] = e
	}

	wanted := make(map[string]bool, len(names))
	queue := make([]string, 0, len(names))
	for _, n := range names {
		if _, ok := byName[n]; !ok {
			return nil, fmt.Errorf("%w: %s — valid names: %v", ErrUnknownResource, n, Names())
		}
		if !wanted[n] {
			wanted[n] = true
			queue = append(queue, n)
		}
	}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, dep := range byName[curr].DependsOn {
			if !wanted[dep] {
				wanted[dep] = true
				queue = append(queue, dep)
			}
		}
	}

	result := make([]Entry, 0, len(wanted))
	for _, e := range all {
		if wanted[e.Name] {
			result = append(result, e)
		}
	}
	return result, nil
}

// Names returns the registered entry names in declaration order.
func Names() []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return names
}

// projectDependents is the list of caches that must be invalidated whenever a
// project mutation occurs. Consumed by internal/cmd/projects/*.go wiring.
var projectDependents = []string{"issuetypes", "statuses"}

// ProjectDependents returns the names of caches that must be staled alongside
// `projects` when any project mutation succeeds.
func ProjectDependents() []string {
	return append([]string{"projects"}, projectDependents...)
}

// SetEntriesForTest swaps the registry for the duration of a test and returns
// a cleanup function that restores the prior value. Used by both cache-package
// tests and tests in downstream packages (e.g., internal/cmd/refresh).
func SetEntriesForTest(newEntries []Entry) func() {
	old := entries
	entries = newEntries
	return func() { entries = old }
}
