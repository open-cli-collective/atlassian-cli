// Package links provides CLI commands for managing Jira issue links.
package links

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
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
		Use:   "list <issue-key>",
		Short: "List links on an issue",
		Long:  "List all links on a specific issue.",
		Example: `  jtk links list PROJ-123
  jtk links list PROJ-123 -o json`,
		Args: cobra.ExactArgs(1),
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
		v.Info("No links on %s", issueKey)
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(links)
	}

	headers := []string{"ID", "TYPE", "DIRECTION", "ISSUE", "SUMMARY"}
	var rows [][]string

	for _, link := range links {
		var direction, key, summary string

		if link.OutwardIssue != nil {
			// OutwardIssue is set → current issue is the inward side
			direction = link.Type.Inward
			key = link.OutwardIssue.Key
			summary = link.OutwardIssue.Fields.Summary
		} else if link.InwardIssue != nil {
			// InwardIssue is set → current issue is the outward side
			direction = link.Type.Outward
			key = link.InwardIssue.Key
			summary = link.InwardIssue.Fields.Summary
		}

		rows = append(rows, []string{
			link.ID,
			link.Type.Name,
			direction,
			key,
			summary,
		})
	}

	return v.Table(headers, rows)
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

	if opts.Output == "json" {
		return v.JSON(map[string]string{
			"status":       "created",
			"outwardIssue": outwardKey,
			"inwardIssue":  inwardKey,
			"type":         linkType,
		})
	}

	v.Success("Created %s link: %s → %s", linkType, outwardKey, inwardKey)
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

	if opts.Output == "json" {
		return v.JSON(map[string]string{"status": "deleted", "linkId": linkID})
	}

	v.Success("Deleted link %s", linkID)
	return nil
}

func newTypesCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "types",
		Short: "List available link types",
		Long:  "List all available issue link types in the Jira instance.",
		Example: `  jtk links types
  jtk links types -o json`,
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
		v.Info("No link types available")
		return nil
	}

	if opts.Output == "json" {
		return v.JSON(linkTypes)
	}

	headers := []string{"ID", "NAME", "OUTWARD", "INWARD"}
	var rows [][]string

	for _, lt := range linkTypes {
		rows = append(rows, []string{
			lt.ID,
			lt.Name,
			lt.Outward,
			lt.Inward,
		})
	}

	return v.Table(headers, rows)
}

// GetIssueLinkTypes returns all link types (exported for use by other commands)
func GetIssueLinkTypes(client *api.Client) ([]api.IssueLinkType, error) {
	return client.GetIssueLinkTypes()
}
