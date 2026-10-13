package present

import (
	"context"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
)

// AppendIssueCustomFields appends the "Custom Fields:" section for issue to
// model, labeling values with names from fetchFields. It backs
// --custom-fields on every command that renders an issue detail view.
// fetchFields has the same shape projection.Resolve takes, so the command
// owns where field metadata comes from. When it fails the section is left
// out and the rest of the view still renders.
func AppendIssueCustomFields(ctx context.Context, fetchFields func(context.Context) ([]api.Field, error), issue *api.Issue, model *present.OutputModel) {
	fields, err := fetchFields(ctx)
	if err != nil {
		return
	}
	AppendCustomFieldsSection(model, api.ExtractIssueFieldValues(issue, fields))
}
