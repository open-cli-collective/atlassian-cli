package present

import (
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestIssuePresenter_PresentDetail_Default(t *testing.T) {
	t.Parallel()
	issue := &api.Issue{
		Key: "PROJ-123",
		Fields: api.IssueFields{
			Summary:   "Fix the bug",
			Status:    &api.Status{Name: "In Progress"},
			IssueType: &api.IssueType{Name: "Bug"},
			Priority:  &api.Priority{Name: "High"},
			Assignee:  &api.User{DisplayName: "Alice"},
			Project:   &api.Project{Key: "PROJ"},
			Updated:   "2026-04-16",
		},
	}

	p := IssuePresenter{}
	model := p.PresentDetail(issue, "https://jira.example.com/browse/PROJ-123", false, false)

	if len(model.Sections) < 2 {
		t.Fatalf("expected at least 2 sections, got %d", len(model.Sections))
	}

	rendered := renderMsgSections(model)
	if !strings.Contains(rendered, "PROJ-123  Fix the bug") {
		t.Errorf("missing title line in output:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Status: In Progress") {
		t.Errorf("missing Status in output:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Assignee: Alice") {
		t.Errorf("missing Assignee in output:\n%s", rendered)
	}
}

func TestIssuePresenter_PresentDetail_Unassigned(t *testing.T) {
	t.Parallel()
	issue := &api.Issue{
		Key: "PROJ-123",
		Fields: api.IssueFields{
			Summary: "Unassigned issue",
		},
	}

	p := IssuePresenter{}
	model := p.PresentDetail(issue, "https://jira.example.com/browse/PROJ-123", false, false)

	rendered := renderMsgSections(model)
	if !strings.Contains(rendered, "Assignee: Unassigned") {
		t.Errorf("expected 'Assignee: Unassigned' for nil assignee, got:\n%s", rendered)
	}
}

func TestIssuePresenter_PresentDetail_Extended(t *testing.T) {
	t.Parallel()
	issue := &api.Issue{
		Key: "PROJ-123",
		Fields: api.IssueFields{
			Summary:   "Fix the bug",
			Status:    &api.Status{Name: "In Progress", StatusCategory: api.StatusCategory{Name: "In Progress"}},
			IssueType: &api.IssueType{Name: "Bug"},
			Assignee:  &api.User{DisplayName: "Alice", AccountID: "abc123"},
			Reporter:  &api.User{DisplayName: "Bob", AccountID: "def456"},
			Created:   "2026-04-15T10:00:00+0000",
			Updated:   "2026-04-16T07:16:24+0000",
		},
	}

	p := IssuePresenter{}
	model := p.PresentDetail(issue, "https://jira.example.com/browse/PROJ-123", true, false)

	rendered := renderMsgSections(model)
	if !strings.Contains(rendered, "category: In Progress") {
		t.Errorf("expected status category in extended output:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Reporter: Bob (def456)") {
		t.Errorf("expected reporter with ID in extended output:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Created:") {
		t.Errorf("expected Created in extended output:\n%s", rendered)
	}
}

func TestIssuePresenter_PresentList_Default(t *testing.T) {
	t.Parallel()
	issues := []api.Issue{
		{
			Key: "PROJ-1",
			Fields: api.IssueFields{
				Summary:   "First issue",
				Status:    &api.Status{Name: "Done"},
				Assignee:  &api.User{DisplayName: "Bob"},
				IssueType: &api.IssueType{Name: "Task"},
			},
		},
		{
			Key: "PROJ-2",
			Fields: api.IssueFields{
				Summary:   "Second issue",
				Status:    &api.Status{Name: "Open"},
				IssueType: &api.IssueType{Name: "Bug"},
			},
		},
	}

	p := IssuePresenter{}
	model := p.PresentList(issues, false)

	if len(model.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(model.Sections))
	}

	table, ok := model.Sections[0].(*present.TableSection)
	if !ok {
		t.Fatalf("expected TableSection, got %T", model.Sections[0])
	}

	expectedHeaders := []string{"KEY", "STATUS", "TYPE", "PTS", "ASSIGNEE", "SUMMARY"}
	if len(table.Headers) != len(expectedHeaders) {
		t.Errorf("expected %d headers, got %d", len(expectedHeaders), len(table.Headers))
	}
	for i, h := range expectedHeaders {
		if i < len(table.Headers) && table.Headers[i] != h {
			t.Errorf("header[%d]: expected %q, got %q", i, h, table.Headers[i])
		}
	}

	if len(table.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(table.Rows))
	}

	if table.Rows[0].Cells[0] != "PROJ-1" {
		t.Errorf("row 0 key: expected 'PROJ-1', got %q", table.Rows[0].Cells[0])
	}
	if table.Rows[0].Cells[4] != "Bob" {
		t.Errorf("row 0 assignee: expected 'Bob', got %q", table.Rows[0].Cells[4])
	}
	if table.Rows[1].Cells[4] != "Unassigned" {
		t.Errorf("row 1 assignee: expected 'Unassigned', got %q", table.Rows[1].Cells[4])
	}
}

func TestIssuePresenter_PresentList_Extended(t *testing.T) {
	t.Parallel()
	issues := []api.Issue{
		{
			Key: "PROJ-1",
			Fields: api.IssueFields{
				Summary:   "First issue",
				Status:    &api.Status{Name: "Done"},
				Assignee:  &api.User{DisplayName: "Bob"},
				Reporter:  &api.User{DisplayName: "Alice"},
				IssueType: &api.IssueType{Name: "Task"},
				Sprint:    &api.Sprint{Name: "Sprint 1"},
				Parent:    &api.Issue{Key: "PROJ-100"},
				Updated:   "2026-04-16",
				Labels:    []string{"bug", "urgent"},
				Components: []api.Component{
					{ID: "1", Name: "Backend"},
				},
			},
		},
	}

	p := IssuePresenter{}
	model := p.PresentList(issues, true)

	table := model.Sections[0].(*present.TableSection)

	expectedHeaders := []string{"KEY", "STATUS", "TYPE", "PTS", "ASSIGNEE", "REPORTER", "SPRINT", "PARENT", "UPDATED", "LABELS", "COMPONENTS", "SUMMARY"}
	if len(table.Headers) != len(expectedHeaders) {
		t.Fatalf("expected %d headers, got %d", len(expectedHeaders), len(table.Headers))
	}
	for i, h := range expectedHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d]: expected %q, got %q", i, h, table.Headers[i])
		}
	}

	row := table.Rows[0]
	if row.Cells[5] != "Alice" {
		t.Errorf("reporter: expected 'Alice', got %q", row.Cells[5])
	}
	if row.Cells[6] != "Sprint 1" {
		t.Errorf("sprint: expected 'Sprint 1', got %q", row.Cells[6])
	}
	if row.Cells[7] != "PROJ-100" {
		t.Errorf("parent: expected 'PROJ-100', got %q", row.Cells[7])
	}
}

