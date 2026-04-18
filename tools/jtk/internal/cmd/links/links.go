// Package links provides CLI commands for managing Jira issue links.
package links

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/resolve"
)

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
	cmd := &cobra.Command{
		Use:     "list <issue-key>",
		Short:   "List links on an issue",
		Long:    "List all links on a specific issue.",
		Example: `  jtk links list PROJ-123`,
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runList(opts, args[0])
		},
	}

	return cmd
}

func runList(opts *root.Options, issueKey string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	links, err := client.GetIssueLinks(issueKey)
	if err != nil {
		return err
	}

	if len(links) == 0 {
		model := jtkpresent.LinkPresenter{}.PresentEmpty(issueKey)
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	if v.Format == view.FormatJSON {
		return v.JSON(links)
	}

	// Text path: presenter → render → write
	model := jtkpresent.LinkPresenter{}.PresentList(links)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
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

	model := jtkpresent.LinkPresenter{}.PresentCreated(linkType, outwardKey, inwardKey)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
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

	model := jtkpresent.LinkPresenter{}.PresentDeleted(linkID)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
}

func newTypesCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "types",
		Short:   "List available link types",
		Long:    "List all available issue link types in the Jira instance.",
		Example: `  jtk links types`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTypes(opts)
		},
	}

	return cmd
}

func runTypes(opts *root.Options) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	linkTypes, err := client.GetIssueLinkTypes()
	if err != nil {
		return err
	}

	if len(linkTypes) == 0 {
		model := jtkpresent.LinkPresenter{}.PresentNoTypes()
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	if v.Format == view.FormatJSON {
		return v.JSON(linkTypes)
	}

	// Text path: presenter → render → write
	model := jtkpresent.LinkPresenter{}.PresentTypes(linkTypes)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

// GetIssueLinkTypes returns all link types (exported for use by other commands)
func GetIssueLinkTypes(client *api.Client) ([]api.IssueLinkType, error) {
	return client.GetIssueLinkTypes()
}
