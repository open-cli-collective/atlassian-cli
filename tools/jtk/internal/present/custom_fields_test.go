package present

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func renderMessages(m *present.OutputModel) string {
	var b strings.Builder
	for _, s := range m.Sections {
		if msg, ok := s.(*present.MessageSection); ok {
			b.WriteString(msg.Message)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func TestAppendIssueCustomFields(t *testing.T) {
	issue := &api.Issue{Key: "TEST-1", Fields: api.IssueFields{CustomFields: map[string]any{
		"customfield_10005": map[string]any{"value": "Feature"},
		"customfield_10099": nil,
	}}}
	fields := []api.Field{{ID: "customfield_10005", Name: "Change type", Custom: true}}

	t.Run("appends named non-empty fields", func(t *testing.T) {
		model := &present.OutputModel{}
		AppendIssueCustomFields(context.Background(), func(context.Context) ([]api.Field, error) {
			return fields, nil
		}, issue, model)
		out := renderMessages(model)
		if !strings.Contains(out, "Custom Fields:") || !strings.Contains(out, "  Change type: Feature") {
			t.Fatalf("missing custom fields section:\n%s", out)
		}
		if strings.Contains(out, "customfield_10099") {
			t.Fatalf("empty field rendered:\n%s", out)
		}
	})

	t.Run("omits the section when field metadata fails", func(t *testing.T) {
		model := &present.OutputModel{}
		AppendIssueCustomFields(context.Background(), func(context.Context) ([]api.Field, error) {
			return nil, errors.New("offline")
		}, issue, model)
		if len(model.Sections) != 0 {
			t.Fatalf("expected no sections, got %d", len(model.Sections))
		}
	})
}
