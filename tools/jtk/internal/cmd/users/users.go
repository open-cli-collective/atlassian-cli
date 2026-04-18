// Package users provides CLI commands for searching Jira users.
package users

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// noFieldFetch is the projection.Resolve fetcher for user commands. Users are
// not Jira issue fields, so there is no metadata to fetch; returning nil
// routes any deferred tokens cleanly to UnknownFieldError rather than into a
// real /rest/api/3/field call that would surface unrelated issue-field names.
// Package-local rather than shared because it is trivial and typed for the
// caller's context — consolidating it would obscure the intent.
func noFieldFetch(_ context.Context) ([]api.Field, error) { return nil, nil }

// Register registers the users commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "users",
		Aliases: []string{"user", "u"},
		Short:   "Search and lookup users",
		Long:    "Commands for searching and looking up Jira users.",
	}

	cmd.AddCommand(newGetCmd(opts))
	cmd.AddCommand(newSearchCmd(opts))

	parent.AddCommand(cmd)
}

func newGetCmd(opts *root.Options) *cobra.Command {
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "get <account-id>",
		Short: "Get user details by account ID",
		Long:  "Retrieve and display details for a specific user by their Jira account ID.",
		Example: `  # Get user details (pipe one-liner)
  jtk users get 61292e4c4f29230069621c5f

  # Include timezone, locale, and group/application-role counts
  jtk users get 61292e4c4f29230069621c5f --extended

  # Just the account ID
  jtk users get 61292e4c4f29230069621c5f --id

  # Restrict output to selected fields
  jtk users get 61292e4c4f29230069621c5f --fields NAME,EMAIL`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0], fieldsFlag)
		},
	}

	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display fields (UserDetailSpec headers)")

	return cmd
}

func runGet(ctx context.Context, opts *root.Options, accountID, fieldsFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	// --id wins: collapse output before we do any projection work. The
	// account ID is its own canonical identifier (no ID→key lookup step like
	// projects), so we can round-trip the caller's input without fetching.
	// Saves an API call and an expand=groups,applicationRoles query that
	// would be thrown away immediately.
	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{accountID})
	}

	if fieldsFlag != "" && v.Format == view.FormatJSON {
		return jtkpresent.ErrFieldsWithJSON
	}

	selected, projected, err := projection.Resolve(
		ctx,
		jtkpresent.UserDetailSpec,
		opts.IsExtended(),
		fieldsFlag,
		noFieldFetch,
		"users get",
	)
	if err != nil {
		return err
	}

	expand := ""
	if opts.IsExtended() {
		expand = api.UserExtendedExpand
	}
	user, err := client.GetUser(ctx, accountID, expand)
	if err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectUser(user, opts.ArtifactMode()))
	}

	presenter := jtkpresent.UserPresenter{}
	if projected {
		model := presenter.PresentUserDetailProjection(user)
		projection.ApplyToDetailInModel(model, selected)
		return jtkpresent.Emit(opts, model)
	}

	var model = presenter.PresentUserOneLiner(user)
	if opts.IsExtended() {
		model = presenter.PresentUserExtended(user)
	}
	return jtkpresent.Emit(opts, model)
}

func newSearchCmd(opts *root.Options) *cobra.Command {
	var maxResults int
	var nextPageToken string
	var fieldsFlag string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for users",
		Long: `Search for users by name, email, or username.

The search is case-insensitive and matches against display name, email
address, and other user attributes. Pagination uses /user/search's offset
model — the continuation token is the next startAt as a decimal string. The
endpoint does not return an authoritative terminator, so hasMore is inferred
from the page being full (len(results) == --max).`,
		Example: `  # Search for users named "john"
  jtk users search john

  # Include timezone and locale columns
  jtk users search john --extended

  # Emit just account IDs, one per line
  jtk users search john --id

  # Project output to selected columns
  jtk users search john --fields ACCOUNT_ID,NAME

  # Fetch the second page
  jtk users search john --max 10 --next-page-token 10`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(cmd.Context(), opts, args[0], maxResults, nextPageToken, fieldsFlag)
		},
	}

	cmd.Flags().IntVar(&maxResults, "max", 10, "Maximum number of results")
	cmd.Flags().StringVar(&nextPageToken, "next-page-token", "", "Decimal startAt for the next page")
	cmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated display columns (UserListSpec headers)")

	return cmd
}

func runSearch(ctx context.Context, opts *root.Options, query string, maxResults int, nextPageToken, fieldsFlag string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	idOnly := opts.EmitIDOnly()

	startAt, err := jtkpresent.ParseStartAtToken(nextPageToken)
	if err != nil {
		return err
	}

	if !idOnly && fieldsFlag != "" && v.Format == view.FormatJSON {
		return jtkpresent.ErrFieldsWithJSON
	}

	var selected []projection.ColumnSpec
	var projected bool
	if !idOnly {
		selected, projected, err = projection.Resolve(
			ctx,
			jtkpresent.UserListSpec,
			opts.IsExtended(),
			fieldsFlag,
			noFieldFetch,
			"users search",
		)
		if err != nil {
			return err
		}
	}

	users, err := client.SearchUsers(ctx, query, startAt, maxResults)
	if err != nil {
		return err
	}

	// /user/search has no native isLast; the heuristic is that a full page
	// implies more pages may exist. Over-reporting in the last window is the
	// documented tradeoff for a command whose endpoint lacks an authoritative
	// terminator. When maxResults <= 0 (no cap), hasMore stays false.
	hasMore := maxResults > 0 && len(users) == maxResults
	nextToken := ""
	if hasMore {
		nextToken = strconv.Itoa(startAt + len(users))
	}

	if idOnly {
		ids := make([]string, len(users))
		for i, u := range users {
			ids[i] = u.AccountID
		}
		return jtkpresent.EmitIDsWithPaginationToken(opts, ids, hasMore, nextToken)
	}

	if len(users) == 0 {
		return jtkpresent.Emit(opts, jtkpresent.UserPresenter{}.PresentEmpty(query))
	}

	if v.Format == view.FormatJSON {
		arts := jtkartifact.ProjectUsers(users, opts.ArtifactMode())
		return v.RenderArtifactList(artifact.NewListResult(arts, hasMore))
	}

	presenter := jtkpresent.UserPresenter{}
	model := presenter.PresentUserList(users, opts.IsExtended())
	if projected {
		projection.ApplyToTableInModel(model, selected)
	}
	model.Sections = jtkpresent.AppendPaginationHintWithToken(model.Sections, hasMore, nextToken)
	return jtkpresent.Emit(opts, model)
}
