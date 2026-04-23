// Package links provides CLI commands for managing Jira issue links.
package links

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
)

func noFieldFetch(_ context.Context) ([]api.Field, error) { return nil, nil }

// Register registers the links commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "links",
		Aliases: []string{"link", "l"},
		Short:   "Manage issue links",
		Long:    "Commands for listing, creating, and deleting issue links.",
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newCreateCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))
	cmd.AddCommand(newTypesCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "list <issue-key>",
		Short: "List links on an issue",
		Long:  "List all links on a specific issue.",
		Example: `  jtk links list PROJ-123
  jtk links list PROJ-123 --extended
  jtk links list PROJ-123 --id
  jtk links list PROJ-123 --fields TYPE,ISSUE`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts, args[0], fieldsFlag)
		},
	}

	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display columns")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, issueKey, fieldsFlag string) error {
	v := opts.View()
	idOnly := opts.EmitIDOnly()

	if !idOnly && fieldsFlag != "" && v.Format == view.FormatJSON {
		return jtkpresent.ErrFieldsWithJSON
	}

	var selected []projection.ColumnSpec
	var projected bool
	if !idOnly {
		var err error
		selected, projected, err = projection.Resolve(
			ctx,
			jtkpresent.LinkListSpec,
			opts.IsExtended(),
			fieldsFlag,
			noFieldFetch,
			"links list",
		)
		if err != nil {
			return err
		}
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	links, err := client.GetIssueLinks(issueKey)
	if err != nil {
		return err
	}

	if idOnly {
		ids := make([]string, len(links))
		for i, l := range links {
			ids[i] = l.ID
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	if len(links) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.LinkPresenter{}.PresentEmpty(issueKey))
	}

	if v.Format == view.FormatJSON {
		return v.JSON(links)
	}

	model := jtkpresent.LinkPresenter{}.PresentList(links, opts.IsExtended())
	if projected {
		projection.ApplyToTableInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
}

func newCreateCmd(opts *root.Options) *cobra.Command {
	var linkType string

	cmd := &cobra.Command{
		Use:   "create <issue-key> <target-issue-key>",
		Short: "Create a link between two issues",
		Long: `Create a link between two issues.

The first issue is the outward issue and the second is the inward issue.
For example, "jtk links create A B --type Blocks" means "A blocks B".`,
		Example: `  # --type accepts the canonical name, the outward verb, or the inward verb.
  # With an inward verb the issue-key ordering is interpreted from the user's
  # perspective: ` + "`" + `A is blocked by B` + "`" + ` creates B → blocks → A.
  jtk links create PROJ-123 PROJ-456 --type Blocker
  jtk links create PROJ-123 PROJ-456 --type blocks            # A blocks B
  jtk links create PROJ-123 PROJ-456 --type "is blocked by"   # A is blocked by B

  # A relates to B
  jtk links create PROJ-123 PROJ-456 --type Relates

  # A is cloned by B
  jtk links create PROJ-123 PROJ-456 --type "is cloned by"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd.Context(), opts, args[0], args[1], linkType)
		},
	}

	cmd.Flags().StringVarP(&linkType, "type", "t", "", "Link type: canonical name, outward verb, or inward verb (required)")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}

func runCreate(ctx context.Context, opts *root.Options, outwardKey, inwardKey, linkType string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	resolvedLinkType, err := resolve.New(client).LinkType(ctx, linkType)
	if err != nil {
		return err
	}

	// A cold-start synthetic has Name=input and empty Inward/Outward/ID.
	// Without the verbs, we can't tell whether the user typed a canonical
	// name or a directional verb — creating the link anyway would either
	// silently reverse the direction (inward verb typed) or fail at the
	// API (unknown type). Refuse up front with a concrete remediation.
	if resolvedLinkType.ID == "" && resolvedLinkType.Inward == "" && resolvedLinkType.Outward == "" {
		return fmt.Errorf(
			"cannot resolve link type %q from cache — "+
				"run `jtk refresh linktypes` to load verbs and IDs, "+
				"or pass the canonical link type name once refreshed",
			linkType)
	}

	// If the user typed the inward verb ("is blocked by"), the positional
	// arg order reads <inward> <outward> from their perspective. Swap so
	// the resulting link matches the verb they chose. Input matching the
	// canonical name or the outward verb maps to outward→inward as given.
	if strings.EqualFold(linkType, resolvedLinkType.Inward) &&
		!strings.EqualFold(linkType, resolvedLinkType.Outward) &&
		!strings.EqualFold(linkType, resolvedLinkType.Name) {
		outwardKey, inwardKey = inwardKey, outwardKey
	}
	linkType = resolvedLinkType.Name

	if err := client.CreateIssueLink(outwardKey, inwardKey, linkType); err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.JSON(map[string]string{
			"status":       "created",
			"outwardIssue": outwardKey,
			"inwardIssue":  inwardKey,
			"type":         linkType,
		})
	}

	return jtkpresent.Emit(opts, jtkpresent.LinkPresenter{}.PresentCreated(linkType, outwardKey, inwardKey))
}

func newDeleteCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <link-id>",
		Short: "Delete an issue link",
		Long:  "Delete an issue link by its ID. Use 'jtk links list' to find link IDs.",
		Example: `  jtk links delete 10001
  jtk links list PROJ-123   # find link IDs first`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDelete(opts, args[0])
		},
	}

	return cmd
}

func runDelete(opts *root.Options, linkID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteIssueLink(linkID); err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.JSON(map[string]string{"status": "deleted", "linkId": linkID})
	}

	return jtkpresent.Emit(opts, jtkpresent.LinkPresenter{}.PresentDeleted(linkID))
}

func newTypesCmd(opts *root.Options) *cobra.Command {
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "types",
		Short: "List available link types",
		Long:  "List all available issue link types in the Jira instance.",
		Example: `  jtk links types
  jtk links types --id`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTypes(cmd.Context(), opts, fieldsFlag)
		},
	}

	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display columns")

	return cmd
}

func runTypes(ctx context.Context, opts *root.Options, fieldsFlag string) error {
	v := opts.View()
	idOnly := opts.EmitIDOnly()

	if !idOnly && fieldsFlag != "" && v.Format == view.FormatJSON {
		return jtkpresent.ErrFieldsWithJSON
	}

	var selected []projection.ColumnSpec
	var projected bool
	if !idOnly {
		var err error
		selected, projected, err = projection.Resolve(
			ctx,
			jtkpresent.LinkTypesSpec,
			opts.IsExtended(),
			fieldsFlag,
			noFieldFetch,
			"links types",
		)
		if err != nil {
			return err
		}
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	linkTypes, err := client.GetIssueLinkTypes()
	if err != nil {
		return err
	}

	if len(linkTypes) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.LinkPresenter{}.PresentNoTypes())
	}

	if idOnly {
		ids := make([]string, len(linkTypes))
		for i, t := range linkTypes {
			ids[i] = t.ID
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	if v.Format == view.FormatJSON {
		return v.JSON(linkTypes)
	}

	model := jtkpresent.LinkPresenter{}.PresentTypes(linkTypes)
	if projected {
		projection.ApplyToTableInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
}

// GetIssueLinkTypes returns all link types (exported for use by other commands)
func GetIssueLinkTypes(client *api.Client) ([]api.IssueLinkType, error) {
	return client.GetIssueLinkTypes()
}
