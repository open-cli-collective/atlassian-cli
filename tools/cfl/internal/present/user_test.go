package present

import (
	"testing"

	sharedpresent "github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/confluence-cli/api"
)

func TestUserPresenter_PresentUserOneLiner(t *testing.T) {
	t.Parallel()

	model := UserPresenter{}.PresentUserOneLiner(&api.User{
		AccountID:   "abc123",
		DisplayName: "Rian Stockbower",
		Email:       "rian@example.com",
	})

	msg := userMessageSection(t, model)
	testutil.Equal(t, sharedpresent.MessageInfo, msg.Kind)
	testutil.Equal(t, sharedpresent.StreamStdout, msg.Stream)
	testutil.Equal(t, "abc123 | Rian Stockbower | rian@example.com", msg.Message)
}

func TestUserPresenter_PresentUserIDOnly(t *testing.T) {
	t.Parallel()

	model := UserPresenter{}.PresentUserIDOnly(&api.User{
		AccountID: "abc123",
	})

	msg := userMessageSection(t, model)
	testutil.Equal(t, sharedpresent.MessageInfo, msg.Kind)
	testutil.Equal(t, sharedpresent.StreamStdout, msg.Stream)
	testutil.Equal(t, "abc123", msg.Message)
}

func TestUserPresenter_NormalizesEmptyFields(t *testing.T) {
	t.Parallel()

	model := UserPresenter{}.PresentUserOneLiner(&api.User{})

	msg := userMessageSection(t, model)
	testutil.Equal(t, "- | - | -", msg.Message)
}

func TestUserPresenter_NormalizesNewlinesAndPipes(t *testing.T) {
	t.Parallel()

	model := UserPresenter{}.PresentUserOneLiner(&api.User{
		AccountID:   "abc|123",
		DisplayName: "Joe | Pwn\nNext\rEnd",
		Email:       "joe\r\n@example.com",
	})

	msg := userMessageSection(t, model)
	testutil.Equal(t, `abc\|123 | Joe \| Pwn Next End | joe @example.com`, msg.Message)
}

func TestUserPresenter_NormalizesIDOnly(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		model := UserPresenter{}.PresentUserIDOnly(&api.User{})
		msg := userMessageSection(t, model)
		testutil.Equal(t, "-", msg.Message)
	})

	t.Run("pipe and newline", func(t *testing.T) {
		t.Parallel()

		model := UserPresenter{}.PresentUserIDOnly(&api.User{AccountID: "abc|def\nghi"})
		msg := userMessageSection(t, model)
		testutil.Equal(t, `abc\|def ghi`, msg.Message)
	})
}

func userMessageSection(t *testing.T, model *sharedpresent.OutputModel) *sharedpresent.MessageSection {
	t.Helper()

	if len(model.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(model.Sections))
	}
	msg, ok := model.Sections[0].(*sharedpresent.MessageSection)
	if !ok {
		t.Fatalf("expected MessageSection, got %T", model.Sections[0])
	}
	return msg
}
