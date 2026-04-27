// Package transitions provides CLI commands for managing Jira issue transitions.
package transitions

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	"github.com/open-cli-collective/jira-ticket-cli/internal/mutation"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

// Register registers the transitions commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "transitions",
		Aliases: []string{"transition", "tr"},
		Short:   "Manage issue transitions",
		Long:    "Commands for viewing and performing workflow transitions on issues.",
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newDoCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	var showFields bool

	cmd := &cobra.Command{
		Use:   "list <issue-key>",
		Short: "List available transitions",
		Long:  "List the available workflow transitions for an issue.",
		Example: `  # List transitions
  jtk transitions list PROJ-123

  # Extended output with status category, screen/conditional info, and required fields
  jtk transitions list PROJ-123 --extended

  # Emit only transition IDs
  jtk transitions list PROJ-123 --id`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts, args[0], showFields || opts.IsExtended())
		},
	}

	cmd.Flags().BoolVar(&showFields, "fields", false, "Show required fields for each transition")
	_ = cmd.Flags().MarkDeprecated("fields", "use --extended instead")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, issueKey string, showFields bool) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	idOnly := opts.EmitIDOnly()
	expandFields := showFields && !idOnly

	transitions, err := client.GetTransitionsWithFields(ctx, issueKey, expandFields)
	if err != nil {
		return err
	}

	if idOnly {
		ids := make([]string, len(transitions))
		for i, t := range transitions {
			ids[i] = t.ID
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	if len(transitions) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.TransitionPresenter{}.PresentEmpty(issueKey))
	}

	v := opts.View()
	if v.Format == "json" {
		return v.JSON(transitions)
	}

	model := jtkpresent.TransitionPresenter{}.PresentList(transitions, showFields)
	return jtkpresent.Emit(opts, model)
}

// getRequiredFields delegates to the presenter implementation (exported for testing).
func getRequiredFields(t api.Transition) string {
	return jtkpresent.GetRequiredFieldsForTransition(t)
}

func newDoCmd(opts *root.Options) *cobra.Command {
	var fields []string

	cmd := &cobra.Command{
		Use:   "do <issue-key> <transition>",
		Short: "Perform a transition",
		Long: `Perform a workflow transition on an issue. The transition can be specified by name or ID.

Some transitions require additional fields to be set. Use --field to provide them.`,
		Example: `  # Transition by name
  jtk transitions do PROJ-123 "In Progress"

  # Transition by ID
  jtk transitions do PROJ-123 21

  # Transition with required fields
  jtk transitions do PROJ-123 "In Progress" --field resolution=Done
  jtk transitions do PROJ-123 "Done" --field customfield_10001="some value"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDo(cmd.Context(), opts, args[0], args[1], fields)
		},
	}

	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, "Fields to set during transition (key=value)")

	return cmd
}

func runDo(ctx context.Context, opts *root.Options, issueKey, transitionNameOrID string, fieldArgs []string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	transitions, err := client.GetTransitions(ctx, issueKey)
	if err != nil {
		return err
	}

	var transitionID string

	for _, t := range transitions {
		if t.ID == transitionNameOrID {
			transitionID = t.ID
			break
		}
	}

	if transitionID == "" {
		if t := api.FindTransitionByName(transitions, transitionNameOrID); t != nil {
			transitionID = t.ID
		}
	}

	if transitionID == "" {
		model := jtkpresent.TransitionPresenter{}.PresentNotFound(transitionNameOrID, transitions)
		_ = jtkpresent.Emit(opts, model)
		return fmt.Errorf("transition not found: %s", transitionNameOrID)
	}

	var fields map[string]any
	if len(fieldArgs) > 0 {
		fields = make(map[string]any)

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

			fields[fieldID] = api.FormatFieldValue(field, value)
		}
	}

	if opts.EmitIDOnly() {
		if err := client.DoTransition(ctx, issueKey, transitionID, fields); err != nil {
			return err
		}
		return jtkpresent.EmitIDs(opts, []string{issueKey})
	}

	var targetStatus string
	for _, t := range transitions {
		if t.ID == transitionID {
			targetStatus = t.To.Name
			break
		}
	}

	return mutation.WriteAndPresent(ctx, opts, mutation.Config{
		Write: func(ctx context.Context) (string, error) {
			return issueKey, client.DoTransition(ctx, issueKey, transitionID, fields)
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
		IsFresh: func(m *present.OutputModel) bool {
			if targetStatus == "" {
				return true
			}
			return mutation.ModelContainsStatus(m, targetStatus)
		},
		Fallback: func(id string) *present.OutputModel {
			return jtkpresent.TransitionPresenter{}.PresentTransitioned(id)
		},
	})
}
