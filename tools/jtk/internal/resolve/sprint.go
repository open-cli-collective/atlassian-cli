package resolve

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

// Sprint resolves a sprint reference (numeric ID or name) to an api.Sprint.
//
// If boardID is non-zero, name matching is scoped to that board first. If
// boardID is 0, the resolver searches every cached board and requires a
// unique match globally. Ambiguity errors include board ID/name and state
// so the caller can disambiguate with --board or a board-scoped CLI path.
//
// Numeric inputs pass through as synthetic api.Sprint{ID: n} when not cached.
func (r *Resolver) Sprint(ctx context.Context, input string, boardID int) (api.Sprint, error) {
	return resolveEntity(ctx, r, "sprints",
		func() (api.Sprint, error) { return lookupSprint(input, boardID) },
		func() (api.Sprint, bool) {
			if looksLikeNumeric(input) {
				n, _ := strconv.Atoi(input)
				return api.Sprint{ID: n}, true
			}
			return api.Sprint{}, false
		},
		func() error {
			return &NotFoundError{
				Entity:      "sprint",
				Input:       input,
				RefreshHint: "jtk refresh sprints",
			}
		},
	)
}

func lookupSprint(input string, boardID int) (api.Sprint, error) {
	env, err := cache.ReadResource[map[int][]api.Sprint]("sprints")
	if errors.Is(err, cache.ErrCacheMiss) || errors.Is(err, cache.ErrNoInstance) {
		return api.Sprint{}, errCacheEmpty
	}
	if err != nil {
		return api.Sprint{}, err
	}

	// Numeric ID match — global across boards.
	if looksLikeNumeric(input) {
		n, _ := strconv.Atoi(input)
		for _, sprints := range env.Data {
			for _, s := range sprints {
				if s.ID == n {
					return s, nil
				}
			}
		}
	}

	// Name match. If boardID != 0, restrict to that board; otherwise search
	// across every cached board.
	lower := strings.ToLower(input)

	if boardID != 0 {
		matches := nameMatchesOnBoard(env.Data[boardID], lower)
		switch len(matches) {
		case 0:
			return api.Sprint{}, errNoMatch
		case 1:
			return matches[0], nil
		default:
			return api.Sprint{}, ambiguousSprints(input, matches, boardID)
		}
	}

	type boardHit struct {
		sprint api.Sprint
		board  int
	}
	var hits []boardHit
	for bID, sprints := range env.Data {
		for _, s := range sprints {
			if strings.ToLower(s.Name) == lower {
				hits = append(hits, boardHit{sprint: s, board: bID})
			}
		}
	}
	switch len(hits) {
	case 0:
		return api.Sprint{}, errNoMatch
	case 1:
		return hits[0].sprint, nil
	default:
		cands := make([]AmbiguousCandidate, 0, len(hits))
		boardNames := boardNameLookup()
		for _, h := range hits {
			cands = append(cands, AmbiguousCandidate{
				ID:          strconv.Itoa(h.sprint.ID),
				DisplayName: h.sprint.Name,
				Extra:       sprintExtra(h.sprint, h.board, boardNames),
			})
		}
		return api.Sprint{}, &AmbiguousMatchError{
			Entity:  "sprint",
			Input:   input,
			Matches: cands,
			Hint:    "Use sprint ID or scope to a board to disambiguate.",
		}
	}
}

func nameMatchesOnBoard(sprints []api.Sprint, lowerInput string) []api.Sprint {
	var out []api.Sprint
	for _, s := range sprints {
		if strings.ToLower(s.Name) == lowerInput {
			out = append(out, s)
		}
	}
	return out
}

func ambiguousSprints(input string, matches []api.Sprint, boardID int) *AmbiguousMatchError {
	cands := make([]AmbiguousCandidate, 0, len(matches))
	boardNames := boardNameLookup()
	for _, s := range matches {
		cands = append(cands, AmbiguousCandidate{
			ID:          strconv.Itoa(s.ID),
			DisplayName: s.Name,
			Extra:       sprintExtra(s, boardID, boardNames),
		})
	}
	return &AmbiguousMatchError{
		Entity:  "sprint",
		Input:   input,
		Matches: cands,
		Hint:    "Use sprint ID to disambiguate.",
	}
}

// boardNameLookup returns a map from board ID → "ID (Name)" for use in
// ambiguous-match error output. Returns nil if the boards cache is missing
// or unreadable — callers fall back to bare board IDs.
func boardNameLookup() map[int]string {
	env, err := cache.ReadResource[[]api.Board]("boards")
	if err != nil {
		return nil
	}
	out := make(map[int]string, len(env.Data))
	for _, b := range env.Data {
		out[b.ID] = b.Name
	}
	return out
}

func sprintExtra(s api.Sprint, boardID int, boardNames map[int]string) string {
	board := fmt.Sprintf("board %d", boardID)
	if name, ok := boardNames[boardID]; ok && name != "" {
		board = fmt.Sprintf("board %d (%s)", boardID, name)
	}
	if s.State != "" {
		return fmt.Sprintf("%s, %s", board, s.State)
	}
	return board
}