func TestIssuePresenter_PresentTypes(t *testing.T) {
	t.Parallel()
	types := []api.IssueType{
		{ID: "1", Name: "Bug", Subtask: false, Description: "A bug in the software"},
		{ID: "2", Name: "Sub-task", Subtask: true, Description: "A subtask of another issue"},
	}

	p := IssuePresenter{}
	model := p.PresentTypes(types)

	table := model.Sections[0].(*present.TableSection)

	if len(table.Headers) != 4 {
		t.Errorf("expected 4 headers, got %d", len(table.Headers))
	}
	if len(table.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(table.Rows))
	}

	if table.Rows[0].Cells[2] != "no" {
		t.Errorf("Bug subtask: expected 'no', got %q", table.Rows[0].Cells[2])
	}
	if table.Rows[1].Cells[2] != "yes" {
		t.Errorf("Sub-task subtask: expected 'yes', got %q", table.Rows[1].Cells[2])
	}
}

// TestIssueListSpec_MatchesPresentListHeaders locks the IssueListSpec default
// headers against the hardcoded headers in PresentList(extended=false).
func TestIssueListSpec_MatchesPresentListHeaders(t *testing.T) {
	t.Parallel()
	issues := []api.Issue{{Key: "PROJ-1", Fields: api.IssueFields{Summary: "x"}}}

	defaultSpecs := IssueListSpec.ForMode(false)

	model := IssuePresenter{}.PresentList(issues, false)
	table := model.Sections[0].(*present.TableSection)

	if len(table.Headers) != len(defaultSpecs) {
		t.Fatalf("header count mismatch: spec has %d, table has %d", len(defaultSpecs), len(table.Headers))
	}
	for i, spec := range defaultSpecs {
		if table.Headers[i] != spec.Header {
			t.Errorf("index %d: spec Header=%q, table header=%q", i, spec.Header, table.Headers[i])
		}
	}
}

// TestIssueListSpec_ExtendedMatchesPresentListHeaders locks the extended spec
// headers against PresentList(extended=true).
func TestIssueListSpec_ExtendedMatchesPresentListHeaders(t *testing.T) {
	t.Parallel()
	issues := []api.Issue{{Key: "PROJ-1", Fields: api.IssueFields{Summary: "x"}}}

	extendedSpecs := IssueListSpec.ForMode(true)

	model := IssuePresenter{}.PresentList(issues, true)
	table := model.Sections[0].(*present.TableSection)

	if len(table.Headers) != len(extendedSpecs) {
		t.Fatalf("header count mismatch: spec has %d, table has %d", len(extendedSpecs), len(table.Headers))
	}
	for i, spec := range extendedSpecs {
		if table.Headers[i] != spec.Header {
			t.Errorf("index %d: spec Header=%q, table header=%q", i, spec.Header, table.Headers[i])
		}
	}
}

