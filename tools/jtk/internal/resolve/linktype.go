package resolve

import (
	"context"
	"errors"
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

// LinkType resolves a link type by name, inward verb, or outward verb — all
// case-insensitive. No pass-through: link type IDs are not user-facing, so
// any unrecognized input is a NotFoundError with a refresh hint.
func (r *Resolver) LinkType(ctx context.Context, input string) (api.IssueLinkType, error) {
	return resolveEntity(ctx, r, "linktypes",
		func() (api.IssueLinkType, error) { return lookupLinkType(input) },
		// Warm cache: no pass-through — typos must fail loudly.
		func() (api.IssueLinkType, bool) { return api.IssueLinkType{}, false },
		func() error {
			return &NotFoundError{
				Entity:      "link type",
				Input:       input,
				RefreshHint: "jtk refresh linktypes",
			}
		},
		// Cold-start fallback: without a cache to verify against, trust
		// the caller. The API will reject unknown link types on its own.
		// Note: directional-verb handling in links/links.go depends on
		// Inward/Outward being set, which a synthetic can't provide — on
		// cold start the user effectively must use the canonical name
		// (the CLI still functions, just without verb support).
		withColdFallback(func() (api.IssueLinkType, bool) {
			return api.IssueLinkType{Name: input}, true
		}),
	)
}

func lookupLinkType(input string) (api.IssueLinkType, error) {
	env, err := cache.ReadResource[[]api.IssueLinkType]("linktypes")
	if errors.Is(err, cache.ErrCacheMiss) {
		return api.IssueLinkType{}, errCacheEmpty
	}
	if err != nil {
		return api.IssueLinkType{}, err
	}

	lower := strings.ToLower(input)
	var matches []api.IssueLinkType
	for _, lt := range env.Data {
		if strings.ToLower(lt.Name) == lower ||
			strings.ToLower(lt.Inward) == lower ||
			strings.ToLower(lt.Outward) == lower {
			matches = append(matches, lt)
		}
	}
	switch len(matches) {
	case 0:
		return api.IssueLinkType{}, errNoMatch
	case 1:
		return matches[0], nil
	default:
		cands := make([]AmbiguousCandidate, 0, len(matches))
		for _, lt := range matches {
			cands = append(cands, AmbiguousCandidate{
				ID:          lt.ID,
				DisplayName: lt.Name,
				Extra:       lt.Outward + " / " + lt.Inward,
			})
		}
		return api.IssueLinkType{}, &AmbiguousMatchError{
			Entity:  "link type",
			Input:   input,
			Matches: cands,
			Hint:    "Use the exact link type name to disambiguate.",
		}
	}
}
