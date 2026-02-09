package issues

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// resolveAssignee resolves an assignee value to a Jira account ID.
// It accepts:
//   - "me" — resolves to the authenticated user's account ID
//   - an email address (contains @) — searches users and matches by email
//   - a raw account ID — returned as-is
func resolveAssignee(client *api.Client, value string) (string, error) {
	if strings.EqualFold(value, "me") {
		user, err := client.GetCurrentUser()
		if err != nil {
			return "", fmt.Errorf("failed to resolve 'me': %w", err)
		}
		return user.AccountID, nil
	}

	if strings.Contains(value, "@") {
		users, err := client.SearchUsers(value, 50)
		if err != nil {
			return "", fmt.Errorf("failed to search users: %w", err)
		}

		// Look for exact email match
		for _, u := range users {
			if strings.EqualFold(u.EmailAddress, value) {
				return u.AccountID, nil
			}
		}

		return "", fmt.Errorf("no user found with email %q", value)
	}

	return value, nil
}
