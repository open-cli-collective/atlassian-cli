// Package links provides CLI commands for managing Jira issue links.
package links

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
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
		Example: `  # A blocks B
  jtk links create PROJ-123 PROJ-456 --type Blocks

  # A relates to B
  jtk links create PROJ-123 PROJ-456 --type Relates

  # A is cloned by B
  jtk links create PROJ-123 PROJ-456 --type Cloners`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runCreate(opts, args[0], args[1], linkType)
		},
	}

	cmd.Flags().StringVarP(&linkType, "type", "t", "", "Link type name (required)")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}

func runCreate(opts *root.Options, outwardKey, inwardKey, linkType string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Validate link type exists
	linkTypes, err := client.GetIssueLinkTypes()
	if err != nil {
		return fmt.Errorf("failed to get link types: %w", err)
	}

	var found bool
	for _, lt := range linkTypes {
		if strings.EqualFold(lt.Name, linkType) {
			linkType = lt.Name // Use exact casing from server
			found = true
			break
		}
	}

	if !found {
		var available []string
		for _, lt := range linkTypes {
			available = append(available, lt.Name)
		}
		return fmt.Errorf("link type %q not found (available: %s)", linkType, strings.Join(available, ", "))
	}

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
