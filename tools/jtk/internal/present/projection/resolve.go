package projection

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// UnknownFieldError reports one or more --fields tokens that matched
// nothing: not a registry header/alias, not a registry FieldID, and not
// a live api.Field.Name. Suggestions lists the valid registry headers
// for the active mode so callers can nudge the user toward the right token.
type UnknownFieldError struct {
	Unknown     []string
	Suggestions []string
}

func (e *UnknownFieldError) Error() string {
	if len(e.Unknown) == 1 {
		return fmt.Sprintf("unknown field %q; supported fields: %s",
			e.Unknown[0], strings.Join(e.Suggestions, ", "))
	}
	return fmt.Sprintf("unknown fields %s; supported fields: %s",
		quoteAll(e.Unknown), strings.Join(e.Suggestions, ", "))
}

// AmbiguousFieldNameError reports a --fields token that matches multiple
// Jira fields by human name. The user must disambiguate with an exact
// field ID.
type AmbiguousFieldNameError struct {
	Token   string
	Matches []api.Field
}

func (e *AmbiguousFieldNameError) Error() string {
	parts := make([]string, len(e.Matches))
	for i, f := range e.Matches {
		parts[i] = fmt.Sprintf("%s (%s)", f.Name, f.ID)
	}
	return fmt.Sprintf(
		"field name %q is ambiguous — matches: %s; use the field ID to disambiguate",
		e.Token, strings.Join(parts, ", "),
	)
}

// ExtendedOnlyError reports a --fields token that matches an Extended-only
// spec while --extended is off. The user can fix the command by adding
// --extended.
type ExtendedOnlyError struct {
	Token  string
	Header string
}

func (e *ExtendedOnlyError) Error() string {
	return fmt.Sprintf(
		"field %q is only available with --extended (matches %q)",
		e.Token, e.Header,
	)
}

