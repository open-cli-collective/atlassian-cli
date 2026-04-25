package present

import (
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

func TestPresentUserOneLiner(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:    "abc123",
		DisplayName:  "John Doe",
		EmailAddress: "john@example.com",
		Active:       true,
	}

	model := UserPresenter{}.PresentUserOneLiner(user)
	got := renderMessage(t, model, 0)
	want := "abc123 | John Doe | john@example.com"
	if got != want {
		t.Errorf("one-liner = %q, want %q", got, want)
	}
}

func TestPresentUserOneLiner_DashForEmptyEmail(t *testing.T) {
	t.Parallel()
	user := &api.User{AccountID: "abc123", DisplayName: "Jane"}
	model := UserPresenter{}.PresentUserOneLiner(user)
	got := renderMessage(t, model, 0)
	want := "abc123 | Jane | -"
	if got != want {
		t.Errorf("one-liner with empty email = %q, want %q", got, want)
	}
}

func TestPresentUserExtended_AllFieldsPresent(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:        "abc",
		DisplayName:      "Alice",
		EmailAddress:     "alice@example.com",
		Active:           true,
		TimeZone:         "Etc/GMT",
		Locale:           "en_US",
		Groups:           &api.UserCountBlock{Size: 9},
		ApplicationRoles: &api.UserCountBlock{Size: 1},
	}

	model := UserPresenter{}.PresentUserExtended(user)
	if len(model.Sections) != 3 {
		t.Fatalf("extended model: expected 3 sections, got %d", len(model.Sections))
	}

	expect := []string{
		"abc | Alice | alice@example.com",
		"Timezone: Etc/GMT   Locale: en_US   Active: yes",
		"Groups: 9   Application Roles: 1",
	}
	for i, want := range expect {
		if got := renderMessage(t, model, i); got != want {
			t.Errorf("section %d = %q, want %q", i, got, want)
		}
	}
}

func TestPresentUserExtended_DashesForMissingOptionalFields(t *testing.T) {
	t.Parallel()
	user := &api.User{
		AccountID:   "abc",
		DisplayName: "Bob",
		Active:      false,
		// TimeZone / Locale empty; Groups / ApplicationRoles nil (API omitted expansion)
	}

	model := UserPresenter{}.PresentUserExtended(user)

	expect := []string{
		"abc | Bob | -",
		"Timezone: -   Locale: -   Active: no",
		"Groups: -   Application Roles: -",
	}
	for i, want := range expect {
		if got := renderMessage(t, model, i); got != want {
			t.Errorf("section %d = %q, want %q", i, got, want)
		}
	}
}

func TestPresentUserList_DefaultHeaders(t *testing.T) {
	t.Parallel()
	users := []api.User{
		{AccountID: "a1", DisplayName: "Alice", EmailAddress: "a@example.com", Active: true},
		{AccountID: "b2", DisplayName: "Bob", Active: false},
	}
	model := UserPresenter{}.PresentUserList(users, false)
	table := sectionTable(t, model, 0)

	wantHeaders := []string{"ACCOUNT_ID", "NAME", "EMAIL", "ACTIVE"}
	if !equalStringSlices(table.Headers, wantHeaders) {
		t.Errorf("headers = %v, want %v", table.Headers, wantHeaders)
	}

	if len(table.Rows) != 2 {
		t.Fatalf("rows: got %d, want 2", len(table.Rows))
	}
	if got := table.Rows[0].Cells; !equalStringSlices(got, []string{"a1", "Alice", "a@example.com", "yes"}) {
		t.Errorf("row 0 = %v", got)
	}
	if got := table.Rows[1].Cells; !equalStringSlices(got, []string{"b2", "Bob", "-", "no"}) {
		t.Errorf("row 1 = %v", got)
	}
}

func TestPresentUserList_ExtendedHeadersAppendTimezoneLocale(t *testing.T) {
	t.Parallel()
	users := []api.User{{AccountID: "a1", DisplayName: "Alice", TimeZone: "Etc/GMT", Locale: "en_US", Active: true}}
	model := UserPresenter{}.PresentUserList(users, true)
	table := sectionTable(t, model, 0)

	wantHeaders := []string{"ACCOUNT_ID", "NAME", "EMAIL", "ACTIVE", "TIMEZONE", "LOCALE"}
	if !equalStringSlices(table.Headers, wantHeaders) {
		t.Errorf("extended headers = %v, want %v", table.Headers, wantHeaders)
	}

	if got := table.Rows[0].Cells; !equalStringSlices(got, []string{"a1", "Alice", "-", "yes", "Etc/GMT", "en_US"}) {
		t.Errorf("row = %v", got)
	}
}

