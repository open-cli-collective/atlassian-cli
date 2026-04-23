package issues

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/mutation"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
	"github.com/open-cli-collective/jira-ticket-cli/internal/text"
)

func newUpdateCmd(opts *root.Options) *cobra.Command {
	var summary string
	var description string
	var parent string
	var assignee string
	var issueType string
	var fields []string

	cmd := &cobra.Command{
		Use:   "update <issue-key>",
		Short: "Update an issue",
		Long: `Update fields on an existing Jira issue.

To change the issue type, use --type. This uses the Jira Cloud bulk move API
transparently (since the standard update API does not support type changes).`,
		Example: `  # Update summary
  jtk issues update PROJ-123 --summary "New summary"

  # Update description
  jtk issues update PROJ-123 --description "Updated description"

  # Change issue type
  jtk issues update PROJ-123 --type Story

  # Move issue under a different parent/epic
  jtk issues update PROJ-123 --parent PROJ-100

  # Reassign an issue
  jtk issues update PROJ-123 --assignee user@example.com

  # Unassign an issue
  jtk issues update PROJ-123 --assignee none

  # Update custom fields
  jtk issues update PROJ-123 --field priority=High --field "Story Points"=5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context(), opts, args[0], summary, description, parent, assignee, issueType, fields)
		},
	}

	cmd.Flags().StringVarP(&summary, "summary", "s", "", "New summary")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New description")
	cmd.Flags().StringVar(&parent, "parent", "", "Parent issue key (epic or parent issue)")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Assignee (account ID, email, or \"me\")")
	cmd.Flags().StringVarP(&issueType, "type", "t", "", "New issue type (uses bulk move API)")
	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, "Fields to update (key=value)")

	return cmd
}

func runUpdate(ctx context.Context, opts *root.Options, issueKey, summary, description, parent, assignee, issueType string, fieldArgs []string) error {
	// Validate that at least one field is being updated before making API calls
	if summary == "" && description == "" && parent == "" && assignee == "" && issueType == "" && len(fieldArgs) == 0 {
		return fmt.Errorf("no fields specified to update")
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Handle type change via the move API
	if issueType != "" {
		if err := changeIssueType(ctx, client, opts, issueKey, issueType); err != nil {
			return err
		}
	}

	// Handle other field updates via the standard update API
	fields := make(map[string]any)

	if summary != "" {
		fields["summary"] = summary
	}

	if description != "" {
		fields["description"] = api.NewADFDocument(text.InterpretEscapes(description))
	}

	if parent != "" {
		fields["parent"] = map[string]string{"key": parent}
	}

	if assignee != "" {
		if api.IsNullValue(assignee) {
			fields["assignee"] = nil
		} else {
			resolvedUser, err := resolve.New(client).User(ctx, assignee)
			if err != nil {
				return err
			}
			fields["assignee"] = map[string]string{"accountId": resolvedUser.AccountID}
		}
	}

	// Parse additional fields
	if len(fieldArgs) > 0 {
		allFields, err := client.GetFields(ctx)
		if err != nil {
			return fmt.Errorf("getting field metadata: %w", err)
		}

		for _, f := range fieldArgs {
			parts := strings.SplitN(f, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid field format: %s (expected key=value)", f)
			}

			key, value := parts[0], parts[1]

			// Try to resolve field name to ID and get field info
			var fieldID string
			var field *api.Field
			if resolved := api.FindFieldByName(allFields, key); resolved != nil {
				fieldID = resolved.ID
				field = resolved
			} else if resolved := api.FindFieldByID(allFields, key); resolved != nil {
				fieldID = resolved.ID
				field = resolved
			} else {
				fieldID = key
			}

			// Format value based on field type, merging with existing if same key repeated
			formatted := api.FormatFieldValue(field, value)
			if existing, ok := fields[fieldID]; ok {
				fields[fieldID] = api.MergeFieldValues(existing, formatted)
			} else {
				fields[fieldID] = formatted
			}
		}
	}

	// If only --type was specified with no other field changes, still show
	// post-state via the fetch-after-write path below.
	var req *api.UpdateIssueRequest
	if len(fields) > 0 {
		req = api.BuildUpdateRequest(fields)
	}

	if opts.EmitIDOnly() {
		if req != nil {
			if err := client.UpdateIssue(ctx, issueKey, req); err != nil {
				return err
			}
		}
		return jtkpresent.EmitIDs(opts, []string{issueKey})
	}

	return mutation.WriteAndPresent(ctx, opts, mutation.Config{
		Write: func(ctx context.Context) (string, error) {
			if req != nil {
				if err := client.UpdateIssue(ctx, issueKey, req); err != nil {
					return "", err
				}
			}
			return issueKey, nil
		},
		Fetch: func(ctx context.Context, id string) (*present.OutputModel, error) {
			issue, err := client.GetIssue(ctx, id)
			if err != nil {
				return nil, err
			}
			return jtkpresent.IssuePresenter{}.PresentDetail(
				issue, client.IssueURL(id), opts.IsExtended(), opts.IsFullText(),
			), nil
		},
		Fallback: func(id string) *present.OutputModel {
			return jtkpresent.IssuePresenter{}.PresentUpdated(id)
		},
	})
}

// changeIssueType performs the type change via the bulk move API.
// It emits progress advisories on stderr but does NOT emit any success
// output to stdout — the caller is responsible for showing post-state
// via the fetch-after-write path.
func changeIssueType(ctx context.Context, client *api.Client, opts *root.Options, issueKey, targetTypeName string) error {
	issue, err := client.GetIssue(ctx, issueKey)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	if issue.Fields.Project == nil {
		return fmt.Errorf("issue %s has no project information", issueKey)
	}
	projectKey := issue.Fields.Project.Key

	if issue.Fields.IssueType != nil && strings.EqualFold(issue.Fields.IssueType.Name, targetTypeName) {
		advisory := jtkpresent.MutationPresenter{}.Advisory("type is already %s", targetTypeName)
		advOut := present.Render(advisory, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stderr, advOut.Stderr)
		return nil
	}

	resolvedType, err := resolve.New(client).IssueType(ctx, projectKey, targetTypeName)
	if err != nil {
		return err
	}
	targetIssueType := &resolvedType

	advisory := jtkpresent.IssuePresenter{}.PresentTypeChangeProgress(issueKey, targetIssueType.Name)
	advOut := present.Render(advisory, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stderr, advOut.Stderr)

	req := api.BuildMoveRequest([]string{issueKey}, projectKey, targetIssueType.ID, false)

	resp, err := client.MoveIssues(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("type change failed - this feature requires Jira Cloud")
		}
		return fmt.Errorf("failed to change issue type: %w", err)
	}

	for {
		status, err := client.GetMoveTaskStatus(ctx, resp.TaskID)
		if err != nil {
			return fmt.Errorf("failed to get task status: %w", err)
		}

		switch status.Status {
		case "COMPLETE":
			if status.Result != nil && len(status.Result.Failed) > 0 {
				for _, failed := range status.Result.Failed {
					return fmt.Errorf("type change failed for %s: %s", failed.IssueKey, strings.Join(failed.Errors, ", "))
				}
			}
			return nil

		case "FAILED":
			return fmt.Errorf("type change failed")

		case "CANCELLED":
			return fmt.Errorf("type change was cancelled")

		case "ENQUEUED", "RUNNING":
			time.Sleep(1 * time.Second)

		default:
			return fmt.Errorf("unknown task status: %s", status.Status)
		}
	}
}
