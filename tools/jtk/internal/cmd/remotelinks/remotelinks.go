// Package remotelinks provides CLI commands for managing Jira issue remote
// (web) links — external URLs shown in an issue's links sidebar.
package remotelinks

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
	"github.com/open-cli-collective/jira-ticket-cli/internal/text"
)

// noFieldFetch is the projection.Resolve fetcher for remote links. Remote
// link fields are not Jira issue fields, so there is no metadata to fetch;
// returning nil routes deferred tokens cleanly to UnknownFieldError rather
// than a real network call against /rest/api/3/field.
func noFieldFetch(_ context.Context) ([]api.Field, error) { return nil, nil }

// Register registers the remotelinks commands.
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "remotelinks",
		Aliases: []string{"remotelink", "rl"},
		Short:   "Manage issue remote (web) links",
		Long:    "Commands for listing, adding, and deleting an issue's remote (web) links — external URLs shown in the Jira issue links sidebar.",
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newAddCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "list <issue-key>",
		Short: "List remote links on an issue",
		Long:  "List all remote (web) links on a specific issue.",
		Example: `  jtk remotelinks list PROJ-123
  jtk remotelinks list PROJ-123 --extended
  jtk remotelinks list PROJ-123 --id
  jtk remotelinks list PROJ-123 --fields TITLE,URL`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts, args[0], fieldsFlag)
		},
	}

	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display columns")

	return cmd
}

func runList(ctx context.Context, opts *root.Options, issueKey, fieldsFlag string) error {
	idOnly := opts.EmitIDOnly()

	var selected []projection.ColumnSpec
	var projected bool
	if !idOnly {
		var err error
		selected, projected, err = projection.Resolve(
			ctx,
			jtkpresent.RemoteLinkListSpec,
			opts.IsExtended(),
			fieldsFlag,
			noFieldFetch,
			"remotelinks list",
		)
		if err != nil {
			return err
		}
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	links, err := client.GetRemoteLinks(ctx, issueKey)
	if err != nil {
		return err
	}

	if idOnly {
		ids := make([]string, len(links))
		for i, l := range links {
			ids[i] = strconv.Itoa(l.ID)
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	if len(links) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.RemoteLinkPresenter{}.PresentEmpty(issueKey))
	}

	model := jtkpresent.RemoteLinkPresenter{}.PresentList(links, opts.IsExtended())
	if projected {
		projection.ApplyToTableInModel(model, selected)
	}
	return jtkpresent.Emit(opts, model)
}

func newAddCmd(opts *root.Options) *cobra.Command {
	var url, title, summary, relationship string

	cmd := &cobra.Command{
		Use:   "add <issue-key>",
		Short: "Add a remote (web) link to an issue",
		Long:  "Add a remote (web) link to an issue, pointing at an external URL such as a GitHub issue or a documentation page.",
		Example: `  jtk remotelinks add PROJ-123 --url "https://github.com/owner/repo/issues/456" --title "GitHub #456: Some issue"
  jtk remotelinks add PROJ-123 --url "https://example.com" --title "Docs" --summary "Reference docs"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd.Context(), opts, args[0], url, title, summary, relationship)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "External URL the link points at (required)")
	cmd.Flags().StringVar(&title, "title", "", "Display title for the link (defaults to the URL)")
	cmd.Flags().StringVar(&summary, "summary", "", "Optional one-line summary shown under the title")
	cmd.Flags().StringVar(&relationship, "relationship", "", "Optional relationship label (e.g. \"mentioned in\")")
	_ = cmd.MarkFlagRequired("url")

	return cmd
}

func runAdd(ctx context.Context, opts *root.Options, issueKey, url, title, summary, relationship string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// Title defaults to the URL so the link is never anonymous in the sidebar.
	if title == "" {
		title = url
	}

	req := api.CreateRemoteLinkRequest{
		Relationship: text.InterpretEscapes(relationship),
		Object: api.RemoteLinkObject{
			URL:     url,
			Title:   text.InterpretEscapes(title),
			Summary: text.InterpretEscapes(summary),
		},
	}

	link, err := client.AddRemoteLink(ctx, issueKey, req)
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{strconv.Itoa(link.ID)})
	}

	return jtkpresent.Emit(opts, jtkpresent.RemoteLinkPresenter{}.PresentAddedDetail(issueKey, link))
}

func newDeleteCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <issue-key> <link-id>",
		Short: "Delete a remote (web) link from an issue",
		Long:  "Delete a remote (web) link from an issue by its ID. Use 'jtk remotelinks list' to find link IDs.",
		Example: `  jtk remotelinks delete PROJ-123 12345
  jtk remotelinks list PROJ-123   # find link IDs first`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.Context(), opts, args[0], args[1])
		},
	}

	return cmd
}

func runDelete(ctx context.Context, opts *root.Options, issueKey, linkIDArg string) error {
	// Remote link IDs are integers; reject typos before the API call so the
	// user gets a clear message instead of a server-side 404.
	linkID, err := strconv.Atoi(linkIDArg)
	if err != nil {
		return fmt.Errorf("invalid link ID %q: must be a number", linkIDArg)
	}

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteRemoteLink(ctx, issueKey, linkID); err != nil {
		return err
	}

	return jtkpresent.Emit(opts, jtkpresent.RemoteLinkPresenter{}.PresentDeleted(linkID, issueKey))
}
