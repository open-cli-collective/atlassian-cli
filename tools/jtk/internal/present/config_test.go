package present

import (
	"errors"
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestConfigPresenter_PresentTestResult_NoURL(t *testing.T) {
	t.Parallel()

	p := ConfigPresenter{}
	model := p.PresentTestResult("", nil, nil, nil)

	testutil.Equal(t, len(model.Sections), 3)

	// First section should be error
	msg := model.Sections[0].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageError)
	testutil.Contains(t, msg.Message, "No Jira URL configured")
	testutil.Equal(t, msg.Stream, present.StreamStderr)

	// Second section should be info with init suggestion
	msg = model.Sections[1].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageInfo)
	testutil.Contains(t, msg.Message, "jtk init")

	// Third section should be info with env var suggestion
	msg = model.Sections[2].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageInfo)
	testutil.Contains(t, msg.Message, "JIRA_URL")
}

func TestConfigPresenter_PresentTestResult_ClientError(t *testing.T) {
	t.Parallel()

	p := ConfigPresenter{}
	clientErr := errors.New("invalid credentials format")
	model := p.PresentTestResult("https://test.atlassian.net", nil, clientErr, nil)

	testutil.Equal(t, len(model.Sections), 4)

	// First section: testing connection message
	msg := model.Sections[0].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageInfo)
	testutil.Contains(t, msg.Message, "Testing connection to https://test.atlassian.net")

	// Second section: error message
	msg = model.Sections[1].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageError)
	testutil.Contains(t, msg.Message, "Failed to create client")
	testutil.Contains(t, msg.Message, "invalid credentials format")
	testutil.Equal(t, msg.Stream, present.StreamStderr)

	// Third and fourth sections: help suggestions
	msg = model.Sections[2].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageInfo)
	testutil.Contains(t, msg.Message, "jtk config show")

	msg = model.Sections[3].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageInfo)
	testutil.Contains(t, msg.Message, "jtk init")
}

func TestConfigPresenter_PresentTestResult_AuthError(t *testing.T) {
	t.Parallel()

	p := ConfigPresenter{}
	authErr := errors.New("401 Unauthorized")
	model := p.PresentTestResult("https://test.atlassian.net", nil, nil, authErr)

	testutil.Equal(t, len(model.Sections), 4)

	// First section: testing connection message
	msg := model.Sections[0].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageInfo)
	testutil.Contains(t, msg.Message, "Testing connection")

	// Second section: auth error
	msg = model.Sections[1].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageError)
	testutil.Contains(t, msg.Message, "Authentication failed")
	testutil.Contains(t, msg.Message, "401 Unauthorized")
	testutil.Equal(t, msg.Stream, present.StreamStderr)

	// Third and fourth: help suggestions
	msg = model.Sections[2].(*present.MessageSection)
	testutil.Contains(t, msg.Message, "jtk config show")

	msg = model.Sections[3].(*present.MessageSection)
	testutil.Contains(t, msg.Message, "jtk init")
}

func TestConfigPresenter_PresentTestResult_Success(t *testing.T) {
	t.Parallel()

	p := ConfigPresenter{}
	user := &api.User{
		AccountID:    "abc-123",
		DisplayName:  "Test User",
		EmailAddress: "test@example.com",
	}
	model := p.PresentTestResult("https://test.atlassian.net", user, nil, nil)

	testutil.Equal(t, len(model.Sections), 5)

	// First: testing connection
	msg := model.Sections[0].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageInfo)

	// Second: auth successful
	msg = model.Sections[1].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageSuccess)
	testutil.Contains(t, msg.Message, "Authentication successful")

	// Third: API access verified
	msg = model.Sections[2].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageSuccess)
	testutil.Contains(t, msg.Message, "API access verified")

	// Fourth: user display name
	msg = model.Sections[3].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageInfo)
	testutil.Contains(t, msg.Message, "Test User")
	testutil.Contains(t, msg.Message, "test@example.com")

	// Fifth: account ID
	msg = model.Sections[4].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageInfo)
	testutil.Contains(t, msg.Message, "abc-123")
}

func TestConfigPresenter_PresentTestResult_SuccessNoUser(t *testing.T) {
	t.Parallel()

	p := ConfigPresenter{}
	// Success but user is nil (edge case)
	model := p.PresentTestResult("https://test.atlassian.net", nil, nil, nil)

	// Should have: testing message + 2 success messages (no user info)
	testutil.Equal(t, len(model.Sections), 3)

	msg := model.Sections[1].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageSuccess)
	testutil.Contains(t, msg.Message, "Authentication successful")

	msg = model.Sections[2].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageSuccess)
	testutil.Contains(t, msg.Message, "API access verified")
}

func TestConfigPresenter_PresentConfigShow(t *testing.T) {
	t.Parallel()

	p := ConfigPresenter{}
	model := p.PresentConfigShow(
		"https://test.atlassian.net", "env (JIRA_URL)",
		"test@example.com", "config",
		"****token****", "env (JIRA_API_TOKEN)",
		"PROJ", "config",
		"basic", "default",
		"", "-",
		"/path/to/config.json",
	)

	testutil.Equal(t, len(model.Sections), 2)

	// First section: table
	table := model.Sections[0].(*present.TableSection)
	testutil.Equal(t, table.Headers, []string{"KEY", "VALUE", "SOURCE"})
	testutil.Equal(t, len(table.Rows), 6)

	// Verify first row (url)
	testutil.Equal(t, table.Rows[0].Cells[0], "url")
	testutil.Equal(t, table.Rows[0].Cells[1], "https://test.atlassian.net")
	testutil.Equal(t, table.Rows[0].Cells[2], "env (JIRA_URL)")

	// Second section: config path message
	msg := model.Sections[1].(*present.MessageSection)
	testutil.Equal(t, msg.Kind, present.MessageInfo)
	testutil.Contains(t, msg.Message, "/path/to/config.json")
}
