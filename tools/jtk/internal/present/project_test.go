package present

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func ptrBool(b bool) *bool { return &b }

func sampleProject() *api.ProjectDetail {
	return &api.ProjectDetail{
		ID:             json.Number("10000"),
		Key:            "MON",
		Name:           "Platform Development",
		ProjectTypeKey: "software",
		Lead:           &api.User{AccountID: "u123", DisplayName: "Rusty Hall"},
		Style:          "classic",
		Simplified:     ptrBool(false),
		IsPrivate:      ptrBool(false),
		IssueTypes: []api.IssueType{
			{ID: "10000", Name: "Epic"},
			{ID: "10025", Name: "SDLC"},
			{ID: "10026", Name: "Kanban"},
		},
		Components: []api.Component{
			{ID: "10143", Name: "Admin Portal"},
			{ID: "10144", Name: "Admin Service"},
			{ID: "10145", Name: "Banker Portal"},
			{ID: "10146", Name: "Auth Service"},
			{ID: "10147", Name: "Codat Sync Service"},
			{ID: "10148", Name: "Another Component"},
		},
		Versions:    nil,
		Description: "Long-lived platform hub project.",
	}
}

func TestPresentProjectDetail_Default(t *testing.T) {
	t.Parallel()
	model := ProjectPresenter{}.PresentProjectDetail(sampleProject(), false)

	expect := []string{
		"MON  Platform Development",
		"Type: software   Lead: Rusty Hall   Style: classic",
		"Issue Types: Epic, SDLC, Kanban",
		"Components: 6   Versions: 0",
	}
	if len(model.Sections) != len(expect) {
		t.Fatalf("sections = %d, want %d", len(model.Sections), len(expect))
	}
	for i, want := range expect {
		if got := renderMessage(t, model, i); got != want {
			t.Errorf("section %d = %q, want %q", i, got, want)
		}
	}
}

func TestPresentProjectDetail_Extended_EnumeratesComponentsWithTruncation(t *testing.T) {
	t.Parallel()
	p := sampleProject()
	model := ProjectPresenter{}.PresentProjectDetail(p, true)

	lines := renderAllMessages(t, model)
	// Title line (1) + compound KV (1) + issue types (1) + Components
	// header (1) + 4 component rows (4) + "... [2 more]" (1) + Versions (1)
	// + Simplified/Private (1) + Description label (1) + description body (1)
	// = 13 sections.
	if len(lines) != 13 {
		t.Fatalf("extended sections = %d, want 13\nlines=%v", len(lines), lines)
	}
	checks := map[int]string{
		0:  "MON  Platform Development",
		1:  "Type: software   Lead: Rusty Hall (u123)   Style: classic",
		2:  "Issue Types: Epic (10000), SDLC (10025), Kanban (10026)",
		3:  "Components: 6",
		4:  "  10143 | Admin Portal",
		7:  "  10146 | Auth Service",
		8:  "  ... [2 more]",
		9:  "Versions: 0",
		10: "Simplified: no   Private: no",
		11: "Description:",
		12: "Long-lived platform hub project.",
	}
	for i, want := range checks {
		if lines[i] != want {
			t.Errorf("line %d = %q, want %q", i, lines[i], want)
		}
	}
}

