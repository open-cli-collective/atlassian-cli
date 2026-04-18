// Package users provides CLI commands for searching Jira users.
package users

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/artifact"
	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	jtkartifact "github.com/open-cli-collective/jira-ticket-cli/internal/artifact"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
	"github.com/open-cli-collective/jira-ticket-cli/internal/present/projection"
)

// errFieldsWithJSON is returned when --fields is combined with --output json.
// JSON artifacts are not projected; mixing them silently would hide the flag.
var errFieldsWithJSON = errors.New("--fields is not supported with --output json")

// noFieldFetch is the projection.Resolve fetcher for user commands. Users are
// not Jira issue fields, so there is no metadata to fetch; returning nil
// routes any deferred tokens cleanly to UnknownFieldError rather than into a
// real /rest/api/3/field call that would surface unrelated issue-field names.
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

	// --id wins: collapse output before we do any projection work.
	if opts.EmitIDOnly() {
		user, err := client.GetUser(ctx, accountID)
		if err != nil {
			return err
		}
		return jtkpresent.EmitIDs(opts, []string{user.AccountID})
	}

	if fieldsFlag != "" && v.Format == view.FormatJSON {
		return errFieldsWithJSON
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

	user, err := client.GetUser(ctx, accountID)
	if err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.RenderArtifact(jtkartifact.ProjectUser(user, opts.ArtifactMode()))
	}

	presenter := jtkpresent.UserPresenter{}
	if projected {
		model := presenter.PresentUserDetailProjection(user)
		projectDetailSectionInModel(model, selected)
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

	startAt, err := parseStartAtToken(nextPageToken)
	if err != nil {
		return err
	}

	if !idOnly && fieldsFlag != "" && v.Format == view.FormatJSON {
		return errFieldsWithJSON
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
		projectTableSectionInModel(model, selected)
	}
	model.Sections = jtkpresent.AppendPaginationHintWithToken(model.Sections, hasMore, nextToken)
	return jtkpresent.Emit(opts, model)
}

// parseStartAtToken converts a `--next-page-token` value to a 0-based offset.
// The spec surfaces the token as a decimal number; non-numeric input is
// rejected up-front so we don't silently paginate past the start of the
// result set.
func parseStartAtToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(token)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid --next-page-token %q: expected a non-negative decimal", token)
	}
	return n, nil
}

// projectDetailSectionInModel rewrites the first DetailSection of model to
// the selected fields. No-op when there is no DetailSection.
func projectDetailSectionInModel(model *present.OutputModel, selected []projection.ColumnSpec) {
	for i, s := range model.Sections {
		if ds, ok := s.(*present.DetailSection); ok {
			model.Sections[i] = projection.ProjectDetail(ds, selected)
			return
		}
	}
}

// projectTableSectionInModel rewrites the first TableSection of model to the
// selected columns. No-op when there is no TableSection.
func projectTableSectionInModel(model *present.OutputModel, selected []projection.ColumnSpec) {
	for i, s := range model.Sections {
		if ts, ok := s.(*present.TableSection); ok {
			model.Sections[i] = projection.ProjectTable(ts, selected)
			return
		}
	}
}
