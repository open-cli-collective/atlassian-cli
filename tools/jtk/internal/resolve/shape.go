package resolve

import "regexp"

// Shape heuristics applied *after* cache lookup fails, to decide whether an
// unrecognized input should pass through as a raw identifier or fail outright.
// Keep these intentionally narrow — over-broad regexes would let typos sneak
// through to the API as literal IDs.
var (
	// Standard Jira project key: leading letter, total 2–10 uppercase
	// alphanumeric/underscore. Matches "MON", "CAPONE", "A1_B2", etc.
	projectKeyRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,9}$`)

	// Pure-numeric tokens (board/sprint IDs).
	numericRe = regexp.MustCompile(`^\d+$`)

	// Atlassian account IDs. Legacy IDs are 24-char hex; federated IDs look
	// like "557058:295fe89c-10c2-4b0c-ba84-a4dd14ea7729". We require ≥16 chars
	// and restrict to identifier-safe characters so short display names like
	// "Rusty Hall" don't accidentally pass through.
	accountIDRe = regexp.MustCompile(`^[a-zA-Z0-9:.\-_]{16,}$`)
)

func looksLikeProjectKey(s string) bool { return projectKeyRe.MatchString(s) }
func looksLikeNumeric(s string) bool    { return numericRe.MatchString(s) }
func looksLikeAccountID(s string) bool  { return accountIDRe.MatchString(s) }