func TestPresentProjectDetail_Extended_DashesWhenSimplifiedAndPrivateMissing(t *testing.T) {
	t.Parallel()
	p := &api.ProjectDetail{
		Key: "X", Name: "Example",
		ProjectTypeKey: "software", Style: "classic",
		Lead: &api.User{AccountID: "u", DisplayName: "Lead"},
		// Simplified / IsPrivate intentionally nil — API omitted the fields.
	}
	model := ProjectPresenter{}.PresentProjectDetail(p, true)
	lines := renderAllMessages(t, model)

	// Find the Simplified/Private line (comes after Versions).
	var found bool
	for _, line := range lines {
		if strings.HasPrefix(line, "Simplified:") {
			if line != "Simplified: -   Private: -" {
				t.Errorf("got %q, want dashes for missing fields", line)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no Simplified/Private line found in %v", lines)
	}
}

func TestPresentProjectList_DefaultColumnOrderMatchesSpec(t *testing.T) {
	t.Parallel()
	// Spec order is KEY | TYPE | LEAD | NAME — deliberately not the same as
	// the pre-migration KEY | NAME | TYPE | LEAD order.
	projects := []api.ProjectDetail{
		{Key: "MON", Name: "Platform Development", ProjectTypeKey: "software", Lead: &api.User{DisplayName: "Rusty Hall"}},
		{Key: "ON", Name: "Customer Onboarding", ProjectTypeKey: "software"},
	}
	model := ProjectPresenter{}.PresentProjectList(projects, false)
	table := sectionTable(t, model, 0)

	wantHeaders := []string{"KEY", "TYPE", "LEAD", "NAME"}
	if !equalStringSlices(table.Headers, wantHeaders) {
		t.Errorf("headers = %v, want %v", table.Headers, wantHeaders)
	}
	if got := table.Rows[0].Cells; !equalStringSlices(got, []string{"MON", "software", "Rusty Hall", "Platform Development"}) {
		t.Errorf("row 0 = %v", got)
	}
	if got := table.Rows[1].Cells; !equalStringSlices(got, []string{"ON", "software", "-", "Customer Onboarding"}) {
		t.Errorf("row 1 = %v (unpaired lead expected '-')", got)
	}
}

func TestPresentProjectList_ExtendedMatchesSpecShape(t *testing.T) {
	t.Parallel()
	// Spec (#230): KEY | TYPE | STYLE | LEAD | ISSUE_TYPES | COMPONENTS | NAME.
	// ISSUE_TYPES is comma-joined issue-type names, COMPONENTS is a count.
	projects := []api.ProjectDetail{
		{
			Key: "MON", Name: "Platform Development", ProjectTypeKey: "software",
			Lead:       &api.User{DisplayName: "Rusty Hall"},
			Style:      "classic",
			IssueTypes: []api.IssueType{{ID: "1", Name: "Epic"}, {ID: "2", Name: "SDLC"}},
			Components: []api.Component{{ID: "c1", Name: "A"}, {ID: "c2", Name: "B"}, {ID: "c3", Name: "C"}},
		},
	}
	model := ProjectPresenter{}.PresentProjectList(projects, true)
	table := sectionTable(t, model, 0)

	wantHeaders := []string{"KEY", "TYPE", "STYLE", "LEAD", "ISSUE_TYPES", "COMPONENTS", "NAME"}
	if !equalStringSlices(table.Headers, wantHeaders) {
		t.Errorf("extended headers = %v, want %v", table.Headers, wantHeaders)
	}
	wantCells := []string{"MON", "software", "classic", "Rusty Hall", "Epic, SDLC", "3", "Platform Development"}
	if got := table.Rows[0].Cells; !equalStringSlices(got, wantCells) {
		t.Errorf("row = %v\nwant %v", got, wantCells)
	}
}

func TestPresentProjectList_Extended_DashForEmptyIssueTypesAndStyle(t *testing.T) {
	t.Parallel()
	projects := []api.ProjectDetail{
		{Key: "X", Name: "Example", ProjectTypeKey: "software"}, // no lead, no style, no types, no components
	}
	model := ProjectPresenter{}.PresentProjectList(projects, true)
	table := sectionTable(t, model, 0)

	wantCells := []string{"X", "software", "-", "-", "-", "0", "Example"}
	if got := table.Rows[0].Cells; !equalStringSlices(got, wantCells) {
		t.Errorf("empty-field row = %v\nwant %v", got, wantCells)
	}
}

func TestPresentProjectDetail_Default_IssueTypesRowRendersDashWhenEmpty(t *testing.T) {
	t.Parallel()
	// Fix 2 regression: the row must appear unconditionally so callers see a
	// stable line count. Empty IssueTypes renders as "Issue Types: -".
	p := &api.ProjectDetail{
		Key: "X", Name: "Example",
		ProjectTypeKey: "software",
		Lead:           &api.User{DisplayName: "Lead"},
		Style:          "classic",
	}
	model := ProjectPresenter{}.PresentProjectDetail(p, false)
	lines := renderAllMessages(t, model)

	want := []string{
		"X  Example",
		"Type: software   Lead: Lead   Style: classic",
		"Issue Types: -",
		"Components: 0   Versions: 0",
	}
	if len(lines) != len(want) {
		t.Fatalf("line count = %d, want %d\nlines=%v", len(lines), len(want), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestPresentProjectDetail_Extended_IssueTypesRowRendersDashWhenEmpty(t *testing.T) {
	t.Parallel()
	p := &api.ProjectDetail{
		Key: "X", Name: "Example",
		ProjectTypeKey: "software",
		Lead:           &api.User{AccountID: "u", DisplayName: "Lead"},
		Style:          "classic",
	}
	model := ProjectPresenter{}.PresentProjectDetail(p, true)
	lines := renderAllMessages(t, model)

	// Specifically verify the Issue Types line is the third section (after
	// title + compound row) — ordering is part of the contract.
	if lines[2] != "Issue Types: -" {
		t.Errorf("line 2 = %q, want \"Issue Types: -\"", lines[2])
	}
}

func TestPresentProjectTypes_HeaderRename(t *testing.T) {
	t.Parallel()
	types := []api.ProjectType{
		{Key: "software", FormattedKey: "Software", DescriptionI18nKey: "jira.project.type.software.description"},
	}
	model := ProjectPresenter{}.PresentProjectTypes(types, false)
	table := sectionTable(t, model, 0)
	if !equalStringSlices(table.Headers, []string{"KEY", "NAME"}) {
		t.Errorf("default headers = %v, want [KEY NAME]", table.Headers)
	}
	if got := table.Rows[0].Cells; !equalStringSlices(got, []string{"software", "Software"}) {
		t.Errorf("row = %v", got)
	}
}

func TestPresentProjectTypes_ExtendedAddsDescriptionKey(t *testing.T) {
	t.Parallel()
	types := []api.ProjectType{
		{Key: "software", FormattedKey: "Software", DescriptionI18nKey: "jira.project.type.software.description"},
	}
	model := ProjectPresenter{}.PresentProjectTypes(types, true)
	table := sectionTable(t, model, 0)
	if !equalStringSlices(table.Headers, []string{"KEY", "NAME", "DESCRIPTION_KEY"}) {
		t.Errorf("extended headers = %v", table.Headers)
	}
	if got := table.Rows[0].Cells; !equalStringSlices(got, []string{"software", "Software", "jira.project.type.software.description"}) {
		t.Errorf("row = %v", got)
	}
}

func TestProjectListSpec_HeaderParityWithPresenter(t *testing.T) {
	t.Parallel()
	for _, extended := range []bool{false, true} {
		model := ProjectPresenter{}.PresentProjectList(nil, extended)
		table := sectionTable(t, model, 0)
		want := registryHeadersFor(ProjectListSpec, extended)
		if !equalStringSlices(table.Headers, want) {
			t.Errorf("extended=%v headers mismatch: presenter %v vs registry %v",
				extended, table.Headers, want)
		}
	}
}

func TestProjectTypeSpec_HeaderParityWithPresenter(t *testing.T) {
	t.Parallel()
	for _, extended := range []bool{false, true} {
		model := ProjectPresenter{}.PresentProjectTypes(nil, extended)
		table := sectionTable(t, model, 0)
		want := registryHeadersFor(ProjectTypeSpec, extended)
		if !equalStringSlices(table.Headers, want) {
			t.Errorf("extended=%v headers mismatch: presenter %v vs registry %v",
				extended, table.Headers, want)
		}
	}
}

func TestPresentProjectDetailProjection_ContainsAllSpecHeaders(t *testing.T) {
	t.Parallel()
	model := ProjectPresenter{}.PresentProjectDetailProjection(sampleProject())
	detail, ok := model.Sections[0].(*present.DetailSection)
	if !ok {
		t.Fatalf("expected DetailSection, got %T", model.Sections[0])
	}
	have := make(map[string]struct{}, len(detail.Fields))
	for _, f := range detail.Fields {
		have[strings.ToLower(f.Label)] = struct{}{}
	}
	for _, c := range ProjectDetailSpec {
		if _, ok := have[strings.ToLower(c.Header)]; !ok {
			t.Errorf("ProjectDetailSpec header %q has no matching DetailSection label", c.Header)
		}
	}
}

func renderAllMessages(t *testing.T, model *present.OutputModel) []string {
	t.Helper()
	out := make([]string, len(model.Sections))
	for i := range model.Sections {
		out[i] = renderMessage(t, model, i)
	}
	return out
}
