package issues

import (
	"testing"

	"github.com/open-cli-collective/atlassian-go/testutil"
	"github.com/open-cli-collective/jira-ticket-cli/api"
)

func TestResolveFields(t *testing.T) {
	tests := []struct {
		name       string
		fieldsFlag string
		output     string
		full       bool
		want       []string
	}{
		{
			name:       "explicit fields flag takes precedence",
			fieldsFlag: "summary,customfield_10005",
			output:     "json",
			full:       true,
			want:       []string{"summary", "customfield_10005"},
		},
		{
			name:       "json output without fields flag returns all",
			fieldsFlag: "",
			output:     "json",
			full:       false,
			want:       []string{"*all"},
		},
		{
			name:       "json output with full flag still returns all",
			fieldsFlag: "",
			output:     "json",
			full:       true,
			want:       []string{"*all"},
		},
		{
			name:       "full flag returns DefaultSearchFields",
			fieldsFlag: "",
			output:     "",
			full:       true,
			want:       api.DefaultSearchFields,
		},
		{
			name:       "default returns ListSearchFields",
			fieldsFlag: "",
			output:     "",
			full:       false,
			want:       api.ListSearchFields,
		},
		{
			name:       "table output returns ListSearchFields",
			fieldsFlag: "",
			output:     "table",
			full:       false,
			want:       api.ListSearchFields,
		},
		{
			name:       "single field",
			fieldsFlag: "summary",
			output:     "",
			full:       false,
			want:       []string{"summary"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveFields(tt.fieldsFlag, tt.output, tt.full)
			testutil.Equal(t, len(tt.want), len(got))
			for i := range tt.want {
				testutil.Equal(t, tt.want[i], got[i])
			}
		})
	}
}
