package issues

import (
	"fmt"
	"strings"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// resolveAssignee resolves an assignee value to a Jira account ID.
// Accepts "me" (current user), an email address (searched via API), or a raw account ID.
func resolveAssignee(client *api.Client, assignee string) (string, error) {
	if strings.EqualFold(assignee, "me") {
		user, err := client.GetCurrentUser()
		if err != nil {
			return "", fmt.Errorf("failed to resolve current user: %w", err)
		}
		return user.AccountID, nil
	}

	if strings.Contains(assignee, "@") {
		users, err := client.SearchUsers(assignee, 1)
		if err != nil {
			return "", fmt.Errorf("failed to search for user %q: %w", assignee, err)
		}
		if len(users) == 0 {
			return "", fmt.Errorf("no user found matching %q", assignee)
		}
		return users[0].AccountID, nil
	}

	return assignee, nil
}
