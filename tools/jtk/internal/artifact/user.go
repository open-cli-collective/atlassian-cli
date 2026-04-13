package artifact

import (
	"github.com/open-cli-collective/atlassian-go/artifact"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// UserArtifact is the projected output for a user.
type UserArtifact struct {
	// Agent fields - essential for user identification
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`

	// Full-only fields
	Email  string `json:"email,omitempty"`
	Active *bool  `json:"active,omitempty"` // Pointer so inactive users show "active": false
}

// ProjectUser projects an api.User to a UserArtifact.
func ProjectUser(u *api.User, mode artifact.Type) *UserArtifact {
	a := &UserArtifact{
		AccountID:   u.AccountID,
		DisplayName: u.DisplayName,
	}
	if mode.IsFull() {
		a.Email = u.EmailAddress
		a.Active = &u.Active // Explicit true/false, never omitted in full mode
	}
	return a
}

// ProjectUsers projects a slice of api.User to UserArtifacts.
func ProjectUsers(users []api.User, mode artifact.Type) []*UserArtifact {
	result := make([]*UserArtifact, len(users))
	for i := range users {
		result[i] = ProjectUser(&users[i], mode)
	}
	return result
}
