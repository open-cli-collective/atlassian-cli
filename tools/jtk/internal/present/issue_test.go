package present

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestIssuePresenter_PresentDetail(t *testing.T) {
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
		},
	}

	p := IssuePresenter{}
	model := p.PresentDetail(issue, "https://jira.example.com/browse/PROJ-123", false)

	if len(model.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(model.Sections))
	}

	detail, ok := model.Sections[0].(*present.DetailSection)
	if !ok {
		t.Fatalf("expected DetailSection, got %T", model.Sections[0])
	}

	// Verify key fields are present
	fieldMap := make(map[string]string)
	for _, f := range detail.Fields {
		fieldMap[f.Label] = f.Value
	}

	if fieldMap["Key"] != "PROJ-123" {
		t.Errorf("expected Key='PROJ-123', got %q", fieldMap["Key"])
	}
	if fieldMap["Summary"] != "Fix the bug" {
		t.Errorf("expected Summary='Fix the bug', got %q", fieldMap["Summary"])
	}
	if fieldMap["Status"] != "In Progress" {
		t.Errorf("expected Status='In Progress', got %q", fieldMap["Status"])
	}
	if fieldMap["Assignee"] != "Alice" {
		t.Errorf("expected Assignee='Alice', got %q", fieldMap["Assignee"])
	}
	if fieldMap["URL"] != "https://jira.example.com/browse/PROJ-123" {
		t.Errorf("expected URL to be set, got %q", fieldMap["URL"])
	}
}

func TestIssuePresenter_PresentDetail_Unassigned(t *testing.T) {
	t.Parallel()
	issue := &api.Issue{
		Key: "PROJ-123",
		Fields: api.IssueFields{
			Summary: "Unassigned issue",
			// Assignee is nil
		},
	}

	p := IssuePresenter{}
	model := p.PresentDetail(issue, "https://jira.example.com/browse/PROJ-123", false)

	detail := model.Sections[0].(*present.DetailSection)
	fieldMap := make(map[string]string)
	for _, f := range detail.Fields {
		fieldMap[f.Label] = f.Value
	}

	if fieldMap["Assignee"] != "Unassigned" {
		t.Errorf("expected Assignee='Unassigned' for nil assignee, got %q", fieldMap["Assignee"])
	}
}

func TestIssuePresenter_PresentList(t *testing.T) {
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
				// Assignee is nil
			},
		},
	}

	p := IssuePresenter{}
	model := p.PresentList(issues)

	if len(model.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(model.Sections))
	}

	table, ok := model.Sections[0].(*present.TableSection)
	if !ok {
		t.Fatalf("expected TableSection, got %T", model.Sections[0])
	}

	// Verify headers
	expectedHeaders := []string{"KEY", "SUMMARY", "STATUS", "ASSIGNEE", "TYPE"}
	if len(table.Headers) != len(expectedHeaders) {
		t.Errorf("expected %d headers, got %d", len(expectedHeaders), len(table.Headers))
	}
	for i, h := range expectedHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d]: expected %q, got %q", i, h, table.Headers[i])
		}
	}

	// Verify rows
	if len(table.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(table.Rows))
	}

	// Row 1
	if table.Rows[0].Cells[0] != "PROJ-1" {
		t.Errorf("row 0 key: expected 'PROJ-1', got %q", table.Rows[0].Cells[0])
	}
	if table.Rows[0].Cells[3] != "Bob" {
		t.Errorf("row 0 assignee: expected 'Bob', got %q", table.Rows[0].Cells[3])
	}

	// Row 2 - unassigned
	if table.Rows[1].Cells[3] != "Unassigned" {
		t.Errorf("row 1 assignee: expected 'Unassigned', got %q", table.Rows[1].Cells[3])
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

	// Headers: ID, NAME, SUBTASK, DESCRIPTION
	if len(table.Headers) != 4 {
		t.Errorf("expected 4 headers, got %d", len(table.Headers))
	}
	if len(table.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(table.Rows))
	}

	// Verify subtask display (lowercase)
	if table.Rows[0].Cells[2] != "no" {
		t.Errorf("Bug subtask: expected 'no', got %q", table.Rows[0].Cells[2])
	}
	if table.Rows[1].Cells[2] != "yes" {
		t.Errorf("Sub-task subtask: expected 'yes', got %q", table.Rows[1].Cells[2])
	}

	// Verify description is included
	if table.Rows[0].Cells[3] != "A bug in the software" {
		t.Errorf("Bug description: expected 'A bug in the software', got %q", table.Rows[0].Cells[3])
	}
}

