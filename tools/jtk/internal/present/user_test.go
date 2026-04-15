package present

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestUserPresenter_Present(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:    "abc123",
		DisplayName:  "John Doe",
		EmailAddress: "john@example.com",
		Active:       true,
	}

	p := UserPresenter{}
	model := p.Present(user)

	if len(model.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(model.Sections))
	}

	detail, ok := model.Sections[0].(*present.DetailSection)
	if !ok {
		t.Fatalf("expected DetailSection, got %T", model.Sections[0])
	}

	findField := func(label string) string {
		for _, f := range detail.Fields {
			if f.Label == label {
				return f.Value
			}
		}
		return ""
	}

	if got := findField("Account ID"); got != "abc123" {
		t.Errorf("Account ID = %q, want abc123", got)
	}
	if got := findField("Display Name"); got != "John Doe" {
		t.Errorf("Display Name = %q, want John Doe", got)
	}
	if got := findField("Email"); got != "john@example.com" {
		t.Errorf("Email = %q, want john@example.com", got)
	}
	if got := findField("Active"); got != "yes" {
		t.Errorf("Active = %q, want yes", got)
	}
}

func TestUserPresenter_OmitsEmptyEmail(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:   "abc123",
		DisplayName: "John Doe",
		Active:      true,
	}

	p := UserPresenter{}
	model := p.Present(user)

	detail := model.Sections[0].(*present.DetailSection)
	for _, f := range detail.Fields {
		if f.Label == "Email" {
			t.Error("Email field should be omitted when empty")
		}
	}
}

func TestUserPresenter_InactiveUser(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:   "abc123",
		DisplayName: "Jane Doe",
		Active:      false,
	}

	p := UserPresenter{}
	model := p.Present(user)

	detail := model.Sections[0].(*present.DetailSection)
	findField := func(label string) string {
		for _, f := range detail.Fields {
			if f.Label == label {
				return f.Value
			}
		}
		return ""
	}

	if got := findField("Active"); got != "no" {
		t.Errorf("Active = %q, want no", got)
	}
}
