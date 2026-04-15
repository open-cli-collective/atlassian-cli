package automation

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

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
			name: "trigger only",
			components: []api.RuleComponent{
				{Component: "TRIGGER", Type: "jira.issue.create"},
			},
			want: "1 total — 1 trigger(s)",
		},
		{
			name: "all types",
			components: []api.RuleComponent{
				{Component: "TRIGGER", Type: "jira.issue.create"},
				{Component: "CONDITION", Type: "jira.jql.condition"},
				{Component: "ACTION", Type: "jira.issue.assign"},
			},
			want: "3 total — 1 trigger(s), 1 condition(s), 1 action(s)",
		},
		{
			name: "multiple actions",
			components: []api.RuleComponent{
				{Component: "TRIGGER", Type: "jira.issue.create"},
				{Component: "ACTION", Type: "jira.issue.assign"},
				{Component: "ACTION", Type: "jira.issue.transition"},
				{Component: "ACTION", Type: "jira.issue.comment"},
			},
			want: "4 total — 1 trigger(s), 3 action(s)",
		},
		{
			name: "unknown component types ignored in breakdown",
			components: []api.RuleComponent{
				{Component: "TRIGGER", Type: "jira.issue.create"},
				{Component: "BRANCH", Type: "jira.issue.branch"},
			},
			want: "2 total — 1 trigger(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := jtkpresent.SummarizeComponents(tt.components)
			testutil.Equal(t, got, tt.want)
		})
	}
}