func TestPresentUserList_ExtendedDashesForRedactedFields(t *testing.T) {
	t.Parallel()
	// Simulates an instance where /user/search returns users without
	// timeZone/locale fields — the PR calls this out explicitly.
	users := []api.User{{AccountID: "a1", DisplayName: "Alice", Active: true}}
	model := UserPresenter{}.PresentUserList(users, true)
	table := sectionTable(t, model, 0)

	if got := table.Rows[0].Cells; !equalStringSlices(got, []string{"a1", "Alice", "-", "yes", "-", "-"}) {
		t.Errorf("row with redacted tz/locale = %v", got)
	}
}

func TestPresentUserDetailProjection_ContainsAllSpecHeaders(t *testing.T) {
	t.Parallel()
	// Parity between UserDetailSpec and the DetailSection that feeds
	// projection.ProjectDetail: every spec header must have a corresponding
	// Field label, otherwise --fields tokens silently project to nothing.
	user := &api.User{
		AccountID: "abc", DisplayName: "Alice", EmailAddress: "a@example.com",
		Active: true, TimeZone: "Etc/GMT", Locale: "en_US",
		Groups: &api.UserCountBlock{Size: 3}, ApplicationRoles: &api.UserCountBlock{Size: 1},
	}
	model := UserPresenter{}.PresentUserDetailProjection(user)
	detail, ok := model.Sections[0].(*present.DetailSection)
	if !ok {
		t.Fatalf("expected DetailSection, got %T", model.Sections[0])
	}
	have := make(map[string]struct{}, len(detail.Fields))
	for _, f := range detail.Fields {
		have[strings.ToLower(f.Label)] = struct{}{}
	}
	for _, c := range UserDetailSpec {
		if _, ok := have[strings.ToLower(c.Header)]; !ok {
			t.Errorf("UserDetailSpec header %q has no matching DetailSection label", c.Header)
		}
	}
}

func TestUserListSpec_HeaderParityWithPresenter(t *testing.T) {
	t.Parallel()
	// Default-mode registry must match the default-mode table headers.
	model := UserPresenter{}.PresentUserList(nil, false)
	table := sectionTable(t, model, 0)
	wantHeaders := registryHeadersFor(UserListSpec, false)
	if !equalStringSlices(table.Headers, wantHeaders) {
		t.Errorf("default headers mismatch: presenter %v vs registry %v", table.Headers, wantHeaders)
	}

	// Extended mode must also match.
	modelExt := UserPresenter{}.PresentUserList(nil, true)
	tableExt := sectionTable(t, modelExt, 0)
	wantHeadersExt := registryHeadersFor(UserListSpec, true)
	if !equalStringSlices(tableExt.Headers, wantHeadersExt) {
		t.Errorf("extended headers mismatch: presenter %v vs registry %v", tableExt.Headers, wantHeadersExt)
	}
}

func TestPresentEmptyUsers(t *testing.T) {
	t.Parallel()
	model := UserPresenter{}.PresentEmpty("ghost")
	if got := renderMessage(t, model, 0); got != "No users found matching 'ghost'" {
		t.Errorf("empty message = %q", got)
	}
}

// --- tiny test helpers shared with project_test.go ---

func renderMessage(t *testing.T, model *present.OutputModel, i int) string {
	t.Helper()
	if i >= len(model.Sections) {
		t.Fatalf("section index %d out of range (have %d)", i, len(model.Sections))
	}
	msg, ok := model.Sections[i].(*present.MessageSection)
	if !ok {
		t.Fatalf("section %d: expected MessageSection, got %T", i, model.Sections[i])
	}
	return msg.Message
}

func sectionTable(t *testing.T, model *present.OutputModel, i int) *present.TableSection {
	t.Helper()
	ts, ok := model.Sections[i].(*present.TableSection)
	if !ok {
		t.Fatalf("section %d: expected TableSection, got %T", i, model.Sections[i])
	}
	return ts
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func registryHeadersFor(r projection.Registry, extended bool) []string {
	filtered := r.ForMode(extended)
	out := make([]string, len(filtered))
	for i, c := range filtered {
		out[i] = c.Header
	}
	return out
}

func TestUserPresenter_PresentUserListWithPagination(t *testing.T) {
	t.Parallel()
	users := []api.User{{AccountID: "a", DisplayName: "U"}}

	t.Run("appends_hint", func(t *testing.T) {
		model := UserPresenter{}.PresentUserListWithPagination(users, false, true, "tok")
		if len(model.Sections) != 2 {
			t.Fatalf("want 2 sections, got %d", len(model.Sections))
		}
	})

	t.Run("no_hint", func(t *testing.T) {
		model := UserPresenter{}.PresentUserListWithPagination(users, false, false, "")
		if len(model.Sections) != 1 {
			t.Errorf("want 1 section, got %d", len(model.Sections))
		}
	})
}
