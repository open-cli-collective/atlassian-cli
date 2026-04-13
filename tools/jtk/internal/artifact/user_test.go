package artifact

import (
	"encoding/json"
	"testing"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestProjectUser_AgentMode(t *testing.T) {
	t.Parallel()

	user := &api.User{
		AccountID:    "abc123",
		DisplayName:  "John Doe",
		EmailAddress: "john@example.com",
		Active:       true,
	}

	art := ProjectUser(user, artifact.Agent)

	// Agent fields populated
	testutil.Equal(t, art.AccountID, "abc123")
	testutil.Equal(t, art.DisplayName, "John Doe")

	// Full-only fields empty/nil
	testutil.Equal(t, art.Email, "")
	testutil.Nil(t, art.Active)
}

func TestProjectUser_FullMode_ActiveUser(t *testing.T) {
	t.Parallel()

	user := &api.User{
		AccountID:    "abc123",
		DisplayName:  "John Doe",
		EmailAddress: "john@example.com",
		Active:       true,
	}

	art := ProjectUser(user, artifact.Full)

	// Agent fields populated
	testutil.Equal(t, art.AccountID, "abc123")
	testutil.Equal(t, art.DisplayName, "John Doe")

	// Full-only fields populated
	testutil.Equal(t, art.Email, "john@example.com")
	testutil.NotNil(t, art.Active)
	testutil.True(t, *art.Active)
}

func TestProjectUser_FullMode_InactiveUser(t *testing.T) {
	t.Parallel()

	user := &api.User{
		AccountID:    "abc123",
		DisplayName:  "Jane Doe",
		EmailAddress: "jane@example.com",
		Active:       false,
	}

	art := ProjectUser(user, artifact.Full)

	// Full-only Active should be explicitly false, not omitted
	testutil.NotNil(t, art.Active)
	testutil.False(t, *art.Active)
}

func TestProjectUser_JSONSerialization(t *testing.T) {
	t.Parallel()

	t.Run("agent mode omits active", func(t *testing.T) {
		t.Parallel()
		user := &api.User{AccountID: "abc", DisplayName: "Test", Active: true}
		art := ProjectUser(user, artifact.Agent)

		data, err := json.Marshal(art)
		testutil.RequireNoError(t, err)

		var parsed map[string]any
		err = json.Unmarshal(data, &parsed)
		testutil.RequireNoError(t, err)

		_, exists := parsed["active"]
		testutil.False(t, exists) // Should not be present in agent mode
	})

	t.Run("full mode includes active false", func(t *testing.T) {
		t.Parallel()
		user := &api.User{AccountID: "abc", DisplayName: "Test", Active: false}
		art := ProjectUser(user, artifact.Full)

		data, err := json.Marshal(art)
		testutil.RequireNoError(t, err)

		var parsed map[string]any
		err = json.Unmarshal(data, &parsed)
		testutil.RequireNoError(t, err)

		active, exists := parsed["active"]
		testutil.True(t, exists)                // Should be present
		testutil.Equal(t, active.(bool), false) // Should be false, not omitted
	})
}

func TestProjectUsers(t *testing.T) {
	t.Parallel()

	users := []api.User{
		{AccountID: "1", DisplayName: "User 1"},
		{AccountID: "2", DisplayName: "User 2"},
	}

	arts := ProjectUsers(users, artifact.Agent)

	testutil.Equal(t, len(arts), 2)
	testutil.Equal(t, arts[0].AccountID, "1")
	testutil.Equal(t, arts[1].AccountID, "2")
}

func TestProjectUsers_Empty(t *testing.T) {
	t.Parallel()

	var users []api.User
	arts := ProjectUsers(users, artifact.Agent)

	testutil.Equal(t, len(arts), 0)
	testutil.NotNil(t, arts)
}
