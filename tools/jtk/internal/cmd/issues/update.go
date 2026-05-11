package issues

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	sharederrors "github.com/open-cli-collective/atlassian-go/errors"
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
	var status string
	var fields []string

	cmd := &cobra.Command{
		Use:   "update <issue-key>",
		Short: "Update an issue",
		Long: `Update fields on an existing Jira issue.

To change the issue type, use --type. This uses the Jira Cloud bulk move API
transparently (since the standard update API does not support type changes).

To change the workflow status, use --status. This resolves the matching
transition against the issue's current workflow and POSTs to the transitions
endpoint. If multiple transitions land on the same target status, run
` + "`jtk transitions do <key> <id>`" + ` instead. ` + "`--status`" + ` is resolved before any
writes; condition-based transitions that become available only after a
preceding field edit must be performed as a separate command.`,
		Example: `  # Update summary
  jtk issues update PROJ-123 --summary "New summary"

  # Update description
  jtk issues update PROJ-123 --description "Updated description"

  # Change issue type
  jtk issues update PROJ-123 --type Story

  # Change workflow status (quote multi-word names: --status "In Progress")
  jtk issues update PROJ-123 --status "Done"

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
			return runUpdate(cmd.Context(), opts, args[0], summary, description, parent, assignee, issueType, status, fields)
		},
	}

	cmd.Flags().StringVarP(&summary, "summary", "s", "", "New summary")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New description")
	cmd.Flags().StringVar(&parent, "parent", "", "Parent issue key (epic or parent issue)")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Assignee (account ID, email, or \"me\")")
	cmd.Flags().StringVarP(&issueType, "type", "t", "", "New issue type (uses bulk move API)")
	cmd.Flags().StringVar(&status, "status", "", "New workflow status (uses transitions API)")
	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, "Fields to update (key=value)")

	return cmd
}

// statusChange describes the result of preflight-resolving a --status request.
// It is only populated when --status was requested; an empty value should not
// be interpreted as a noop.
type statusChange struct {
	isNoop       bool   // current status already equals requested
	transitionID string // populated when isNoop is false and a transition was resolved
	targetStatus string // resolved To.Name; used by mutation.ModelContainsStatus
}

func runUpdate(ctx context.Context, opts *root.Options, issueKey, summary, description, parent, assignee, issueType, status string, fieldArgs []string) error {
	// Validate that at least one field is being updated before making API calls
	if summary == "" && description == "" && parent == "" && assignee == "" && issueType == "" && status == "" && len(fieldArgs) == 0 {
		return fmt.Errorf("no fields specified to update")
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Preflight: resolve --status against the issue's *current* workflow before
	// any writes. This lets us reject ambiguous/invalid status names without
	// having partially mutated the issue.
	var sc statusChange
	if status != "" {
		sc, err = resolveStatusChange(ctx, client, opts, issueKey, status)
		if err != nil {
			return err
		}
	}

	// Handle type change via the move API
	if issueType != "" {
		if err := changeIssueType(ctx, client, opts, issueKey, issueType); err != nil {
			if errors.Is(err, errTypeChangeUnverified) {
				_, _ = fmt.Fprintf(opts.Stderr, "Type change accepted but status could not be verified\n")
			} else {
				return err
			}
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
			fieldID, field, value, err := api.ResolveFieldArg(allFields, f)
			if err != nil {
				return err
			}

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

	// Status-only no-op short-circuit: nothing else to write, status already
	// matches. In ID-only mode we still emit the key (machine-parseable
	// output); otherwise emit a stderr advisory.
	if status != "" && sc.isNoop && req == nil && issueType == "" {
		if opts.EmitIDOnly() {
			return jtkpresent.EmitIDs(opts, []string{issueKey})
		}
		return jtkpresent.Emit(opts, jtkpresent.IssuePresenter{}.PresentStatusAlreadyCurrent(sc.targetStatus))
	}

	doTransition := func(ctx context.Context) error {
		if status == "" || sc.isNoop {
			return nil
		}
		return client.DoTransition(ctx, issueKey, sc.transitionID, nil)
	}

	if opts.EmitIDOnly() {
		if req != nil {
			if err := client.UpdateIssue(ctx, issueKey, req); err != nil {
				return err
			}
		}
		if err := doTransition(ctx); err != nil {
			return err
		}
		return jtkpresent.EmitIDs(opts, []string{issueKey})
	}

	var isFresh func(*present.OutputModel) bool
	if status != "" && !sc.isNoop {
		target := sc.targetStatus
		isFresh = func(m *present.OutputModel) bool {
			return mutation.ModelContainsStatus(m, target)
		}
	}

	return mutation.WriteAndPresent(ctx, opts, mutation.Config{
		Write: func(ctx context.Context) (string, error) {
			if req != nil {
				if err := client.UpdateIssue(ctx, issueKey, req); err != nil {
					return "", err
				}
			}
			if err := doTransition(ctx); err != nil {
				return "", err
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
		IsFresh: isFresh,
		Fallback: func(id string) *present.OutputModel {
			return jtkpresent.IssuePresenter{}.PresentUpdated(id)
		},
	})
}

// resolveStatusChange preflights a --status request: it fetches the issue and
// (if a transition is needed) the available transitions, then classifies the
// outcome. On user-facing errors (no matching transition, ambiguous match) it
// emits the error via the presenter and returns root.ErrAlreadyReported so
// the caller does not double-print.
func resolveStatusChange(ctx context.Context, client *api.Client, opts *root.Options, issueKey, status string) (statusChange, error) {
	issue, err := client.GetIssue(ctx, issueKey)
	if err != nil {
		return statusChange{}, fmt.Errorf("failed to get issue: %w", err)
	}
	if issue.Fields.Status == nil {
		return statusChange{}, fmt.Errorf("issue %s has no current status; cannot resolve --status", issueKey)
	}
	if strings.EqualFold(issue.Fields.Status.Name, status) {
		return statusChange{isNoop: true, targetStatus: issue.Fields.Status.Name}, nil
	}

	transitions, err := client.GetTransitions(ctx, issueKey)
	if err != nil {
		return statusChange{}, fmt.Errorf("failed to get transitions: %w", err)
	}

	matches := api.FindTransitionsByStatus(transitions, status)
	switch len(matches) {
	case 0:
		_ = jtkpresent.Emit(opts, jtkpresent.TransitionPresenter{}.PresentStatusNotFound(status, transitions))
		return statusChange{}, root.ErrAlreadyReported
	case 1:
		return statusChange{transitionID: matches[0].ID, targetStatus: matches[0].To.Name}, nil
	default:
		_ = jtkpresent.Emit(opts, jtkpresent.TransitionPresenter{}.PresentStatusAmbiguous(issueKey, status, matches))
		return statusChange{}, root.ErrAlreadyReported
	}
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
		return jtkpresent.Emit(opts, jtkpresent.IssuePresenter{}.PresentTypeAlreadyCurrent(targetTypeName))
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
		if sharederrors.IsNotFound(err) {
			return fmt.Errorf("type change failed - this feature requires Jira Cloud")
		}
		return fmt.Errorf("failed to change issue type: %w", err)
	}

	status, err := pollMoveTask(ctx, client, resp.TaskID)
	if errors.Is(err, errStatusUnavailable) {
		return errTypeChangeUnverified
	}
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

	default:
		return fmt.Errorf("unknown task status: %s", status.Status)
	}
}