func TestIssuePresenter_PresentListWithPagination_NoMore(t *testing.T) {
	t.Parallel()
	issues := []api.Issue{
		{Key: "PROJ-1", Fields: api.IssueFields{Summary: "Issue 1"}},
	}

	p := IssuePresenter{}
	model := p.PresentListWithPagination(issues, false)

	// Should have only 1 section (table, no pagination hint)
	if len(model.Sections) != 1 {
		t.Errorf("expected 1 section, got %d", len(model.Sections))
	}
	if _, ok := model.Sections[0].(*present.TableSection); !ok {
		t.Errorf("expected TableSection, got %T", model.Sections[0])
	}
}

func TestIssuePresenter_PresentListWithPagination_HasMore(t *testing.T) {
	t.Parallel()
	issues := []api.Issue{
		{Key: "PROJ-1", Fields: api.IssueFields{Summary: "Issue 1"}},
	}

	p := IssuePresenter{}
	model := p.PresentListWithPagination(issues, true)

	// Should have 2 sections (table + pagination hint)
	if len(model.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(model.Sections))
	}

	msg, ok := model.Sections[1].(*present.MessageSection)
	if !ok {
		t.Fatalf("expected MessageSection for pagination hint, got %T", model.Sections[1])
	}
	// Pagination continuation routes to stdout (inline with data rows) per #230
	// so agents reading a single stream see both data and the hint.
	if msg.Stream != present.StreamStdout {
		t.Errorf("pagination hint should go to stdout, got %v", msg.Stream)
	}
}

func TestIssuePresenter_PresentTypeNotFound(t *testing.T) {
	t.Parallel()
	p := IssuePresenter{}
	model := p.PresentTypeNotFound("Story", "PROJ", []string{"Bug", "Task", "Epic"})

	// Should have: error + "Available types" header + 3 type entries = 5 sections
	if len(model.Sections) != 5 {
		t.Fatalf("expected 5 sections, got %d", len(model.Sections))
	}

	// First section is error
	errMsg := model.Sections[0].(*present.MessageSection)
	if errMsg.Kind != present.MessageError {
		t.Errorf("first section should be error, got %v", errMsg.Kind)
	}
	if errMsg.Stream != present.StreamStderr {
		t.Errorf("error should go to stderr")
	}

	// All sections should go to stderr
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

	// Should have 2 sections: success + info hint
	if len(model.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(model.Sections))
	}

	success := model.Sections[0].(*present.MessageSection)
	if success.Kind != present.MessageSuccess {
		t.Errorf("expected success, got %v", success.Kind)
	}
	if success.Stream != present.StreamStdout {
		t.Errorf("success should go to stdout")
	}

	info := model.Sections[1].(*present.MessageSection)
	if info.Kind != present.MessageInfo {
		t.Errorf("expected info, got %v", info.Kind)
	}
	if info.Stream != present.StreamStdout {
		t.Errorf("info hint should go to stdout")
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

	// Should have: warning + 1 error + 1 success = 3 sections
	if len(model.Sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(model.Sections))
	}

	// Warning first
	warn := model.Sections[0].(*present.MessageSection)
	if warn.Kind != present.MessageWarning {
		t.Errorf("expected warning, got %v", warn.Kind)
	}
	if warn.Stream != present.StreamStderr {
		t.Errorf("warning should go to stderr")
	}

	// Error second
	errMsg := model.Sections[1].(*present.MessageSection)
	if errMsg.Kind != present.MessageError {
		t.Errorf("expected error, got %v", errMsg.Kind)
	}
	if errMsg.Stream != present.StreamStderr {
		t.Errorf("error should go to stderr")
	}

	// Success third
	success := model.Sections[2].(*present.MessageSection)
	if success.Kind != present.MessageSuccess {
		t.Errorf("expected success, got %v", success.Kind)
	}
	if success.Stream != present.StreamStdout {
		t.Errorf("success should go to stdout")
	}
}

func TestIssuePresenter_PresentMovePartialFailure_NoSuccessful(t *testing.T) {
	t.Parallel()
	p := IssuePresenter{}
	failed := []api.MoveFailedIssue{
		{IssueKey: "PROJ-1", Errors: []string{"Error 1"}},
	}
	model := p.PresentMovePartialFailure(nil, failed)

	// Should have: warning + 1 error = 2 sections (no success section)
	if len(model.Sections) != 2 {
		t.Errorf("expected 2 sections when no successful, got %d", len(model.Sections))
	}
}