// Resolve is the single entrypoint commands call.
//
// projectionApplied is the authoritative switch; callers MUST branch on it,
// not on len(selected). selected carries two meanings depending on the flag:
//   - projectionApplied == false → selected is the full mode registry
//     (r.ForMode(extended)). Callers render the full model; do NOT call
//     ProjectTable/ProjectDetail.
//   - projectionApplied == true  → selected is the user's chosen subset
//     (identity-prepended, user order preserved). Callers MUST call
//     ProjectTable/ProjectDetail to slice the model.
//
// Behavior:
//   - fieldsFlag empty → (r.ForMode(extended), false, nil). fetchFields is
//     NOT called.
//   - fieldsFlag non-empty: parse CSV; resolve each token. Tokens that miss
//     header/alias/FieldID matching trigger a single fetchFields() call
//     (memoized across tokens in the invocation) and retry against
//     api.Field.Name (case-insensitive).
//   - Identity specs are prepended if the user omitted them; dedup preserved.
//   - If a token matches an Extended-only spec with extended==false, return
//     ExtendedOnlyError.
//   - If a token resolves to a real api.Field but no registry entry, a
//     dynamic ColumnSpec is created (Dynamic: true) so the command can
//     render it from the issue's raw field data.
//   - If a human name matches multiple Jira fields, return
//     AmbiguousFieldNameError so the user can disambiguate with a field ID.
//   - Otherwise, unresolved tokens → UnknownFieldError with registry-based
//     suggestions.
func Resolve(
	ctx context.Context,
	r Registry,
	extended bool,
	fieldsFlag string,
	fetchFields func(context.Context) ([]api.Field, error),
	_ string,
) (selected []ColumnSpec, projectionApplied bool, err error) {
	modeRegistry := r.ForMode(extended)

	tokens := parseTokens(fieldsFlag)
	if len(tokens) == 0 {
		return modeRegistry, false, nil
	}

	var cachedFields []api.Field
	var fieldsFetched bool

	lookupFields := func() ([]api.Field, error) {
		if fieldsFetched {
			return cachedFields, nil
		}
		fieldsFetched = true
		fields, err := fetchFields(ctx)
		if err != nil {
			return nil, err
		}
		cachedFields = fields
		return fields, nil
	}

	seen := make(map[string]struct{})
	out := make([]ColumnSpec, 0, len(tokens)+1)

	appendSpec := func(c ColumnSpec) {
		if _, ok := seen[c.Header]; ok {
			return
		}
		seen[c.Header] = struct{}{}
		out = append(out, c)
	}

	// Identity first — always included, silently prepended if the user
	// omitted it.
	for _, c := range modeRegistry {
		if c.Identity {
			appendSpec(c)
		}
	}

	// Two-pass resolution that preserves user token order.
	//
	// Pass 1 (fast path): try header/alias/FieldID matching for every token
	// without consulting Jira metadata. Tokens that miss are queued for the
	// slow path. Resolved specs are stored by their token index so they land
	// in the user's order even when interleaved with slow-path tokens.
	//
	// Pass 2 (slow path): if any tokens deferred, fetchFields() once, then
	// retry each deferred token against the mode registry (picks up human
	// names), the full registry (for Extended-only errors), and raw Jira
	// metadata (for UnrenderedFieldError).
	resolved := make([]*ColumnSpec, len(tokens))
	var deferred []int
	for i, tok := range tokens {
		if spec, ok := modeRegistry.Match(tok, nil); ok {
			s := spec
			resolved[i] = &s
			continue
		}
		deferred = append(deferred, i)
	}

	if len(deferred) > 0 {
		fields, ferr := lookupFields()
		if ferr != nil {
			return nil, false, ferr
		}

		// Collect all known headers to detect collisions when creating
		// dynamic specs. Seed with the full registry (not just selected
		// or mode-filtered) so dynamic fields named like built-ins always
		// get disambiguated.
		resolvedHeaders := make(map[string]struct{})
		for _, c := range r {
			resolvedHeaders[strings.ToLower(c.Header)] = struct{}{}
		}

		var unknown []string
		for _, i := range deferred {
			tok := tokens[i]
			if spec, ok := modeRegistry.Match(tok, fields); ok {
				s := spec
				resolved[i] = &s
				resolvedHeaders[strings.ToLower(s.Header)] = struct{}{}
				continue
			}

			if !extended {
				if spec, ok := r.Match(tok, fields); ok && spec.Extended {
					return nil, false, &ExtendedOnlyError{Token: tok, Header: spec.Header}
				}
			}

			if jf := findJiraField(fields, tok); jf != nil {
				// Check for ambiguous human-name matches (only when
				// the token is NOT an exact field ID match).
				if api.FindFieldByID(fields, tok) == nil {
					if matches := findAllJiraFieldsByName(fields, tok); len(matches) > 1 {
						return nil, false, &AmbiguousFieldNameError{Token: tok, Matches: matches}
					}
				}

				header := jf.Name
				if _, collision := resolvedHeaders[strings.ToLower(header)]; collision {
					header = fmt.Sprintf("%s (%s)", jf.Name, jf.ID)
				}
				resolvedHeaders[strings.ToLower(header)] = struct{}{}

				resolved[i] = &ColumnSpec{
					Header:  header,
					FieldID: jf.ID,
					Fetch:   []string{jf.ID},
					Dynamic: true,
				}
				continue
			}

			unknown = append(unknown, tok)
		}

		if len(unknown) > 0 {
			return nil, false, &UnknownFieldError{
				Unknown:     unknown,
				Suggestions: registryHeaders(modeRegistry),
			}
		}
	}

	for _, spec := range resolved {
		if spec != nil {
			appendSpec(*spec)
		}
	}

	return out, true, nil
}

// findJiraField looks up a token against the live Jira field metadata by
// ID (exact) or Name (case-insensitive). Returns nil when neither matches.
func findJiraField(fields []api.Field, token string) *api.Field {
	if f := api.FindFieldByID(fields, token); f != nil {
		return f
	}
	if f := api.FindFieldByName(fields, token); f != nil {
		return f
	}
	return nil
}

// findAllJiraFieldsByName returns all fields whose Name matches token
// (case-insensitive). Used to detect ambiguous human-name references.
func findAllJiraFieldsByName(fields []api.Field, token string) []api.Field {
	lower := strings.ToLower(token)
	var matches []api.Field
	for _, f := range fields {
		if strings.ToLower(f.Name) == lower {
			matches = append(matches, f)
		}
	}
	return matches
}

// parseTokens splits a --fields CSV, trims whitespace, drops empty segments.
func parseTokens(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// registryHeaders returns the headers of r in stable order, suitable for
// error suggestion text.
func registryHeaders(r Registry) []string {
	out := make([]string, 0, len(r))
	for _, c := range r {
		out = append(out, c.Header)
	}
	sort.Strings(out)
	return out
}

func quoteAll(ss []string) string {
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(parts, ", ")
}
