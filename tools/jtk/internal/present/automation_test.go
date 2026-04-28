package present

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/open-cli-collective/atlassian-go/atime"
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
	model := p.PresentDetail(rule, true)

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

	msgSec := model.Sections[0].(*present.MessageSection)
	if msgSec.Kind != present.MessageSuccess {
		t.Errorf("expected success, got %v", msgSec.Kind)
	}
	if msgSec.Stream != present.StreamStdout {
		t.Errorf("state change should go to stdout")
	}
}

func TestAutomationPresenter_PresentNoChange(t *testing.T) {
	t.Parallel()
	p := AutomationPresenter{}
	model := p.PresentNoChange("My Rule", "ENABLED")

	msgSec := model.Sections[0].(*present.MessageSection)
	if msgSec.Kind != present.MessageInfo {
		t.Errorf("expected info for no-change, got %v", msgSec.Kind)
	}
	if msgSec.Stream != present.StreamStderr {
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
			want: "6 total — 1 trigger, 2 conditions, 3 actions",
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

func TestAutomationPresenter_PresentList_ColumnOrder(t *testing.T) {
	t.Parallel()
	rules := []api.AutomationRuleSummary{
		{UUID: "uuid-1", Name: "Rule A", State: "ENABLED"},
		{UUID: "uuid-2", Name: "Rule B", State: "DISABLED"},
	}

	model := AutomationPresenter{}.PresentList(rules)
	table := model.Sections[0].(*present.TableSection)

	wantHeaders := []string{"ID", "STATE", "NAME"}
	for i, h := range wantHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d] = %q, want %q", i, table.Headers[i], h)
		}
	}
	if table.Rows[0].Cells[0] != "uuid-1" {
		t.Errorf("first cell should be ID, got %q", table.Rows[0].Cells[0])
	}
	if table.Rows[0].Cells[1] != "ENABLED" {
		t.Errorf("second cell should be STATE, got %q", table.Rows[0].Cells[1])
	}
	if table.Rows[0].Cells[2] != "Rule A" {
		t.Errorf("third cell should be NAME, got %q", table.Rows[0].Cells[2])
	}
}

func TestAutomationPresenter_PresentListExtended(t *testing.T) {
	t.Parallel()
	rules := []api.AutomationRuleSummary{
		{
			UUID:            "uuid-1",
			Name:            "Rule A",
			State:           "ENABLED",
			Labels:          []string{"onboarding"},
			Tags:            []string{"auto-create"},
			AuthorAccountID: "acct-1",
		},
		{
			UUID:  "uuid-2",
			Name:  "Rule B",
			State: "DISABLED",
		},
	}

	authorNames := map[string]string{"acct-1": "Rian Stockbower"}
	model := AutomationPresenter{}.PresentListExtended(rules, authorNames)
	table := model.Sections[0].(*present.TableSection)

	wantHeaders := []string{"ID", "STATE", "LABELS", "TAGS", "AUTHOR", "NAME"}
	for i, h := range wantHeaders {
		if table.Headers[i] != h {
			t.Errorf("header[%d] = %q, want %q", i, table.Headers[i], h)
		}
	}

	if table.Rows[0].Cells[4] != "Rian Stockbower" {
		t.Errorf("author should be resolved, got %q", table.Rows[0].Cells[4])
	}
	if table.Rows[1].Cells[2] != "-" {
		t.Errorf("empty labels should be dash, got %q", table.Rows[1].Cells[2])
	}
}

func TestAutomationPresenter_PresentGetDetail(t *testing.T) {
	t.Parallel()
	rule := &api.AutomationRule{
		UUID:        "uuid-123",
		Name:        "My Rule",
		State:       "ENABLED",
		Description: "Does stuff",
		Components: []api.RuleComponent{
			{Component: "TRIGGER", Type: "issue.created"},
			{Component: "ACTION", Type: "assign.issue"},
		},
	}

	model := AutomationPresenter{}.PresentGetDetail(rule, false)

	// Header + State + Components + Description = 4 message sections
	if len(model.Sections) != 4 {
		t.Fatalf("expected 4 sections, got %d", len(model.Sections))
	}

	header := model.Sections[0].(*present.MessageSection)
	if header.Message != "uuid-123  My Rule" {
		t.Errorf("header = %q", header.Message)
	}

	state := model.Sections[1].(*present.MessageSection)
	if state.Message != "State: ENABLED" {
		t.Errorf("state = %q", state.Message)
	}
}

func TestAutomationPresenter_PresentGetDetail_ShowComponents(t *testing.T) {
	t.Parallel()
	rule := &api.AutomationRule{
		UUID:  "uuid-123",
		Name:  "My Rule",
		State: "ENABLED",
		Components: []api.RuleComponent{
			{Component: "TRIGGER", Type: "issue.created"},
			{Component: "ACTION", Type: "assign.issue"},
		},
	}

	model := AutomationPresenter{}.PresentGetDetail(rule, true)

	// Header + State + Components + component tree = 4
	if len(model.Sections) != 4 {
		t.Fatalf("expected 4 sections (3 msg + tree), got %d", len(model.Sections))
	}

	tree, ok := model.Sections[3].(*present.MessageSection)
	if !ok {
		t.Fatalf("expected MessageSection at [3], got %T", model.Sections[3])
	}
	if !strings.Contains(tree.Message, "TRIGGER  issue.created") {
		t.Errorf("expected trigger line, got %q", tree.Message)
	}
	if !strings.Contains(tree.Message, "  ACTION  assign.issue") {
		t.Errorf("expected indented action line, got %q", tree.Message)
	}
}

