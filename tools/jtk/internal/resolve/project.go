package resolve

import (
	"context"
	"errors"
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

// Project resolves a project reference (by key, name, or literal
// project-key-shape token) to an api.Project with .Key canonicalized.
//
// Numeric project IDs are intentionally NOT supported as pass-through:
// downstream consumers like the issues move payload require PROJECT_KEY,
// which rules out numeric-ID pass-through without a post-resolve lookup.
// If you need that, add it explicitly.
func (r *Resolver) Project(ctx context.Context, input string) (api.Project, error) {
	return resolveEntity(ctx, r, "projects",
		func() (api.Project, error) { return lookupProject(input) },
		func() (api.Project, bool) {
			if looksLikeProjectKey(input) {
				return api.Project{Key: input}, true
			}
			return api.Project{}, false
		},
		func() error {
			return &NotFoundError{
				Entity:      "project",
				Input:       input,
				RefreshHint: "jtk refresh projects",
			}
		},
	)
}

func lookupProject(input string) (api.Project, error) {
	env, err := cache.ReadResource[[]api.Project]("projects")
	if errors.Is(err, cache.ErrCacheMiss) {
		return api.Project{}, errCacheEmpty
	}
	if err != nil {
		return api.Project{}, err
	}

	// Exact key match (case-sensitive — keys are uppercase by convention).
	for _, p := range env.Data {
		if p.Key == input {
			return p, nil
		}
	}

	// Name match (case-insensitive).
	lower := strings.ToLower(input)
	var matches []api.Project
	for _, p := range env.Data {
		if strings.ToLower(p.Name) == lower {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 0:
		return api.Project{}, errNoMatch
	case 1:
		return matches[0], nil
	default:
		cands := make([]AmbiguousCandidate, 0, len(matches))
		for _, p := range matches {
			cands = append(cands, AmbiguousCandidate{ID: p.Key, DisplayName: p.Name})
		}
		return api.Project{}, &AmbiguousMatchError{
			Entity:  "project",
			Input:   input,
			Matches: cands,
			Hint:    "Use project key to disambiguate.",
		}
	}
}
