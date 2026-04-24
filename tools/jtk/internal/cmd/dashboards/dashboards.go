// Package dashboards provides CLI commands for managing Jira dashboards.
package dashboards

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/atlassian-go/present"
	"github.com/open-cli-collective/atlassian-go/view"

	"github.com/open-cli-collective/jira-ticket-cli/api"
	"github.com/open-cli-collective/jira-ticket-cli/internal/cmd/root"
	jtkpresent "github.com/open-cli-collective/jira-ticket-cli/internal/present"
)

// Register registers the dashboards commands
func Register(parent *cobra.Command, opts *root.Options) {
	cmd := &cobra.Command{
		Use:     "dashboards",
		Aliases: []string{"dashboard", "dash"},
		Short:   "Manage dashboards",
		Long:    "Commands for listing, creating, and managing Jira dashboards and their gadgets.",
		// IsBearerAuth guards non-Agile scope-restricted APIs (Automation, Dashboard).
		// Agile API commands (boards, sprints) use SupportsAgile() instead.
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			client, err := opts.APIClient()
			if err != nil {
				return err
			}
			if client.IsBearerAuth() {
				return api.ErrDashboardUnavailable
			}
			return nil
		},
	}

	cmd.AddCommand(newListCmd(opts))
	cmd.AddCommand(newGetCmd(opts))
	cmd.AddCommand(newCreateCmd(opts))
	cmd.AddCommand(newDeleteCmd(opts))
	cmd.AddCommand(newGadgetsCmd(opts))

	parent.AddCommand(cmd)
}

func newListCmd(opts *root.Options) *cobra.Command {
	var search string
	var maxResults int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List dashboards",
		Long:  "List accessible dashboards. Use --search to filter by name.",
		Example: `  jtk dashboards list
  jtk dashboards list --search "Sprint"
  jtk dashboards list --max 10`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), opts, search, maxResults)
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "Search dashboards by name")
	cmd.Flags().IntVar(&maxResults, "max", 50, "Maximum number of results")

	return cmd
}

func runList(_ context.Context, opts *root.Options, search string, maxResults int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	var dashboards []api.Dashboard

	if search != "" {
		result, err := client.SearchDashboards(search, maxResults)
		if err != nil {
			return err
		}
		dashboards = result.Values
	} else {
		result, err := client.GetDashboards(0, maxResults)
		if err != nil {
			return err
		}
		dashboards = result.Dashboards
	}

	if len(dashboards) == 0 {
		model := jtkpresent.DashboardPresenter{}.PresentEmpty()
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	if v.Format == view.FormatJSON {
		return v.JSON(dashboards)
	}

	// Text path: presenter → render → write
	model := jtkpresent.DashboardPresenter{}.PresentList(dashboards)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

func newGetCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <dashboard-id>",
		Short:   "Get dashboard details",
		Long:    "Get details of a specific dashboard including its gadgets.",
		Example: `  jtk dashboards get 10001`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(cmd.Context(), opts, args[0])
		},
	}

	return cmd
}

func runGet(_ context.Context, opts *root.Options, dashboardID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	dash, err := client.GetDashboard(dashboardID)
	if err != nil {
		return err
	}

	// Also get gadgets
	gadgetsResp, err := client.GetDashboardGadgets(dashboardID)
	if err != nil {
		return fmt.Errorf("failed to get gadgets: %w", err)
	}

	if v.Format == view.FormatJSON {
		return v.JSON(map[string]interface{}{
			"dashboard": dash,
			"gadgets":   gadgetsResp.Gadgets,
		})
	}

	// Text path: presenter → render → write
	model := jtkpresent.DashboardPresenter{}.PresentDetail(dash, gadgetsResp.Gadgets)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

func newCreateCmd(opts *root.Options) *cobra.Command {
	var name string
	var description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new dashboard",
		Long:  "Create a new Jira dashboard.",
		Example: `  jtk dashboards create --name "My Dashboard"
  jtk dashboards create --name "Sprint Board" --description "Sprint tracking"`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCreate(opts, name, description)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Dashboard name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Dashboard description")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func runCreate(opts *root.Options, name, description string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	req := api.CreateDashboardRequest{
		Name:             name,
		Description:      description,
		EditPermissions:  []api.SharePerm{},
		SharePermissions: []api.SharePerm{},
	}

	dash, err := client.CreateDashboard(req)
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{dash.ID})
	}

	if v.Format == view.FormatJSON {
		return v.JSON(dash)
	}

	return jtkpresent.Emit(opts, jtkpresent.DashboardPresenter{}.PresentList([]api.Dashboard{*dash}))
}

func newDeleteCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <dashboard-id>",
		Short:   "Delete a dashboard",
		Long:    "Delete a Jira dashboard by its ID.",
		Example: `  jtk dashboards delete 10001`,
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runDelete(opts, args[0])
		},
	}

	return cmd
}

func runDelete(opts *root.Options, dashboardID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.DeleteDashboard(dashboardID); err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.JSON(map[string]string{"status": "deleted", "dashboardId": dashboardID})
	}

	model := jtkpresent.DashboardPresenter{}.PresentDeleted(dashboardID)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
}

func newGadgetsCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gadgets",
		Short: "Manage dashboard gadgets",
		Long:  "Commands for listing, adding, and removing gadgets on dashboards.",
	}

	cmd.AddCommand(newGadgetsListCmd(opts))
	cmd.AddCommand(newGadgetsAddCmd(opts))
	cmd.AddCommand(newGadgetsRemoveCmd(opts))

	return cmd
}

func newGadgetsListCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <dashboard-id>",
		Short:   "List gadgets on a dashboard",
		Long:    "List all gadgets on a specific dashboard.",
		Example: `  jtk dashboards gadgets list 10001`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGadgetsList(cmd.Context(), opts, args[0])
		},
	}

	return cmd
}

func runGadgetsList(_ context.Context, opts *root.Options, dashboardID string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	result, err := client.GetDashboardGadgets(dashboardID)
	if err != nil {
		return err
	}

	if len(result.Gadgets) == 0 {
		model := jtkpresent.DashboardPresenter{}.PresentNoGadgets(dashboardID)
		out := present.Render(model, opts.RenderStyle())
		_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
		return nil
	}

	if v.Format == view.FormatJSON {
		return v.JSON(result.Gadgets)
	}

	// Text path: presenter → render → write
	model := jtkpresent.DashboardPresenter{}.PresentGadgets(result.Gadgets)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	_, _ = fmt.Fprint(opts.Stderr, out.Stderr)
	return nil
}

func newGadgetsAddCmd(opts *root.Options) *cobra.Command {
	var moduleKey, title, color, position string

	cmd := &cobra.Command{
		Use:   "add <dashboard-id>",
		Short: "Add a gadget to a dashboard",
		Long:  "Add a gadget to a dashboard by its module key.",
		Example: `  # Add a sprint burndown gadget
  jtk dashboards gadgets add 10001 --type com.atlassian.jira.gadgets:sprint-burndown-gadget

  # Add with position and title
  jtk dashboards gadgets add 10001 --type com.atlassian.jira.gadgets:filter-results-gadget --position 1,0 --title "My Filter"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGadgetsAdd(cmd.Context(), opts, args[0], moduleKey, title, color, position)
		},
	}

	cmd.Flags().StringVarP(&moduleKey, "type", "t", "", "Gadget module key (required)")
	cmd.Flags().StringVar(&title, "title", "", "Gadget title")
	cmd.Flags().StringVar(&color, "color", "", "Gadget color")
	cmd.Flags().StringVarP(&position, "position", "p", "", "Position as row,column (e.g. 1,0)")

	_ = cmd.MarkFlagRequired("type")

	return cmd
}

func runGadgetsAdd(_ context.Context, opts *root.Options, dashboardID, moduleKey, title, color, positionStr string) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	req := api.AddDashboardGadgetRequest{
		ModuleKey: moduleKey,
		Title:     title,
		Color:     color,
	}

	if positionStr != "" {
		parts := strings.SplitN(positionStr, ",", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid position %q: expected row,column (e.g. 1,0)", positionStr)
		}
		row, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return fmt.Errorf("invalid position row %q: %w", parts[0], err)
		}
		col, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return fmt.Errorf("invalid position column %q: %w", parts[1], err)
		}
		if row < 0 || col < 0 {
			return fmt.Errorf("invalid position %q: row and column must be non-negative", positionStr)
		}
		req.Position = &api.DashboardGadgetPos{Row: row, Column: col}
	}

	gadget, err := client.AddDashboardGadget(dashboardID, req)
	if err != nil {
		return err
	}

	if opts.EmitIDOnly() {
		return jtkpresent.EmitIDs(opts, []string{strconv.Itoa(gadget.ID)})
	}

	if v.Format == view.FormatJSON {
		return v.JSON(gadget)
	}

	return jtkpresent.Emit(opts, jtkpresent.DashboardPresenter{}.PresentGadgets([]api.DashboardGadget{*gadget}))
}

func newGadgetsRemoveCmd(opts *root.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <dashboard-id> <gadget-id>",
		Short:   "Remove a gadget from a dashboard",
		Long:    "Remove a gadget from a dashboard by its ID.",
		Example: `  jtk dashboards gadgets remove 10001 42`,
		Args:    cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			gadgetID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid gadget ID: %s", args[1])
			}
			return runGadgetsRemove(opts, args[0], gadgetID)
		},
	}

	return cmd
}

func runGadgetsRemove(opts *root.Options, dashboardID string, gadgetID int) error {
	v := opts.View()

	client, err := opts.APIClient()
	if err != nil {
		return err
	}

	if err := client.RemoveDashboardGadget(dashboardID, gadgetID); err != nil {
		return err
	}

	if v.Format == view.FormatJSON {
		return v.JSON(map[string]interface{}{
			"status":      "removed",
			"dashboardId": dashboardID,
			"gadgetId":    gadgetID,
		})
	}

	model := jtkpresent.DashboardPresenter{}.PresentGadgetRemoved(gadgetID, dashboardID)
	out := present.Render(model, opts.RenderStyle())
	_, _ = fmt.Fprint(opts.Stdout, out.Stdout)
	return nil
}
