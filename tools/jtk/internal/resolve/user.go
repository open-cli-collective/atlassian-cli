package resolve

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

// User resolves an assignee/lead/user reference to an api.User.
//
// Lookup order:
//  1. "me" → live GetCurrentUser (inherent — the cache can't know who *you* are).
//  2. Cached accountId exact match.
//  3. For inputs containing "@": cached email exact match (case-insensitive).
//  4. Cached display-name exact match (case-insensitive).
//  5. On miss/no-match: refresh `users` once and retry 2–4.
//  6. If input matches the accountId shape: synthetic pass-through.
//  7. Otherwise: NotFoundError with a refresh hint.
//
// Multiple display-name matches produce an AmbiguousMatchError with all
// candidates listed.
func (r *Resolver) User(ctx context.Context, input string) (api.User, error) {
	if strings.EqualFold(input, "me") {
		u, err := r.client.GetCurrentUser(ctx, "")
		if err != nil {
			return api.User{}, fmt.Errorf("resolving current user: %w", err)
		}
		return *u, nil
	}

	u, err := resolveEntity(ctx, r, "users",
		func() (api.User, error) { return lookupUser(input) },
		func() (api.User, bool) {
			if looksLikeAccountID(input) {
				return api.User{AccountID: input}, true
			}
			return api.User{}, false
		},
		func() error {
			return &NotFoundError{
				Entity:      "user",
				Input:       input,
				RefreshHint: "jtk refresh users",
			}
		},
	)
	if err == nil {
		return u, nil
	}
	// Last-resort live lookup for email-shaped input. The users cache is
	// truncated on large instances (~1000-user ceiling in Jira's paginated
	// enumeration), and newly-added users won't appear until the next
	// refresh. Email is a unique identifier by Jira's contract, so there's
	// no ambiguity risk. Only fires on NotFoundError *after* the one-shot
	// refresh already ran — not a fast-path bypass of the cache.
	if strings.Contains(input, "@") {
		var nf *NotFoundError
		if errors.As(err, &nf) {
			if live, lerr := r.client.SearchUsers(ctx, input, 0, 1); lerr == nil && len(live) == 1 {
				return live[0], nil
			}
		}
	}
	return api.User{}, err
}

func lookupUser(input string) (api.User, error) {
	env, err := cache.ReadResource[[]api.User]("users")
	if errors.Is(err, cache.ErrCacheMiss) || errors.Is(err, cache.ErrNoInstance) {
		return api.User{}, errCacheEmpty
	}
	if err != nil {
		return api.User{}, err
	}

	// Exact accountId match — unique by Jira's contract.
	for _, u := range env.Data {
		if u.AccountID == input {
			return u, nil
		}
	}

	// Email match (inputs containing "@" are treated as emails, not names).
	if strings.Contains(input, "@") {
		var matches []api.User
		for _, u := range env.Data {
			if strings.EqualFold(u.EmailAddress, input) {
				matches = append(matches, u)
			}
		}
		switch len(matches) {
		case 0:
			return api.User{}, errNoMatch
		case 1:
			return matches[0], nil
		default:
			return api.User{}, ambiguousUsers(input, matches)
		}
	}

	// Display-name match (case-insensitive).
	lower := strings.ToLower(input)
	var matches []api.User
	for _, u := range env.Data {
		if strings.ToLower(u.DisplayName) == lower {
			matches = append(matches, u)
		}
	}
	switch len(matches) {
	case 0:
		return api.User{}, errNoMatch
	case 1:
		return matches[0], nil
	default:
		return api.User{}, ambiguousUsers(input, matches)
	}
}

func ambiguousUsers(input string, matches []api.User) *AmbiguousMatchError {
	cands := make([]AmbiguousCandidate, 0, len(matches))
	for _, u := range matches {
		cands = append(cands, AmbiguousCandidate{
			ID:          u.AccountID,
			DisplayName: u.DisplayName,
			Extra:       u.EmailAddress,
		})
	}
	return &AmbiguousMatchError{
		Entity:  "user",
		Input:   input,
		Matches: cands,
		Hint:    "Use account ID or email to disambiguate.",
	}
}
