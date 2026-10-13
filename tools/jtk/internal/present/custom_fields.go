package present

import (
	"context"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cache"
)

// AppendIssueCustomFields appends the "Custom Fields:" section for issue to
// model, resolving field names from the field cache (fetched when missing or
// stale). It backs --custom-fields on every command that renders an issue
// detail view. When field metadata cannot be loaded the section is left out
// and the rest of the view still renders.
func AppendIssueCustomFields(ctx context.Context, client *api.Client, issue *api.Issue, model *present.OutputModel) {
	fields, err := cache.GetFieldsCacheFirst(ctx, client)
	if err != nil {
		return
	}
	AppendCustomFieldsSection(model, api.ExtractIssueFieldValues(issue, fields))
}
