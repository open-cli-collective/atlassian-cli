package fields

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	sharederrors "github.com/open-cli-collective/atlassian-go/errors"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

func newShowCmd(opts *root.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "show <field-id>",
		Short: "Show field contexts and options",
		Long:  "Show a flat denormalized view of a field's contexts, project mappings, and options.",
		Example: `  # Show contexts and options for a custom field
  jtk fields show customfield_10100

  # Emit only context IDs
  jtk fields show customfield_10100 --id`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(cmd.Context(), opts, args[0])
		},
	}
}

func runShow(ctx context.Context, opts *root.Options, fieldID string) error {
	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	contexts, err := client.GetAllFieldContexts(ctx, fieldID)
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		seen := make(map[string]bool, len(contexts))
		var ids []string
		for _, c := range contexts {
			if !seen[c.ID] {
				seen[c.ID] = true
				ids = append(ids, c.ID)
			}
		}
		return jtkpresent.EmitIDs(opts, ids)
	}

	if len(contexts) == 0 {
		v := opts.View()
		if v.Format == view.FormatJSON {
			return v.JSON([]jtkpresent.FieldShowRow{})
		}
		return jtkpresent.Emit(opts, jtkpresent.FieldPresenter{}.PresentFieldShowEmpty(fieldID))
	}

	mappings, err := client.GetAllFieldContextProjectMappings(ctx, fieldID)
	if err != nil {
		return err
	}

	projectsByContext := buildProjectMap(mappings)

	rows, err := buildShowRows(ctx, client, fieldID, contexts, projectsByContext)
	if err != nil {
		return err
	}

	v := opts.View()
	if v.Format == view.FormatJSON {
		return v.JSON(rows)
	}

	return jtkpresent.Emit(opts, jtkpresent.FieldPresenter{}.PresentFieldShow(rows))
}

func buildProjectMap(mappings []api.FieldContextProjectMapping) map[string]string {
	grouped := make(map[string][]string)
	globals := make(map[string]bool)
	for _, m := range mappings {
		if m.IsGlobal {
			globals[m.ContextID] = true
			continue
		}
		if m.ProjectID != "" {
			grouped[m.ContextID] = append(grouped[m.ContextID], m.ProjectID)
		}
	}

	result := make(map[string]string, len(grouped)+len(globals))
	for ctxID := range globals {
		result[ctxID] = "(global)"
	}
	for ctxID, ids := range grouped {
		if _, isGlobal := result[ctxID]; isGlobal {
			continue
		}
		sort.Strings(ids)
		result[ctxID] = strings.Join(ids, ", ")
	}
	return result
}

func buildShowRows(ctx context.Context, client *api.Client, fieldID string, contexts []api.FieldContext, projectsByContext map[string]string) ([]jtkpresent.FieldShowRow, error) {
	var rows []jtkpresent.FieldShowRow
	for _, fc := range contexts {
		projects := projectsByContext[fc.ID]
		if projects == "" {
			if fc.IsGlobalContext {
				projects = "(global)"
			} else {
				projects = "-"
			}
		}

		emptyRow := jtkpresent.FieldShowRow{
			ContextID:   fc.ID,
			Context:     fc.Name,
			Projects:    projects,
			OptionID:    "-",
			OptionValue: "-",
		}

		options, err := client.GetAllFieldContextOptions(ctx, fieldID, fc.ID)
		if err != nil {
			if isOptionFetchTolerable(err) {
				rows = append(rows, emptyRow)
				continue
			}
			return nil, err
		}

		if len(options) == 0 {
			rows = append(rows, emptyRow)
			continue
		}

		for _, opt := range options {
			rows = append(rows, jtkpresent.FieldShowRow{
				ContextID:   fc.ID,
				Context:     fc.Name,
				Projects:    projects,
				OptionID:    opt.ID,
				OptionValue: opt.Value,
			})
		}
	}
	return rows, nil
}

func isOptionFetchTolerable(err error) bool {
	return errors.Is(err, sharederrors.ErrBadRequest) || errors.Is(err, sharederrors.ErrNotFound)
}
