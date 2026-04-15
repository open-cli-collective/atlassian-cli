package present

import (
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestSprintPresenter_PresentDetail(t *testing.T) {
	t.Parallel()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)

	sprint := &api.Sprint{
		ID:        42,
		Name:      "Sprint 1",
		State:     "active",
		Goal:      "Complete MVP",
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	p := SprintPresenter{}
	model := p.PresentDetail(sprint)

	if len(model.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(model.Sections))
	}

	detail, ok := model.Sections[0].(*present.DetailSection)
	if !ok {
		t.Fatalf("expected DetailSection, got %T", model.Sections[0])
	}

	fieldMap := make(map[string]string)
	for _, f := range detail.Fields {
		fieldMap[f.Label] = f.Value
	}

	if fieldMap["ID"] != "42" {
		t.Errorf("expected ID='42', got %q", fieldMap["ID"])
	}
	if fieldMap["Name"] != "Sprint 1" {
		t.Errorf("expected Name='Sprint 1', got %q", fieldMap["Name"])
	}
	if fieldMap["State"] != "active" {
		t.Errorf("expected State='active', got %q", fieldMap["State"])
	}
	if fieldMap["Goal"] != "Complete MVP" {
		t.Errorf("expected Goal='Complete MVP', got %q", fieldMap["Goal"])
	}
	if fieldMap["Start Date"] != "2024-01-01" {
		t.Errorf("expected Start Date='2024-01-01', got %q", fieldMap["Start Date"])
	}
	if fieldMap["End Date"] != "2024-01-14" {
		t.Errorf("expected End Date='2024-01-14', got %q", fieldMap["End Date"])
	}
}

func TestSprintPresenter_PresentDetail_MinimalFields(t *testing.T) {
	t.Parallel()
	sprint := &api.Sprint{
		ID:    1,
		Name:  "Backlog",
		State: "future",
		// No goal, no dates
	}

	p := SprintPresenter{}
	model := p.PresentDetail(sprint)

	detail := model.Sections[0].(*present.DetailSection)

	// Should only have ID, Name, State
	if len(detail.Fields) != 3 {
		t.Errorf("expected 3 fields for minimal sprint, got %d", len(detail.Fields))
	}
}

func TestSprintPresenter_PresentList(t *testing.T) {
	t.Parallel()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)

	sprints := []api.Sprint{
		{
			ID:        1,
			Name:      "Sprint 1",
			State:     "closed",
			StartDate: &startDate,
			EndDate:   &endDate,
		},
		{
			ID:    2,
			Name:  "Sprint 2",
			State: "active",
			// No dates yet
		},
	}

	p := SprintPresenter{}
	model := p.PresentList(sprints)

	table, ok := model.Sections[0].(*present.TableSection)
	if !ok {
		t.Fatalf("expected TableSection, got %T", model.Sections[0])
	}

	// Verify headers
	expectedHeaders := []string{"ID", "NAME", "STATE", "START", "END"}
	if len(table.Headers) != len(expectedHeaders) {
		t.Errorf("expected %d headers, got %d", len(expectedHeaders), len(table.Headers))
	}

	// Verify rows
	if len(table.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(table.Rows))
	}

	// Row 1 - with dates
	if table.Rows[0].Cells[0] != "1" {
		t.Errorf("row 0 ID: expected '1', got %q", table.Rows[0].Cells[0])
	}
	if table.Rows[0].Cells[3] != "2024-01-01" {
		t.Errorf("row 0 start: expected '2024-01-01', got %q", table.Rows[0].Cells[3])
	}

	// Row 2 - no dates
	if table.Rows[1].Cells[3] != "" {
		t.Errorf("row 1 start: expected empty for nil date, got %q", table.Rows[1].Cells[3])
	}
}
