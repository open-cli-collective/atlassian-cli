package present

import (
	"encoding/json"
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestAutomationPresenter_PresentDetail(t *testing.T) {
	t.Parallel()
	rule := &api.AutomationRule{
		ID:          json.Number("123"),
		RuleKey:     "abc-123",
		Name:        "Test Rule",
		State:       "ENABLED",
		Description: "A test rule",
		Labels:      []string{"label1", "label2"},
		Components: []api.RuleComponent{
			{Component: "TRIGGER", Type: "issue.created"},
			{Component: "ACTION", Type: "assign.issue"},
		},
	}

	p := AutomationPresenter{}
	model := p.PresentDetail(rule, false)

	if len(model.Sections) != 1 {
		t.Fatalf("expected 1 section without components, got %d", len(model.Sections))
	}

	detail, ok := model.Sections[0].(*present.DetailSection)
	if !ok {
		t.Fatalf("expected DetailSection, got %T", model.Sections[0])
	}

	fieldMap := make(map[string]string)
	for _, f := range detail.Fields {
		fieldMap[f.Label] = f.Value
	}

	if fieldMap["Name"] != "Test Rule" {
		t.Errorf("expected Name='Test Rule', got %q", fieldMap["Name"])
	}
	if fieldMap["State"] != "ENABLED" {
		t.Errorf("expected State='ENABLED', got %q", fieldMap["State"])
	}
}

func TestAutomationPresenter_PresentDetail_WithComponents(t *testing.T) {
	t.Parallel()
	rule := &api.AutomationRule{
		Name:  "Rule",
		State: "ENABLED",
		Components: []api.RuleComponent{
			{Component: "TRIGGER", Type: "issue.created"},
			{Component: "ACTION", Type: "assign.issue"},
		},
	}

	p := AutomationPresenter{}
	model := p.PresentDetail(rule, true) // showComponents=true

	// Should have 2 sections: detail + component table
	if len(model.Sections) != 2 {
		t.Fatalf("expected 2 sections with components, got %d", len(model.Sections))
	}

	table, ok := model.Sections[1].(*present.TableSection)
	if !ok {
		t.Fatalf("expected TableSection for components, got %T", model.Sections[1])
	}

	if len(table.Rows) != 2 {
		t.Errorf("expected 2 component rows, got %d", len(table.Rows))
	}
}

func TestAutomationPresenter_PresentUpdateComplete(t *testing.T) {
	t.Parallel()
	p := AutomationPresenter{}
	model := p.PresentUpdateComplete("My Rule", "uuid-123", "ENABLED", "456")

	// Should have 2 sections: progress (stderr) + success (stdout)
	if len(model.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(model.Sections))
	}

	progress := model.Sections[0].(*present.MessageSection)
	if progress.Kind != present.MessageInfo {
		t.Errorf("expected info for progress, got %v", progress.Kind)
	}
	if progress.Stream != present.StreamStderr {
		t.Errorf("progress should go to stderr")
	}

	success := model.Sections[1].(*present.MessageSection)
	if success.Kind != present.MessageSuccess {
		t.Errorf("expected success, got %v", success.Kind)
	}
	if success.Stream != present.StreamStdout {
		t.Errorf("success should go to stdout")
	}
}

func TestAutomationPresenter_PresentStateChanged(t *testing.T) {
	t.Parallel()
	p := AutomationPresenter{}
	model := p.PresentStateChanged("My Rule", "DISABLED", "ENABLED")

	if len(model.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(model.Sections))
	}

	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageSuccess {
		t.Errorf("expected success, got %v", msg.Kind)
	}
	if msg.Stream != present.StreamStdout {
		t.Errorf("state change should go to stdout")
	}
}

func TestAutomationPresenter_PresentNoChange(t *testing.T) {
	t.Parallel()
	p := AutomationPresenter{}
	model := p.PresentNoChange("My Rule", "ENABLED")

	msg := model.Sections[0].(*present.MessageSection)
	if msg.Kind != present.MessageInfo {
		t.Errorf("expected info for no-change, got %v", msg.Kind)
	}
	if msg.Stream != present.StreamStderr {
		t.Errorf("no-change should go to stderr (advisory)")
	}
}

func TestSummarizeComponents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		components []api.RuleComponent
		want       string
	}{
		{
			name:       "empty",
			components: nil,
			want:       "none",
		},
		{
			name: "mixed",
			components: []api.RuleComponent{
				{Component: "TRIGGER"},
				{Component: "CONDITION"},
				{Component: "CONDITION"},
				{Component: "ACTION"},
				{Component: "ACTION"},
				{Component: "ACTION"},
			},
			want: "6 total — 1 trigger(s), 2 condition(s), 3 action(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeComponents(tt.components)
			if got != tt.want {
				t.Errorf("SummarizeComponents() = %q, want %q", got, tt.want)
			}
		})
	}
}