// TestIssueDetailSpec_MatchesPresentDetailProjectionLabels verifies the
// projection path stays in sync with the detail spec.
func TestIssueDetailSpec_MatchesPresentDetailProjectionLabels(t *testing.T) {
	t.Parallel()
	issue := &api.Issue{
		Key: "PROJ-1",
		Fields: api.IssueFields{
			Summary:     "s",
			Status:      &api.Status{Name: "Open"},
			IssueType:   &api.IssueType{Name: "Bug"},
			Priority:    &api.Priority{Name: "High"},
			Assignee:    &api.User{DisplayName: "Alice"},
			Reporter:    &api.User{DisplayName: "Bob"},
			Project:     &api.Project{Key: "PROJ"},
			Description: &api.Description{},
		},
	}
	model := IssuePresenter{}.PresentDetailProjection(issue, "https://example.com/PROJ-1", true)
	detail := model.Sections[0].(*present.DetailSection)

	renderedLabels := make(map[string]bool, len(detail.Fields))
	for _, f := range detail.Fields {
		renderedLabels[f.Label] = true
	}
	for _, spec := range IssueDetailSpec {
		if !renderedLabels[spec.Header] {
			t.Errorf("spec Header %q not emitted by PresentDetailProjection", spec.Header)
		}
	}

	specLabels := make(map[string]bool, len(IssueDetailSpec))
	for _, spec := range IssueDetailSpec {
		specLabels[spec.Header] = true
	}
	for _, f := range detail.Fields {
		if !specLabels[f.Label] {
			t.Errorf("rendered field %q has no matching IssueDetailSpec entry", f.Label)
		}
	}
}

func TestIssuePresenter_PresentTypeNotFound(t *testing.T) {
	t.Parallel()
	p := IssuePresenter{}
	model := p.PresentTypeNotFound("Story", "PROJ", []string{"Bug", "Task", "Epic"})

	if len(model.Sections) != 5 {
		t.Fatalf("expected 5 sections, got %d", len(model.Sections))
	}

	errMsg := model.Sections[0].(*present.MessageSection)
	if errMsg.Kind != present.MessageError {
		t.Errorf("first section should be error, got %v", errMsg.Kind)
	}

	for i, s := range model.Sections {
		msg := s.(*present.MessageSection)
		if msg.Stream != present.StreamStderr {
			t.Errorf("section %d should go to stderr", i)
		}
	}
}

func TestIssuePresenter_PresentMoveInitiated(t *testing.T) {
	t.Parallel()
	p := IssuePresenter{}
	model := p.PresentMoveInitiated("task-123")

	if len(model.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(model.Sections))
	}

	success := model.Sections[0].(*present.MessageSection)
	if success.Kind != present.MessageSuccess {
		t.Errorf("expected success, got %v", success.Kind)
	}
}

func TestIssuePresenter_PresentMovePartialFailure(t *testing.T) {
	t.Parallel()
	p := IssuePresenter{}
	successful := []string{"PROJ-1", "PROJ-2"}
	failed := []api.MoveFailedIssue{
		{IssueKey: "PROJ-3", Errors: []string{"Invalid type"}},
	}
	model := p.PresentMovePartialFailure(successful, failed)

	if len(model.Sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(model.Sections))
	}

	warn := model.Sections[0].(*present.MessageSection)
	if warn.Kind != present.MessageWarning {
		t.Errorf("expected warning, got %v", warn.Kind)
	}
}

func TestIssuePresenter_PresentMovePartialFailure_NoSuccessful(t *testing.T) {
	t.Parallel()
	p := IssuePresenter{}
	failed := []api.MoveFailedIssue{
		{IssueKey: "PROJ-1", Errors: []string{"Error 1"}},
	}
	model := p.PresentMovePartialFailure(nil, failed)

	if len(model.Sections) != 2 {
		t.Errorf("expected 2 sections when no successful, got %d", len(model.Sections))
	}
}

func TestStoryPoints_Formatting(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		custom map[string]any
		want   string
	}{
		{"nil map", nil, "-"},
		{"missing key", map[string]any{}, "-"},
		{"null value", map[string]any{"customfield_10035": nil}, "-"},
		{"integer 5", map[string]any{"customfield_10035": float64(5)}, "5"},
		{"float 3.5", map[string]any{"customfield_10035": float64(3.5)}, "3.5"},
		{"string value", map[string]any{"customfield_10035": "five"}, "five"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issue := &api.Issue{
				Fields: api.IssueFields{CustomFields: tc.custom},
			}
			got := formatStoryPoints(issue)
			if got != tc.want {
				t.Errorf("formatStoryPoints: got %q, want %q", got, tc.want)
			}
		})
	}
}

func renderMsgSections(model *present.OutputModel) string {
	out := present.Render(model, present.StyleAgent)
	return out.Stdout + out.Stderr
}