func TestComponentTree_NestedChildren(t *testing.T) {
	t.Parallel()
	trigger := &api.RuleComponent{Component: "TRIGGER", Type: "issue.created"}
	components := []api.RuleComponent{
		{
			Component: "CONDITION",
			Type:      "jira.condition.container.block",
			Children:  json.RawMessage(`[{"component":"CONDITION_BLOCK","type":"jira.condition.if.block","children":[{"component":"ACTION","type":"jira.issue.create"}]}]`),
		},
		{Component: "ACTION", Type: "assign.issue"},
	}

	rule := &api.AutomationRule{
		UUID:       "uuid-nested",
		Name:       "Nested",
		State:      "ENABLED",
		Trigger:    trigger,
		Components: components,
	}

	model := AutomationPresenter{}.PresentGetDetail(rule, true)
	tree := model.Sections[3].(*present.MessageSection)

	if !strings.Contains(tree.Message, "TRIGGER  issue.created") {
		t.Errorf("missing trigger, got %q", tree.Message)
	}
	if !strings.Contains(tree.Message, "  CONDITION  jira.condition.container.block") {
		t.Errorf("missing condition, got %q", tree.Message)
	}
	if !strings.Contains(tree.Message, "    CONDITION_BLOCK  jira.condition.if.block") {
		t.Errorf("missing nested condition block, got %q", tree.Message)
	}
	if !strings.Contains(tree.Message, "      ACTION  jira.issue.create") {
		t.Errorf("missing deeply nested action, got %q", tree.Message)
	}
	if !strings.Contains(tree.Message, "  ACTION  assign.issue") {
		t.Errorf("missing top-level action, got %q", tree.Message)
	}
}

func TestComponentTree_TriggerDeduplication(t *testing.T) {
	t.Parallel()
	trigger := &api.RuleComponent{Component: "TRIGGER", Type: "issue.created"}
	components := []api.RuleComponent{
		{Component: "TRIGGER", Type: "issue.created"},
		{Component: "ACTION", Type: "assign.issue"},
	}

	rule := &api.AutomationRule{
		UUID:       "uuid-dedup",
		Name:       "Dedup",
		State:      "ENABLED",
		Trigger:    trigger,
		Components: components,
	}

	model := AutomationPresenter{}.PresentGetDetail(rule, true)
	tree := model.Sections[3].(*present.MessageSection)

	count := strings.Count(tree.Message, "TRIGGER  issue.created")
	if count != 1 {
		t.Errorf("expected 1 TRIGGER line (deduplication), got %d in %q", count, tree.Message)
	}
}

func TestAutomationPresenter_PresentGetDetailExtended(t *testing.T) {
	t.Parallel()
	rule := &api.AutomationRule{
		UUID:        "uuid-123",
		Name:        "My Rule",
		State:       "ENABLED",
		Description: "Does stuff",
		Labels:      []string{"onboarding"},
		Tags:        []string{"auto-create"},
		Created:     &atime.AtlassianTime{Time: time.Date(2023, 12, 4, 10, 0, 0, 0, time.UTC)},
		Updated:     &atime.AtlassianTime{Time: time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)},
		Projects: []api.RuleProject{
			{ProjectKey: "MON"},
			{ProjectKey: "ON"},
		},
		Components: []api.RuleComponent{
			{Component: "TRIGGER", Type: "issue.created"},
		},
	}

	model := AutomationPresenter{}.PresentGetDetailExtended(rule, false, "Rian Stockbower")

	// Header + State + Components + Description + Labels + Tags + Author + Scope + Created/Updated = 9
	if len(model.Sections) != 9 {
		t.Fatalf("expected 9 sections, got %d", len(model.Sections))
	}

	// Check scope
	scope := model.Sections[7].(*present.MessageSection)
	if scope.Message != "Scope: project (MON, ON)" {
		t.Errorf("scope = %q", scope.Message)
	}

	// Check timestamps
	timestamps := model.Sections[8].(*present.MessageSection)
	if timestamps.Message != "Created: 2023-12-04   Updated: 2026-03-15" {
		t.Errorf("timestamps = %q", timestamps.Message)
	}
}

func TestAutomationScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rule *api.AutomationRule
		want string
	}{
		{
			name: "projects",
			rule: &api.AutomationRule{Projects: []api.RuleProject{{ProjectKey: "MON"}, {ProjectKey: "ON"}}},
			want: "project (MON, ON)",
		},
		{
			name: "ARIs",
			rule: &api.AutomationRule{RuleScopeARIs: []string{"ari:cloud:jira::site/123"}},
			want: "scoped",
		},
		{
			name: "global",
			rule: &api.AutomationRule{},
			want: "global",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := automationScope(tt.rule)
			if got != tt.want {
				t.Errorf("automationScope() = %q, want %q", got, tt.want)
			}
		})
	}
}
