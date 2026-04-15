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
