package resolve

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

// IssueType resolves an issue-type reference within a project scope.
// The projectKey should already be canonical (use Project() first).
// Matches by exact name or exact ID, case-insensitive for name.
//
// No pass-through: issue types are always looked up by name in practice,
// and silent ID pass-through would mask typos.
func (r *Resolver) IssueType(ctx context.Context, projectKey, input string) (api.IssueType, error) {
	if projectKey == "" {
		return api.IssueType{}, fmt.Errorf("resolving issue type: projectKey is required")
	}
	return resolveEntity(ctx, r, "issuetypes",
		func() (api.IssueType, error) { return lookupIssueType(projectKey, input) },
		// Issue types never pass through on warm cache — silent ID
		// fall-through would mask typos. Typo-protection only applies
		// when the cache is authoritative.
		func() (api.IssueType, bool) { return api.IssueType{}, false },
		func() error {
			return &NotFoundError{
				Entity:      "issue type",
				Input:       input,
				Scope:       fmt.Sprintf("in project %s", projectKey),
				RefreshHint: "jtk refresh issuetypes",
				Suggestions: cachedIssueTypeNamesForProject(projectKey),
			}
		},
		// Cold-start / offline: without any cache to check against, accept
		// the raw name and let the API adjudicate. This preserves pre-#236
		// behavior for uninitialized installs while staying strict on warm
		// caches.
		withColdFallback(func() (api.IssueType, bool) {
			return api.IssueType{Name: input}, true
		}),
	)
}

// cachedIssueTypeNamesForProject returns the non-subtask type names for
// projectKey from the issuetypes cache, used to enrich NotFoundError with
// "Available: ..." so users don't need a separate discovery command. Returns
// nil if the cache is missing or the project has no entries.
func cachedIssueTypeNamesForProject(projectKey string) []string {
	env, err := cache.ReadResource[map[string][]api.IssueType]("issuetypes")
	if err != nil {
		return nil
	}
	types, ok := env.Data[projectKey]
	if !ok {
		return nil
	}
	names := make([]string, 0, len(types))
	for _, t := range types {
		if !t.Subtask {
			names = append(names, t.Name)
		}
	}
	return names
}

func lookupIssueType(projectKey, input string) (api.IssueType, error) {
	env, err := cache.ReadResource[map[string][]api.IssueType]("issuetypes")
	if errors.Is(err, cache.ErrCacheMiss) || errors.Is(err, cache.ErrNoInstance) {
		return api.IssueType{}, errCacheEmpty
	}
	if err != nil {
		return api.IssueType{}, err
	}

	types, ok := env.Data[projectKey]
	if !ok {
		// The project is absent from the issuetypes cache — treat as miss so
		// the caller refreshes (which also picks up new projects via the
		// projects → issuetypes dependency chain).
		return api.IssueType{}, errCacheEmpty
	}

	// Exact ID match.
	for _, it := range types {
		if it.ID == input {
			return it, nil
		}
	}

	// Name match (case-insensitive).
	lower := strings.ToLower(input)
	var matches []api.IssueType
	for _, it := range types {
		if strings.ToLower(it.Name) == lower {
			matches = append(matches, it)
		}
	}
	switch len(matches) {
	case 0:
		return api.IssueType{}, errNoMatch
	case 1:
		return matches[0], nil
	default:
		cands := make([]AmbiguousCandidate, 0, len(matches))
		for _, it := range matches {
			cands = append(cands, AmbiguousCandidate{
				ID:          it.ID,
				DisplayName: it.Name,
			})
		}
		return api.IssueType{}, &AmbiguousMatchError{
			Entity:  "issue type",
			Input:   input,
			Matches: cands,
			Hint:    "Use the issue type ID to disambiguate.",
		}
	}
}