// TestIssueListSpec_MatchesPresentListHeaders locks the IssueListSpec headers
// against the hardcoded headers in PresentList AND PresentListWithPagination.
// Commands call PresentListWithPagination (not PresentList); drift between
// the two method implementations and the registry would silently break
// ProjectTable in production.
func TestIssueListSpec_MatchesPresentListHeaders(t *testing.T) {
	t.Parallel()
	issues := []api.Issue{{Key: "PROJ-1", Fields: api.IssueFields{Summary: "x"}}}

	cases := []struct {
		name  string
		model *present.OutputModel
	}{
		{"PresentList", IssuePresenter{}.PresentList(issues)},
		{"PresentListWithPagination_NoMore", IssuePresenter{}.PresentListWithPagination(issues, false)},
		{"PresentListWithPagination_HasMore", IssuePresenter{}.PresentListWithPagination(issues, true)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var table *present.TableSection
			for _, s := range tc.model.Sections {
				if ts, ok := s.(*present.TableSection); ok {
					table = ts
					break
				}
			}
			if table == nil {
				t.Fatalf("no TableSection in %s output", tc.name)
			}
			if len(table.Headers) != len(IssueListSpec) {
				t.Fatalf("header count mismatch: spec has %d, table has %d", len(IssueListSpec), len(table.Headers))
			}
			for i, spec := range IssueListSpec {
				if table.Headers[i] != spec.Header {
					t.Errorf("index %d: spec Header=%q, table header=%q", i, spec.Header, table.Headers[i])
				}
			}
		})
	}
}

// TestIssueDetailSpec_MatchesPresentDetailLabels locks the IssueDetailSpec
// entries against the Field labels emitted by PresentDetail, both directions:
//   - Every spec entry must appear in the rendered output.
//   - Every rendered field must have a matching spec entry — otherwise
//     --fields projection would silently drop it.
//
// Order is also checked: IssueDetailSpec's doc comment claims the spec order
// matches the PresentDetail Field order, and ProjectDetail relies on that
// for deterministic projection output.
//
// Description is conditional in PresentDetail; this test constructs an issue
// with a description present so all spec entries should be rendered.
func TestIssueDetailSpec_MatchesPresentDetailLabels(t *testing.T) {
	t.Parallel()
	issue := &api.Issue{
		Key: "PROJ-1",
		Fields: api.IssueFields{
			Summary:     "s",
			Status:      &api.Status{Name: "Open"},
			IssueType:   &api.IssueType{Name: "Bug"},
			Priority:    &api.Priority{Name: "High"},
			Assignee:    &api.User{DisplayName: "Alice"},
			Project:     &api.Project{Key: "PROJ"},
			Description: &api.Description{Text: "body text"},
		},
	}
	model := IssuePresenter{}.PresentDetail(issue, "https://example.com/PROJ-1", true)
	detail := model.Sections[0].(*present.DetailSection)

	// Spec → rendered: every spec entry has a corresponding rendered field.
	renderedLabels := make(map[string]bool, len(detail.Fields))
	for _, f := range detail.Fields {
		renderedLabels[f.Label] = true
	}
	for _, spec := range IssueDetailSpec {
		if !renderedLabels[spec.Header] {
			t.Errorf("spec Header %q not emitted by PresentDetail", spec.Header)
		}
	}

	// Rendered → spec: every rendered field has a corresponding spec entry.
	// Without this, a new field added to PresentDetail would be silently
	// unreachable via --fields.
	specLabels := make(map[string]bool, len(IssueDetailSpec))
	for _, spec := range IssueDetailSpec {
		specLabels[spec.Header] = true
	}
	for _, f := range detail.Fields {
		if !specLabels[f.Label] {
			t.Errorf("rendered field %q has no matching IssueDetailSpec entry", f.Label)
		}
	}

	// Order: the spec must list entries in the same relative order as
	// PresentDetail's Field order, so ProjectDetail output is deterministic.
	specOrder := make([]string, 0, len(IssueDetailSpec))
	for _, spec := range IssueDetailSpec {
		specOrder = append(specOrder, spec.Header)
	}
	renderedOrder := make([]string, 0, len(detail.Fields))
	for _, f := range detail.Fields {
		renderedOrder = append(renderedOrder, f.Label)
	}
	if len(specOrder) != len(renderedOrder) {
		t.Fatalf("spec has %d entries, rendered has %d", len(specOrder), len(renderedOrder))
	}
	for i := range specOrder {
		if specOrder[i] != renderedOrder[i] {
			t.Errorf("order mismatch at index %d: spec=%q rendered=%q", i, specOrder[i], renderedOrder[i])
		}
	}
}
