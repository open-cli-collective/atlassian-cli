package resolve

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

// Board resolves a board reference (numeric ID or name) to an api.Board.
// Numeric inputs pass through as synthetic api.Board{ID: n} if not cached.
func (r *Resolver) Board(ctx context.Context, input string) (api.Board, error) {
	return resolveEntity(ctx, r, "boards",
		func() (api.Board, error) { return lookupBoard(input) },
		func() (api.Board, bool) {
			if looksLikeNumeric(input) {
				n, _ := strconv.Atoi(input)
				return api.Board{ID: n}, true
			}
			return api.Board{}, false
		},
		func() error {
			return &NotFoundError{
				Entity:      "board",
				Input:       input,
				RefreshHint: "jtk refresh boards",
			}
		},
	)
}

func lookupBoard(input string) (api.Board, error) {
	env, err := cache.ReadResource[[]api.Board]("boards")
	if errors.Is(err, cache.ErrCacheMiss) {
		return api.Board{}, errCacheEmpty
	}
	if err != nil {
		return api.Board{}, err
	}

	// Numeric ID match against cache.
	if looksLikeNumeric(input) {
		n, _ := strconv.Atoi(input)
		for _, b := range env.Data {
			if b.ID == n {
				return b, nil
			}
		}
	}

	// Name match (case-insensitive).
	lower := strings.ToLower(input)
	var matches []api.Board
	for _, b := range env.Data {
		if strings.ToLower(b.Name) == lower {
			matches = append(matches, b)
		}
	}
	switch len(matches) {
	case 0:
		return api.Board{}, errNoMatch
	case 1:
		return matches[0], nil
	default:
		cands := make([]AmbiguousCandidate, 0, len(matches))
		for _, b := range matches {
			cands = append(cands, AmbiguousCandidate{
				ID:          strconv.Itoa(b.ID),
				DisplayName: b.Name,
				Extra:       b.Type,
			})
		}
		return api.Board{}, &AmbiguousMatchError{
			Entity:  "board",
			Input:   input,
			Matches: cands,
			Hint:    "Use board ID to disambiguate.",
		}
	}
}
